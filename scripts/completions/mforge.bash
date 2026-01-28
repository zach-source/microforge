# mforge bash completion with optional fzf integration.
# Source this file from your shell profile.

_mforge_home() {
  if [ -n "${MF_HOME:-}" ]; then
    printf '%s' "$MF_HOME"
  else
    printf '%s' "$HOME/.microforge"
  fi
}

_mforge_py() {
  if command -v python3 >/dev/null 2>&1; then
    python3 "$@"
  elif command -v python >/dev/null 2>&1; then
    python "$@"
  else
    return 1
  fi
}

_mforge_active_rig() {
  local path
  path="$(_mforge_home)/context.json"
  [ -f "$path" ] || return 0
  _mforge_py - "$path" <<'PY'
import json
import sys
path = sys.argv[1]
try:
    with open(path, 'r') as f:
        data = json.load(f)
    rig = (data.get('active_rig') or '').strip()
    if rig:
        print(rig)
except Exception:
    pass
PY
}

_mforge_rigs() {
  local dir
  dir="$(_mforge_home)/rigs"
  [ -d "$dir" ] || return 0
  ls -1 "$dir" 2>/dev/null
}

_mforge_cells() {
  local rig="$1"
  local dir
  if [ -z "$rig" ]; then
    rig="$(_mforge_active_rig)"
  fi
  [ -n "$rig" ] || return 0
  dir="$(_mforge_home)/rigs/$rig/cells"
  [ -d "$dir" ] || return 0
  for d in "$dir"/*; do
    [ -d "$d" ] || continue
    basename "$d"
  done
}

_mforge_scopes() {
  local rig="$1"
  local dir
  if [ -z "$rig" ]; then
    rig="$(_mforge_active_rig)"
  fi
  [ -n "$rig" ] || return 0
  dir="$(_mforge_home)/rigs/$rig/cells"
  [ -d "$dir" ] || return 0
  _mforge_py - "$dir" <<'PY'
import json
import os
import sys

dir_path = sys.argv[1]
scopes = []
for name in os.listdir(dir_path):
    cell_json = os.path.join(dir_path, name, 'cell.json')
    if not os.path.isfile(cell_json):
        continue
    try:
        with open(cell_json, 'r') as f:
            data = json.load(f)
    except Exception:
        continue
    scope = (data.get('scope_prefix') or '').strip()
    if scope:
        scopes.append(scope)
for scope in sorted(set(scopes)):
    print(scope)
PY
}

_mforge_turn_id() {
  local rig="$1"
  local path
  if [ -z "$rig" ]; then
    rig="$(_mforge_active_rig)"
  fi
  [ -n "$rig" ] || return 0
  path="$(_mforge_home)/rigs/$rig/turn.json"
  [ -f "$path" ] || return 0
  _mforge_py - "$path" <<'PY'
import json
import sys
path = sys.argv[1]
try:
    with open(path, 'r') as f:
        data = json.load(f)
    tid = (data.get('id') or '').strip()
    if tid:
        print(tid)
except Exception:
    pass
PY
}

_mforge_rig_repo() {
  local rig="$1"
  local path
  if [ -z "$rig" ]; then
    rig="$(_mforge_active_rig)"
  fi
  [ -n "$rig" ] || return 0
  path="$(_mforge_home)/rigs/$rig/rig.json"
  [ -f "$path" ] || return 0
  _mforge_py - "$path" <<'PY'
import json
import sys
path = sys.argv[1]
try:
    with open(path, 'r') as f:
        data = json.load(f)
    repo = (data.get('repo_path') or '').strip()
    if repo:
        print(repo)
except Exception:
    pass
PY
}

_mforge_bd_json() {
  local rig="$1"
  local repo
  repo="$(_mforge_rig_repo "$rig")"
  [ -n "$repo" ] || return 0
  command -v bd >/dev/null 2>&1 || return 0
  (cd "$repo" && bd list --json 2>/dev/null)
}

_mforge_bead_ids() {
  local rig="$1"
  local type="$2"
  _mforge_bd_json "$rig" | _mforge_py - "$type" <<'PY'
import json
import sys
raw = sys.stdin.read().strip()
issue_type = sys.argv[1].strip().lower() if len(sys.argv) > 1 else ''
if not raw:
    sys.exit(0)
try:
    data = json.loads(raw)
except Exception:
    sys.exit(0)
if isinstance(data, dict):
    data = data.get('issues') or [data]
if not isinstance(data, list):
    sys.exit(0)
for item in data:
    if not isinstance(item, dict):
        continue
    t = (item.get('type') or item.get('issue_type') or '').strip().lower()
    if issue_type and t != issue_type:
        continue
    bid = (item.get('id') or '').strip()
    if bid:
        print(bid)
PY
}

_mforge_bead_pairs() {
  local rig="$1"
  local type="$2"
  _mforge_bd_json "$rig" | _mforge_py - "$type" <<'PY'
import json
import re
import sys
raw = sys.stdin.read().strip()
issue_type = sys.argv[1].strip().lower() if len(sys.argv) > 1 else ''
if not raw:
    sys.exit(0)
try:
    data = json.loads(raw)
except Exception:
    sys.exit(0)
if isinstance(data, dict):
    data = data.get('issues') or [data]
if not isinstance(data, list):
    sys.exit(0)
for item in data:
    if not isinstance(item, dict):
        continue
    t = (item.get('type') or item.get('issue_type') or '').strip().lower()
    if issue_type and t != issue_type:
        continue
    bid = (item.get('id') or '').strip()
    if not bid:
        continue
    desc = (item.get('description') or item.get('title') or '').strip()
    desc = re.sub(r'[\r\n\t]+', ' ', desc)
    if len(desc) > 120:
        desc = desc[:120] + "..."
    print(f"{bid}\t{desc}")
PY
}

_mforge_bead_types() {
  local rig="$1"
  _mforge_bd_json "$rig" | _mforge_py - <<'PY'
import json
import sys
raw = sys.stdin.read().strip()
if not raw:
    sys.exit(0)
try:
    data = json.loads(raw)
except Exception:
    sys.exit(0)
if isinstance(data, dict):
    data = data.get('issues') or [data]
if not isinstance(data, list):
    sys.exit(0)
seen = set()
for item in data:
    if not isinstance(item, dict):
        continue
    t = (item.get('type') or item.get('issue_type') or '').strip()
    if t and t not in seen:
        seen.add(t)
for t in sorted(seen):
    print(t)
PY
}

_mforge_agent_specs() {
  local repo
  repo="$(_mforge_rig_repo)"
  [ -n "$repo" ] || return 0
  local dir="$repo/.mf/agents"
  [ -d "$dir" ] || return 0
  local f
  for f in "$dir"/*.json; do
    [ -f "$f" ] || continue
    basename "$f" .json
  done
}

_mforge_should_fzf() {
  [ -z "${MF_FZF_DISABLE:-}" ] && command -v fzf >/dev/null 2>&1
}

_mforge_complete_from_list() {
  local cur="$1"
  shift
  if [ "$#" -eq 0 ]; then
    COMPREPLY=()
    return
  fi
  if _mforge_should_fzf; then
    local selected
    selected=$(printf '%s\n' "$@" | fzf --height 40% --reverse --prompt='mforge> ' --query "$cur" 2>/dev/null)
    COMPREPLY=()
    if [ -n "$selected" ]; then
      COMPREPLY=("$selected")
    fi
    return
  fi
  COMPREPLY=( $(compgen -W "$(printf '%s ' "$@")" -- "$cur") )
}

_mforge_complete_from_pairs() {
  local cur="$1"
  shift
  if [ "$#" -eq 0 ]; then
    COMPREPLY=()
    return
  fi
  if _mforge_should_fzf; then
    local selected
    selected=$(printf '%s\n' "$@" | fzf --height 40% --reverse --prompt='mforge> ' --query "$cur" --delimiter=$'\t' --with-nth=1,2 2>/dev/null)
    selected="${selected%%$'\t'*}"
    COMPREPLY=()
    if [ -n "$selected" ]; then
      COMPREPLY=("$selected")
    fi
    return
  fi
  local ids=()
  local item
  for item in "$@"; do
    ids+=("${item%%$'\t'*}")
  done
  COMPREPLY=( $(compgen -W "$(printf '%s ' "${ids[@]}")" -- "$cur") )
}

_mforge_complete() {
  local cur prev cmd sub
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  cmd="${COMP_WORDS[1]}"
  sub="${COMP_WORDS[2]}"

  if [ $COMP_CWORD -eq 1 ]; then
    _mforge_complete_from_list "$cur" init cell agent task request monitor epic manager turn round checkpoint bead review pr merge wait coordinator digest build deploy contract architect report library scope engine convoy watch quick-assign tui migrate context rig ssh completions hook help
    return
  fi

  if [ "$cmd" = "help" ]; then
    _mforge_complete_from_list "$cur" init cell agent task request monitor epic manager turn round checkpoint bead review pr merge wait coordinator digest build deploy contract architect report library scope engine convoy watch quick-assign tui migrate context rig ssh completions hook
    return
  fi

  case "$cmd" in
    init)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" $(_mforge_rigs)
      else
        _mforge_complete_from_list "$cur" --repo
      fi
      return
      ;;
    cell)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" add bootstrap agent-file
        return
      fi
      case "$sub" in
        add)
          if [ $COMP_CWORD -eq 3 ]; then
            return
          fi
          if [ "$prev" = "--scope" ]; then
            _mforge_complete_from_list "$cur" $(_mforge_scopes)
            return
          fi
          _mforge_complete_from_list "$cur" --scope
          return
          ;;
        bootstrap)
          if [ $COMP_CWORD -eq 3 ]; then
            _mforge_complete_from_list "$cur" $(_mforge_cells)
            return
          fi
          _mforge_complete_from_list "$cur" --architect --single
          return
          ;;
        agent-file)
          if [ $COMP_CWORD -eq 3 ]; then
            _mforge_complete_from_list "$cur" $(_mforge_cells)
            return
          fi
          if [ "$prev" = "--role" ]; then
            _mforge_complete_from_list "$cur" builder monitor reviewer architect cell
            return
          fi
          _mforge_complete_from_list "$cur" --role
          return
          ;;
      esac
      ;;
    agent)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" spawn stop attach wake relaunch send status logs heartbeat create bootstrap
        return
      fi
      if [ "$sub" = "send" ]; then
        _mforge_complete_from_list "$cur" --no-enter
        return
      fi
      if [ "$sub" = "create" ]; then
        if [ "$prev" = "--description" ]; then
          return
        fi
        if [ "$prev" = "--class" ]; then
          _mforge_complete_from_list "$cur" crew worker
          return
        fi
        _mforge_complete_from_list "$cur" --description --class
        return
      fi
      if [ "$sub" = "bootstrap" ]; then
        if [ $COMP_CWORD -eq 3 ]; then
          _mforge_complete_from_list "$cur" $(_mforge_agent_specs)
          return
        fi
        return
      fi
      if [ "$sub" = "status" ]; then
        case "$prev" in
          --cell)
            _mforge_complete_from_list "$cur" $(_mforge_cells)
            return
            ;;
          --role)
            _mforge_complete_from_list "$cur" builder monitor reviewer architect cell
            return
            ;;
        esac
        _mforge_complete_from_list "$cur" --cell --role --remote --json
        return
      fi
      if [ "$sub" = "heartbeat" ]; then
        if [ $COMP_CWORD -eq 3 ]; then
          _mforge_complete_from_list "$cur" $(_mforge_cells)
          return
        fi
        if [ $COMP_CWORD -eq 4 ]; then
          _mforge_complete_from_list "$cur" builder monitor reviewer architect cell
          return
        fi
        return
      fi
      if [ "$sub" = "logs" ]; then
        if [ $COMP_CWORD -eq 3 ]; then
          _mforge_complete_from_list "$cur" $(_mforge_cells)
          return
        fi
        if [ $COMP_CWORD -eq 4 ]; then
          _mforge_complete_from_list "$cur" builder monitor reviewer architect cell
          return
        fi
        _mforge_complete_from_list "$cur" --follow --lines --all
        return
      fi
      if [ $COMP_CWORD -eq 3 ]; then
        _mforge_complete_from_list "$cur" $(_mforge_cells)
        return
      fi
      if [ $COMP_CWORD -eq 4 ]; then
        _mforge_complete_from_list "$cur" builder monitor reviewer architect cell
        return
      fi
      return
      ;;
    task)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" create update list split decompose complete delete
        return
      fi
      case "$sub" in
        create)
          case "$prev" in
            --scope)
              _mforge_complete_from_list "$cur" $(_mforge_scopes)
              return
              ;;
            --kind)
              _mforge_complete_from_list "$cur" improve fix review monitor doc
              return
              ;;
            --title|--body)
              return
              ;;
          esac
          _mforge_complete_from_list "$cur" --title --body --scope --kind
          return
          ;;
        update)
          case "$prev" in
            --task)
              _mforge_complete_from_pairs "$cur" $(_mforge_bead_pairs "" task)
              return
              ;;
            --scope)
              _mforge_complete_from_list "$cur" $(_mforge_scopes)
              return
              ;;
          esac
          _mforge_complete_from_list "$cur" --task --scope
          return
          ;;
        list)
          return
          ;;
        split)
          case "$prev" in
            --task)
              _mforge_complete_from_pairs "$cur" $(_mforge_bead_pairs "" task)
              return
              ;;
            --cells)
              _mforge_complete_from_list "$cur" $(_mforge_cells)
              return
              ;;
          esac
          _mforge_complete_from_list "$cur" --task --cells
          return
          ;;
        decompose)
          case "$prev" in
            --task)
              _mforge_complete_from_pairs "$cur" $(_mforge_bead_pairs "" task)
              return
              ;;
            --titles|--kind)
              return
              ;;
          esac
          _mforge_complete_from_list "$cur" --task --titles --kind
          return
          ;;
        complete)
          case "$prev" in
            --task)
              _mforge_complete_from_pairs "$cur" $(_mforge_bead_pairs "" task)
              return
              ;;
            --reason)
              return
              ;;
          esac
          _mforge_complete_from_list "$cur" --task --reason --force
          return
          ;;
        delete)
          case "$prev" in
            --task)
              _mforge_complete_from_pairs "$cur" $(_mforge_bead_pairs "" task)
              return
              ;;
            --reason)
              return
              ;;
          esac
          _mforge_complete_from_list "$cur" --task --reason --force --cascade --hard --dry-run
          return
          ;;
      esac
      ;;
    assign)
      case "$prev" in
        --task)
          _mforge_complete_from_pairs "$cur" $(_mforge_bead_pairs "" task)
          return
          ;;
        --cell)
          _mforge_complete_from_list "$cur" $(_mforge_cells)
          return
          ;;
        --role)
          _mforge_complete_from_list "$cur" builder monitor reviewer architect cell
          return
          ;;
        --promise)
          return
          ;;
      esac
      _mforge_complete_from_list "$cur" --task --cell --role --promise --quick
      return
      ;;
    quick-assign)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" $(_mforge_bead_ids "")
        return
      fi
      if [ $COMP_CWORD -eq 3 ]; then
        _mforge_complete_from_list "$cur" $(_mforge_cells)
        return
      fi
      if [ "$prev" = "--role" ]; then
        _mforge_complete_from_list "$cur" builder monitor reviewer architect cell
        return
      fi
      _mforge_complete_from_list "$cur" --role --promise
      return
      ;;
    request)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" create list triage
        return
      fi
      case "$sub" in
        create)
          case "$prev" in
            --cell)
              _mforge_complete_from_list "$cur" $(_mforge_cells)
              return
              ;;
            --role)
              _mforge_complete_from_list "$cur" builder monitor reviewer architect cell
              return
              ;;
            --severity)
              _mforge_complete_from_list "$cur" low med high
              return
              ;;
            --priority)
              _mforge_complete_from_list "$cur" p1 p2 p3
              return
              ;;
            --scope)
              _mforge_complete_from_list "$cur" $(_mforge_scopes)
              return
              ;;
            --payload)
              return
              ;;
          esac
          _mforge_complete_from_list "$cur" --cell --role --severity --priority --scope --payload
          return
          ;;
        list)
          case "$prev" in
            --cell)
              _mforge_complete_from_list "$cur" $(_mforge_cells)
              return
              ;;
            --status)
              _mforge_complete_from_list "$cur" open in_progress blocked done closed
              return
              ;;
            --priority)
              _mforge_complete_from_list "$cur" p1 p2 p3
              return
              ;;
          esac
          _mforge_complete_from_list "$cur" --cell --status --priority
          return
          ;;
        triage)
          case "$prev" in
            --request)
              _mforge_complete_from_list "$cur" $(_mforge_bead_ids "" request)
              return
              ;;
            --action)
              _mforge_complete_from_list "$cur" create-task merge block
              return
              ;;
          esac
          _mforge_complete_from_list "$cur" --request --action
          return
          ;;
      esac
      ;;
    scope)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" list show
        return
      fi
      case "$sub" in
        list)
          return
          ;;
        show)
          case "$prev" in
            --scope)
              _mforge_complete_from_list "$cur" $(_mforge_scopes)
              return
              ;;
          esac
          _mforge_complete_from_list "$cur" --scope
          return
          ;;
      esac
      ;;
    context)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" get set unset list
        return
      fi
      if [ "$sub" = "set" ] && [ $COMP_CWORD -eq 3 ]; then
        _mforge_complete_from_list "$cur" $(_mforge_rigs)
        return
      fi
      return
      ;;
    rig)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" list delete rename backup restore message
        return
      fi
      case "$sub" in
        list)
          return
          ;;
        delete|backup)
          if [ $COMP_CWORD -eq 3 ]; then
            _mforge_complete_from_list "$cur" $(_mforge_rigs)
            return
          fi
          return
          ;;
        rename)
          if [ $COMP_CWORD -eq 3 ]; then
            _mforge_complete_from_list "$cur" $(_mforge_rigs)
            return
          fi
          return
          ;;
        restore)
          _mforge_complete_from_list "$cur" --name --force
          return
          ;;
        message)
          case "$prev" in
            --cell)
              _mforge_complete_from_list "$cur" $(_mforge_cells "$rig")
              return
              ;;
            --role)
              _mforge_complete_from_list "$cur" builder monitor reviewer architect cell
              return
              ;;
            --text)
              return
              ;;
          esac
          _mforge_complete_from_list "$cur" --cell --role --text
          return
          ;;
      esac
      ;;
    engine)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" run emit drain
        return
      fi
      case "$sub" in
        run)
          _mforge_complete_from_list "$cur" --wait --rounds --completion-promise
          return
          ;;
        emit)
          case "$prev" in
            --scope)
              _mforge_complete_from_list "$cur" $(_mforge_scopes)
              return
              ;;
            --type|--title|--source|--payload)
              return
              ;;
          esac
          _mforge_complete_from_list "$cur" --type --scope --title --source --payload
          return
          ;;
        drain)
      _mforge_complete_from_list "$cur" --keep
      return
      ;;
    convoy)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" start
        return
      fi
      case "$prev" in
        --epic)
          _mforge_complete_from_list "$cur" $(_mforge_bead_ids "" epic)
          return
          ;;
        --role)
          _mforge_complete_from_list "$cur" builder monitor reviewer architect cell
          return
          ;;
        --title)
          return
          ;;
      esac
      _mforge_complete_from_list "$cur" --epic --role --title
      return
      ;;
      esac
      ;;
    monitor)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" run-tests run
        return
      fi
      if [ $COMP_CWORD -eq 3 ]; then
        _mforge_complete_from_list "$cur" $(_mforge_cells)
        return
      fi
      case "$prev" in
        --severity)
          _mforge_complete_from_list "$cur" low med high
          return
          ;;
        --priority)
          _mforge_complete_from_list "$cur" p1 p2 p3
          return
          ;;
        --scope)
          _mforge_complete_from_list "$cur" $(_mforge_scopes)
          return
          ;;
        --observation)
          return
          ;;
        --cmd)
          return
          ;;
      esac
      _mforge_complete_from_list "$cur" --cmd --severity --priority --scope --observation
      return
      ;;
    epic)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" create add-task assign status close conflict design tree
        return
      fi
      case "$sub" in
        create)
          case "$prev" in
            --title|--body|--short-id)
              return
              ;;
          esac
          _mforge_complete_from_list "$cur" --title --body --short-id
          return
          ;;
        add-task)
          case "$prev" in
            --epic)
              _mforge_complete_from_list "$cur" $(_mforge_bead_ids "" epic)
              return
              ;;
            --task)
              _mforge_complete_from_pairs "$cur" $(_mforge_bead_pairs "" task)
              return
              ;;
          esac
          _mforge_complete_from_list "$cur" --epic --task
          return
          ;;
        assign)
          case "$prev" in
            --epic)
              _mforge_complete_from_list "$cur" $(_mforge_bead_ids "" epic)
              return
              ;;
            --role)
              _mforge_complete_from_list "$cur" builder monitor reviewer architect cell
              return
              ;;
          esac
          _mforge_complete_from_list "$cur" --epic --role
          return
          ;;
        status|close)
          if [ "$prev" = "--epic" ]; then
            _mforge_complete_from_list "$cur" $(_mforge_bead_ids "" epic)
            return
          fi
          _mforge_complete_from_list "$cur" --epic
          return
          ;;
        conflict)
          case "$prev" in
            --epic)
              _mforge_complete_from_list "$cur" $(_mforge_bead_ids "" epic)
              return
              ;;
            --cell)
              _mforge_complete_from_list "$cur" $(_mforge_cells)
              return
              ;;
            --details)
              return
              ;;
          esac
          _mforge_complete_from_list "$cur" --epic --cell --details
          return
          ;;
        design|tree)
          if [ $COMP_CWORD -eq 3 ]; then
            _mforge_complete_from_pairs "$cur" $(_mforge_bead_pairs "" epic)
            return
          fi
          return
          ;;
      esac
      ;;
    architect)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" docs contract design
        return
      fi
      case "$prev" in
        --cell)
          _mforge_complete_from_list "$cur" $(_mforge_cells)
          return
          ;;
        --scope)
          _mforge_complete_from_list "$cur" $(_mforge_scopes)
          return
          ;;
        --details)
          return
          ;;
      esac
      _mforge_complete_from_list "$cur" --cell --details --scope
      return
      ;;
    report)
      if [ "$prev" = "--cell" ]; then
        _mforge_complete_from_list "$cur" $(_mforge_cells)
        return
      fi
      _mforge_complete_from_list "$cur" --cell
      return
      ;;
    library)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" start query
        return
      fi
      case "$sub" in
        start)
          _mforge_complete_from_list "$cur" --addr
          return
          ;;
        query)
          case "$prev" in
            --service|--q|--addr)
              return
              ;;
          esac
          _mforge_complete_from_list "$cur" --q --service --addr
          return
          ;;
      esac
      ;;
    watch)
      _mforge_complete_from_list "$cur" --interval --role --fswatch --tui
      return
      ;;
    tui)
      _mforge_complete_from_list "$cur" --interval --remote --watch --role
      return
      ;;
    migrate)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" beads rig
        return
      fi
      _mforge_complete_from_list "$cur" --all
      return
      ;;
    completions)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" install path bash zsh
        return
      fi
      return
      ;;
    ssh)
      case "$prev" in
        --cmd)
          return
          ;;
      esac
      _mforge_complete_from_list "$cur" --cmd --tty
      return
      ;;
    manager)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" tick assign
        return
      fi
      if [ "$prev" = "--role" ]; then
        _mforge_complete_from_list "$cur" builder monitor reviewer architect cell
        return
      fi
      _mforge_complete_from_list "$cur" --watch --role
      return
      ;;
    turn)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" start status end slate run list diff
        return
      fi
      if [ "$sub" = "start" ]; then
        _mforge_complete_from_list "$cur" --name
        return
      fi
      if [ "$sub" = "end" ]; then
        _mforge_complete_from_list "$cur" --report
        return
      fi
      if [ "$sub" = "diff" ]; then
        _mforge_complete_from_list "$cur" --id
        return
      fi
      if [ "$sub" = "run" ]; then
        if [ "$prev" = "--role" ]; then
          _mforge_complete_from_list "$cur" builder monitor reviewer architect cell
          return
        fi
        _mforge_complete_from_list "$cur" --role --wait
        return
      fi
      return
      ;;
    round)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" start review merge
        return
      fi
      if [ "$sub" = "review" ]; then
        _mforge_complete_from_list "$cur" --wait --all --changes-only --base
        return
      fi
      if [ "$sub" = "merge" ]; then
        _mforge_complete_from_list "$cur" --feature --base
        return
      fi
      _mforge_complete_from_list "$cur" --wait
      return
      ;;
    checkpoint)
      _mforge_complete_from_list "$cur" --message
      return
      ;;
    round)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" start review merge
        return
      fi
      if [ "$sub" = "review" ]; then
        _mforge_complete_from_list "$cur" --wait --all --changes-only --base
        return
      fi
      if [ "$sub" = "merge" ]; then
        _mforge_complete_from_list "$cur" --feature --base
        return
      fi
      _mforge_complete_from_list "$cur" --wait
      return
      ;;
    checkpoint)
      _mforge_complete_from_list "$cur" --message
      return
      ;;
    bead)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" create list show close status triage dep template
        return
      fi
      if [ "$sub" = "dep" ]; then
        if [ $COMP_CWORD -eq 3 ]; then
          _mforge_complete_from_list "$cur" add
          return
        fi
        if [ $COMP_CWORD -eq 4 ]; then
          _mforge_complete_from_list "$cur" $(_mforge_bead_ids "")
          return
        fi
        return
      fi
      case "$prev" in
        --type)
          local types
          types=$(_mforge_bead_types "")
          if [ -n "$types" ]; then
            _mforge_complete_from_list "$cur" $types
          else
            _mforge_complete_from_list "$cur" task request observation assignment pr review epic contract decision build deploy conflictresolution
          fi
          return
          ;;
        --status)
          _mforge_complete_from_list "$cur" open in_progress blocked done closed ready queued
          return
          ;;
        --priority)
          _mforge_complete_from_list "$cur" p1 p2 p3
          return
          ;;
        --cell)
          _mforge_complete_from_list "$cur" $(_mforge_cells)
          return
          ;;
        --role)
          _mforge_complete_from_list "$cur" builder monitor reviewer architect cell
          return
          ;;
        --scope)
          _mforge_complete_from_list "$cur" $(_mforge_scopes)
          return
          ;;
        --turn)
          _mforge_complete_from_list "$cur" $(_mforge_turn_id)
          return
          ;;
        --severity)
          _mforge_complete_from_list "$cur" low med high
          return
          ;;
        --deps|--description|--acceptance|--compat|--links|--title)
          return
          ;;
      esac
      if [ "$sub" = "show" ] || [ "$sub" = "close" ] || [ "$sub" = "status" ]; then
        if [ $COMP_CWORD -eq 3 ]; then
          _mforge_complete_from_list "$cur" $(_mforge_bead_ids "")
          return
        fi
      fi
      if [ "$sub" = "status" ]; then
        if [ $COMP_CWORD -eq 4 ]; then
          _mforge_complete_from_list "$cur" open in_progress blocked done closed ready queued
          return
        fi
      fi
      if [ "$sub" = "triage" ]; then
        case "$prev" in
          --id)
            _mforge_complete_from_list "$cur" $(_mforge_bead_ids "")
            return
            ;;
          --cell)
            _mforge_complete_from_list "$cur" $(_mforge_cells)
            return
            ;;
          --role)
            _mforge_complete_from_list "$cur" builder monitor reviewer architect cell
            return
            ;;
          --turn)
            _mforge_complete_from_list "$cur" $(_mforge_turn_id)
            return
            ;;
        esac
      fi
      return
      ;;
    review)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" create
        return
      fi
      case "$prev" in
        --cell)
          _mforge_complete_from_list "$cur" $(_mforge_cells)
          return
          ;;
        --scope)
          _mforge_complete_from_list "$cur" $(_mforge_scopes)
          return
          ;;
        --title)
          return
          ;;
      esac
      _mforge_complete_from_list "$cur" --title --cell --scope --turn
      return
      ;;
    pr)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" create ready link-review
        return
      fi
      case "$sub" in
        create)
          case "$prev" in
            --cell)
              _mforge_complete_from_list "$cur" $(_mforge_cells)
              return
              ;;
            --title|--url)
              return
              ;;
          esac
          _mforge_complete_from_list "$cur" --title --cell --url --turn --status
          return
          ;;
        ready)
          if [ $COMP_CWORD -eq 3 ]; then
            _mforge_complete_from_list "$cur" $(_mforge_bead_ids "" pr)
            return
          fi
          return
          ;;
        link-review)
          if [ $COMP_CWORD -eq 3 ]; then
            _mforge_complete_from_list "$cur" $(_mforge_bead_ids "" pr)
            return
          fi
          if [ $COMP_CWORD -eq 4 ]; then
            _mforge_complete_from_list "$cur" $(_mforge_bead_ids "" review)
            return
          fi
          return
          ;;
      esac
      ;;
    merge)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" run
        return
      fi
      case "$prev" in
        --turn)
          _mforge_complete_from_list "$cur" $(_mforge_turn_id)
          return
          ;;
        --as)
          _mforge_complete_from_list "$cur" merge-manager
          return
          ;;
      esac
      _mforge_complete_from_list "$cur" --turn --as --dry-run
      return
      ;;
    wait)
      case "$prev" in
        --turn)
          _mforge_complete_from_list "$cur" $(_mforge_turn_id)
          return
          ;;
        --interval)
          return
          ;;
      esac
      _mforge_complete_from_list "$cur" --turn --interval
      return
      ;;
    coordinator)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" sync
        return
      fi
      if [ "$prev" = "--turn" ]; then
        _mforge_complete_from_list "$cur" $(_mforge_turn_id)
        return
      fi
      _mforge_complete_from_list "$cur" --turn
      return
      ;;
    digest)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" render
        return
      fi
      if [ "$prev" = "--turn" ]; then
        _mforge_complete_from_list "$cur" $(_mforge_turn_id)
        return
      fi
      _mforge_complete_from_list "$cur" --turn
      return
      ;;
    build)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" record
        return
      fi
      _mforge_complete_from_list "$cur" --service --image --status --turn
      return
      ;;
    deploy)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" record
        return
      fi
      _mforge_complete_from_list "$cur" --env --service --status --turn
      return
      ;;
    contract)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" create
        return
      fi
      case "$prev" in
        --cell)
          _mforge_complete_from_list "$cur" $(_mforge_cells)
          return
          ;;
        --scope)
          _mforge_complete_from_list "$cur" $(_mforge_scopes)
          return
          ;;
        --title|--acceptance|--compat|--links)
          return
          ;;
      esac
      _mforge_complete_from_list "$cur" --title --cell --scope --acceptance --compat --links
      return
      ;;
    build|deploy)
      return
      ;;
    hook)
      if [ $COMP_CWORD -eq 2 ]; then
        _mforge_complete_from_list "$cur" stop guardrails emit
        return
      fi
      if [ "$sub" = "emit" ]; then
        _mforge_complete_from_list "$cur" --event
        return
      fi
      return
      ;;
  esac
}

complete -F _mforge_complete mforge
