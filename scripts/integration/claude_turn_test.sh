#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MF_HOME_DIR="$(mktemp -d "/tmp/mforge-claude-home.XXXXXX")"
REPO_DIR="$(mktemp -d "/tmp/mforge-claude-repo.XXXXXX")"
RIG_NAME="claude-integration-$(date +%s)"
CELL_NAME="operator"
ROLE="builder"
SUCCESS=0

cleanup() {
  if [[ "${SUCCESS}" == "1" ]]; then
    echo "Cleaning up temp dirs: ${MF_HOME_DIR} ${REPO_DIR}"
    rm -rf "${MF_HOME_DIR}" "${REPO_DIR}" || true
  else
    echo "Test failed; keeping temp dirs for inspection:"
    echo "  MF_HOME=${MF_HOME_DIR}"
    echo "  REPO_DIR=${REPO_DIR}"
  fi
}
trap cleanup EXIT

export MF_HOME="${MF_HOME_DIR}"
export REPO_DIR

cd "${REPO_DIR}"

echo "Initializing temp repo at ${REPO_DIR}"

git init -q
cat <<'EOT' > README.md
# Integration Repo

Initial content.
EOT

git add README.md
git commit -qm "chore: init"

echo "Installing mforge from source"
(cd "${ROOT_DIR}" && go install ./cmd/mforge)

echo "Setting up rig + cell"

mforge init "${RIG_NAME}" --repo "${REPO_DIR}"
mforge context set "${RIG_NAME}"
mforge cell add "${CELL_NAME}" --scope .
mforge cell bootstrap "${CELL_NAME}"

cell_cfg="${MF_HOME_DIR}/rigs/${RIG_NAME}/cells/${CELL_NAME}/cell.json"
worktree=$(sed -n 's/.*"worktree_path"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "${cell_cfg}")
if [[ -z "${worktree}" ]]; then
  echo "Failed to read worktree path from ${cell_cfg}"
  exit 1
fi

# Verify hooks + settings are present and valid
settings_path="${worktree}/.claude/settings.json"
hooks_path="${worktree}/.mf/hooks.json"
active_path="${worktree}/.mf/active-agent.json"
if [[ ! -f "${settings_path}" ]]; then
  echo "Missing Claude settings: ${settings_path}"
  exit 1
fi
if [[ ! -f "${hooks_path}" ]]; then
  echo "Missing hooks config: ${hooks_path}"
  exit 1
fi
if [[ ! -f "${active_path}" ]]; then
  echo "Missing active agent file: ${active_path}"
  exit 1
fi
python3 - <<PY
import json, sys
path = "${settings_path}"
with open(path, "r") as fh:
    data = json.load(fh)
hooks = data.get("hooks", {})
required = {"Stop", "PreToolUse", "PermissionRequest"}
missing = required - set(hooks.keys())
if missing:
    sys.exit(f"settings.json missing hooks: {sorted(missing)}")
permissions = data.get("permissions", {})
if not isinstance(permissions, dict) or "allow" not in permissions:
    sys.exit("settings.json permissions not in expected object form")
print("Claude settings.json hooks validated")
PY
claude -p --settings "${settings_path}" "ok" >/dev/null

# Start a turn so assignments include a turn id
mforge turn start --name "claude-integration"

# Create a task that requires a small change + commit + outbox report
TASK_BODY=$(cat <<'EOT'
Append a line to README.md that says: "integration-test: <task_id>".
Commit the change with message: "test: <task_id>".
Then write the outbox report and include the completion promise token.
EOT
)

TASK_ID=$(mforge task create --title "Integration test task" --body "${TASK_BODY}" --scope . | awk '{print $3}')

if [[ -z "${TASK_ID}" ]]; then
  echo "Failed to parse task id"
  exit 1
fi

PROMISE="DONE:${TASK_ID}"
export TASK_ID

# Spawn agent session
mforge agent spawn "${CELL_NAME}" "${ROLE}"

# Assign and wake the agent
ASSIGN_LINE=$(mforge assign --task "${TASK_ID}" --cell "${CELL_NAME}" --role "${ROLE}" --promise "${PROMISE}" --quick)
echo "${ASSIGN_LINE}"
ASSIGN_ID=$(echo "${ASSIGN_LINE}" | awk -F'assignment ' '{print $2}' | tr -d ')')

if [[ -z "${ASSIGN_ID}" ]]; then
  echo "Failed to parse assignment id from: ${ASSIGN_LINE}"
  exit 1
fi

export ASSIGN_ID

INBOX_FILE="${worktree}/mail/inbox/${TASK_ID}.md"
OUTBOX_FILE="${worktree}/mail/outbox/${TASK_ID}.md"

# Wait for inbox to exist (claimed)
for i in {1..60}; do
  if [[ -f "${INBOX_FILE}" ]]; then
    break
  fi
  sleep 1
  if [[ $i -eq 60 ]]; then
    echo "Timeout waiting for inbox file ${INBOX_FILE}"
    exit 1
  fi
done

# Wait for outbox to contain promise (task completion)
for i in {1..600}; do
  if [[ -f "${OUTBOX_FILE}" ]] && grep -q "${PROMISE}" "${OUTBOX_FILE}"; then
    echo "Outbox promise detected"
    break
  fi
  sleep 2
  if [[ $i -eq 600 ]]; then
    echo "Timeout waiting for outbox promise"
    echo "Last 20 lines of agent log:"
    mforge agent logs "${CELL_NAME}" "${ROLE}" --lines 20 || true
    exit 1
  fi
done

# Run manager tick until assignment is closed
for i in {1..60}; do
  mforge manager tick
  status=$(python3 - <<PY
import json, os, subprocess
repo = os.environ["REPO_DIR"]
assign_id = os.environ["ASSIGN_ID"]
raw = subprocess.check_output(["bd", "list", "--json"], cwd=repo)
issues = json.loads(raw)
for issue in issues:
    if issue.get("id") == assign_id:
        print(issue.get("status", ""))
        break
else:
    print("missing")
PY
)
  if [[ "${status}" == "closed" || "${status}" == "done" || "${status}" == "missing" ]]; then
    echo "Assignment ${ASSIGN_ID} closed"
    break
  fi
  sleep 2
  if [[ $i -eq 60 ]]; then
    echo "Timeout waiting for assignment ${ASSIGN_ID} to close (status=${status})"
    exit 1
  fi
done

# Verify completion signal
signal_path="${worktree}/mail/signals/task-complete-${ASSIGN_ID}.json"
if [[ ! -f "${signal_path}" ]]; then
  echo "Missing completion signal: ${signal_path}"
  exit 1
fi

# Show final status
mforge turn status
mforge agent status --json

SUCCESS=1
echo "Integration test completed successfully."
