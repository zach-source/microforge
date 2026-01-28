# mforge zsh completion with optional fzf integration.
# Source this file from your shell profile.

_mforge_home() {
  if [[ -n "${MF_HOME:-}" ]]; then
    printf '%s' "$MF_HOME"
  else
    printf '%s' "$HOME/.microforge"
  fi
}

_mforge_jq() {
  command -v jq >/dev/null 2>&1 || return 1
  jq -r "$@"
}

_mforge_active_rig() {
  local json_path
  json_path="$(_mforge_home)/context.json"
  [[ -f "$json_path" ]] || return 0
  _mforge_jq '.active_rig // empty' "$json_path" 2>/dev/null
}

_mforge_rigs() {
  local dir
  dir="$(_mforge_home)/rigs"
  [[ -d "$dir" ]] || return 0
  ls -1 "$dir" 2>/dev/null
}

_mforge_cells() {
  local rig="$1"
  local dir
  if [[ -z "$rig" ]]; then
    rig="$(_mforge_active_rig)"
  fi
  [[ -n "$rig" ]] || return 0
  dir="$(_mforge_home)/rigs/$rig/cells"
  [[ -d "$dir" ]] || return 0
  for d in "$dir"/*; do
    [[ -d "$d" ]] || continue
    basename "$d"
  done
}

_mforge_scopes() {
  local rig="$1"
  local dir
  if [[ -z "$rig" ]]; then
    rig="$(_mforge_active_rig)"
  fi
  [[ -n "$rig" ]] || return 0
  dir="$(_mforge_home)/rigs/$rig/cells"
  [[ -d "$dir" ]] || return 0
  command -v jq >/dev/null 2>&1 || return 0
  local cell_json
  for cell_json in "$dir"/*/cell.json; do
    [[ -f "$cell_json" ]] || continue
    _mforge_jq '.scope_prefix // empty' "$cell_json" 2>/dev/null
  done | sort -u
}

_mforge_turn_id() {
  local rig="$1"
  local json_path
  if [[ -z "$rig" ]]; then
    rig="$(_mforge_active_rig)"
  fi
  [[ -n "$rig" ]] || return 0
  json_path="$(_mforge_home)/rigs/$rig/turn.json"
  [[ -f "$json_path" ]] || return 0
  _mforge_jq '.id // empty' "$json_path" 2>/dev/null
}

_mforge_rig_repo() {
  local rig="$1"
  local json_path
  if [[ -z "$rig" ]]; then
    rig="$(_mforge_active_rig)"
  fi
  [[ -n "$rig" ]] || return 0
  json_path="$(_mforge_home)/rigs/$rig/rig.json"
  [[ -f "$json_path" ]] || return 0
  _mforge_jq '.repo_path // empty' "$json_path" 2>/dev/null
}

_mforge_bd_json() {
  local rig="$1"
  local repo
  repo="$(_mforge_rig_repo "$rig")"
  [[ -n "$repo" ]] || return 0
  command -v bd >/dev/null 2>&1 || return 0
  (cd "$repo" && bd list --json 2>/dev/null)
}

_mforge_bead_ids() {
  local rig="$1"
  local type="$2"
  command -v jq >/dev/null 2>&1 || return 0
  _mforge_bd_json "$rig" | _mforge_jq --arg t "${type:l}" '
    (if type=="object" and has("issues") then .issues
     elif type=="object" then [.] else . end)
    | map(select(type=="object"))
    | (if $t=="" then . else map(select((.type // .issue_type // "") | ascii_downcase == $t)) end)
    | .[] | (.id // empty)
  ' 2>/dev/null
}

_mforge_bead_pairs() {
  local rig="$1"
  local type="$2"
  command -v jq >/dev/null 2>&1 || return 0
  _mforge_bd_json "$rig" | _mforge_jq --arg t "${type:l}" '
    (if type=="object" and has("issues") then .issues
     elif type=="object" then [.] else . end)
    | map(select(type=="object"))
    | (if $t=="" then . else map(select((.type // .issue_type // "") | ascii_downcase == $t)) end)
    | map({
        id: (.id // ""),
        desc: (.description // .title // "")
          | gsub("[\\r\\n\\t]+"; " ")
          | (if length > 120 then .[0:120] + "..." else . end)
      })
    | map(select(.id != ""))
    | map([.id, .desc] | @tsv)
    | .[]
  ' 2>/dev/null
}

_mforge_bead_types() {
  local rig="$1"
  command -v jq >/dev/null 2>&1 || return 0
  _mforge_bd_json "$rig" | _mforge_jq '
    (if type=="object" and has("issues") then .issues
     elif type=="object" then [.] else . end)
    | map(select(type=="object"))
    | map(.type // .issue_type // empty)
    | map(select(. != ""))
    | unique
    | .[]
  ' 2>/dev/null
}

_mforge_agent_specs() {
  local repo
  repo="$(_mforge_rig_repo)"
  [[ -n "$repo" ]] || return 0
  local dir="$repo/.mf/agents"
  [[ -d "$dir" ]] || return 0
  local f
  for f in "$dir"/*.json; do
    [[ -f "$f" ]] || continue
    basename "$f" .json
  done
}

_mforge_should_fzf() {
  [[ -z "${MF_FZF_DISABLE:-}" ]] && command -v fzf >/dev/null 2>&1
}

_mforge_complete_from_list() {
  local -a items
  items=("$@")
  if [[ ${#items[@]} -eq 0 ]]; then
    return 0
  fi
  if _mforge_should_fzf; then
    local sel
    sel=$(printf '%s\n' "${items[@]}" | fzf --height 40% --reverse --prompt='mforge> ' --query "$PREFIX" 2>/dev/null)
    [[ -n "$sel" ]] && compadd -Q -- "$sel"
    zle -R 2>/dev/null || true
    return 0
  fi
  compadd -Q -- "${items[@]}"
}

_mforge_complete_from_pairs() {
  local -a items
  items=("$@")
  if [[ ${#items[@]} -eq 0 ]]; then
    return 0
  fi
  if _mforge_should_fzf; then
    local sel
    sel=$(printf '%s\n' "${items[@]}" | fzf --height 40% --reverse --prompt='mforge> ' --query "$PREFIX" --delimiter=$'\t' --with-nth=1,2 2>/dev/null)
    sel="${sel%%$'\t'*}"
    [[ -n "$sel" ]] && compadd -Q -- "$sel"
    zle -R 2>/dev/null || true
    return 0
  fi
  local -a ids
  local item
  for item in "${items[@]}"; do
    ids+=("${item%%$'\t'*}")
  done
  compadd -Q -- "${ids[@]}"
}

_mforge() {
  local cur prev cmd sub
  cur="$words[CURRENT]"
  prev="$words[CURRENT-1]"
  cmd="$words[2]"
  sub="$words[3]"

  if (( CURRENT == 2 )); then
    _mforge_complete_from_list init cell agent task request monitor epic manager turn round checkpoint bead review pr merge wait coordinator digest build deploy contract architect report library scope engine convoy watch quick-assign tui migrate context rig ssh completions hook help
    return
  fi

  if [[ "$cmd" == "help" ]]; then
    _mforge_complete_from_list init cell agent task request monitor epic manager turn round checkpoint bead review pr merge wait coordinator digest build deploy contract architect report library scope engine convoy watch quick-assign tui migrate context rig ssh completions hook
    return
  fi

  case "$cmd" in
    init)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list $(_mforge_rigs)
      else
        _mforge_complete_from_list --repo
      fi
      return
      ;;
    cell)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list add bootstrap agent-file
        return
      fi
      case "$sub" in
        add)
          if (( CURRENT == 4 )); then
            return
          fi
          if [[ "$prev" == "--scope" ]]; then
            _mforge_complete_from_list $(_mforge_scopes)
            return
          fi
          _mforge_complete_from_list --scope
          return
          ;;
        bootstrap)
          if (( CURRENT == 4 )); then
            _mforge_complete_from_list $(_mforge_cells)
            return
          fi
          _mforge_complete_from_list --architect --single
          return
          ;;
        agent-file)
          if (( CURRENT == 4 )); then
            _mforge_complete_from_list $(_mforge_cells)
            return
          fi
          if [[ "$prev" == "--role" ]]; then
            _mforge_complete_from_list builder monitor reviewer architect cell
            return
          fi
          _mforge_complete_from_list --role
          return
          ;;
      esac
      ;;
    agent)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list spawn stop attach wake relaunch send status logs heartbeat create bootstrap
        return
      fi
      if [[ "$sub" == "send" ]]; then
        _mforge_complete_from_list --no-enter
        return
      fi
      if [[ "$sub" == "create" ]]; then
        if [[ "$prev" == "--description" ]]; then
          return
        fi
        if [[ "$prev" == "--class" ]]; then
          _mforge_complete_from_list crew worker
          return
        fi
        _mforge_complete_from_list --description --class
        return
      fi
      if [[ "$sub" == "bootstrap" ]]; then
        if (( CURRENT == 4 )); then
          _mforge_complete_from_list $(_mforge_agent_specs)
          return
        fi
        return
      fi
      if [[ "$sub" == "status" ]]; then
        case "$prev" in
          --cell)
            _mforge_complete_from_list $(_mforge_cells)
            return
            ;;
          --role)
            _mforge_complete_from_list builder monitor reviewer architect cell
            return
            ;;
        esac
        _mforge_complete_from_list --cell --role --remote --json
        return
      fi
      if [[ "$sub" == "heartbeat" ]]; then
        if (( CURRENT == 4 )); then
          _mforge_complete_from_list $(_mforge_cells)
          return
        fi
        if (( CURRENT == 5 )); then
          _mforge_complete_from_list builder monitor reviewer architect cell
          return
        fi
        return
      fi
      if [[ "$sub" == "logs" ]]; then
        if (( CURRENT == 4 )); then
          _mforge_complete_from_list $(_mforge_cells)
          return
        fi
        if (( CURRENT == 5 )); then
          _mforge_complete_from_list builder monitor reviewer architect cell
          return
        fi
        _mforge_complete_from_list --follow --lines --all
        return
      fi
      if (( CURRENT == 4 )); then
        _mforge_complete_from_list $(_mforge_cells)
        return
      fi
      if (( CURRENT == 5 )); then
        _mforge_complete_from_list builder monitor reviewer architect cell
        return
      fi
      return
      ;;
    task)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list create update list split decompose complete delete
        return
      fi
      case "$sub" in
        create)
          case "$prev" in
            --scope)
              _mforge_complete_from_list $(_mforge_scopes)
              return
              ;;
            --kind)
              _mforge_complete_from_list improve fix review monitor doc
              return
              ;;
            --title|--body)
              return
              ;;
          esac
          _mforge_complete_from_list --title --body --scope --kind
          return
          ;;
        update)
          case "$prev" in
            --task)
              _mforge_complete_from_pairs $(_mforge_bead_pairs "" task)
              return
              ;;
            --scope)
              _mforge_complete_from_list $(_mforge_scopes)
              return
              ;;
          esac
          _mforge_complete_from_list --task --scope
          return
          ;;
        list)
          return
          ;;
        split)
          case "$prev" in
            --task)
              _mforge_complete_from_pairs $(_mforge_bead_pairs "" task)
              return
              ;;
            --cells)
              _mforge_complete_from_list $(_mforge_cells)
              return
              ;;
          esac
          _mforge_complete_from_list --task --cells
          return
          ;;
        decompose)
          case "$prev" in
            --task)
              _mforge_complete_from_pairs $(_mforge_bead_pairs "" task)
              return
              ;;
            --titles|--kind)
              return
              ;;
          esac
          _mforge_complete_from_list --task --titles --kind
          return
          ;;
        complete)
          case "$prev" in
            --task)
              _mforge_complete_from_pairs $(_mforge_bead_pairs "" task)
              return
              ;;
            --reason)
              return
              ;;
          esac
          _mforge_complete_from_list --task --reason --force
          return
          ;;
        delete)
          case "$prev" in
            --task)
              _mforge_complete_from_pairs $(_mforge_bead_pairs "" task)
              return
              ;;
            --reason)
              return
              ;;
          esac
          _mforge_complete_from_list --task --reason --force --cascade --hard --dry-run
          return
          ;;
      esac
      ;;
    assign)
      case "$prev" in
        --task)
          _mforge_complete_from_pairs $(_mforge_bead_pairs "" task)
          return
          ;;
        --cell)
          _mforge_complete_from_list $(_mforge_cells)
          return
          ;;
        --role)
          _mforge_complete_from_list builder monitor reviewer architect cell
          return
          ;;
      esac
      _mforge_complete_from_list --task --cell --role --promise --quick
      return
      ;;
    quick-assign)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list $(_mforge_bead_ids "")
        return
      fi
      if (( CURRENT == 4 )); then
        _mforge_complete_from_list $(_mforge_cells)
        return
      fi
      if [[ "$prev" == "--role" ]]; then
        _mforge_complete_from_list builder monitor reviewer architect cell
        return
      fi
      _mforge_complete_from_list --role --promise
      return
      ;;
    request)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list create list triage
        return
      fi
      case "$sub" in
        create)
          case "$prev" in
            --cell)
              _mforge_complete_from_list $(_mforge_cells)
              return
              ;;
            --role)
              _mforge_complete_from_list builder monitor reviewer architect cell
              return
              ;;
            --severity)
              _mforge_complete_from_list low med high
              return
              ;;
            --priority)
              _mforge_complete_from_list p1 p2 p3
              return
              ;;
            --scope)
              _mforge_complete_from_list $(_mforge_scopes)
              return
              ;;
            --payload)
              return
              ;;
          esac
          _mforge_complete_from_list --cell --role --severity --priority --scope --payload
          return
          ;;
        list)
          case "$prev" in
            --cell)
              _mforge_complete_from_list $(_mforge_cells)
              return
              ;;
            --status)
              _mforge_complete_from_list open in_progress blocked done closed
              return
              ;;
            --priority)
              _mforge_complete_from_list p1 p2 p3
              return
              ;;
          esac
          _mforge_complete_from_list --cell --status --priority
          return
          ;;
        triage)
          case "$prev" in
            --request)
              _mforge_complete_from_list $(_mforge_bead_ids "" request)
              return
              ;;
            --action)
              _mforge_complete_from_list create-task merge block
              return
              ;;
          esac
          _mforge_complete_from_list --request --action
          return
          ;;
      esac
      ;;
    scope)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list list show
        return
      fi
      case "$sub" in
        list)
          return
          ;;
        show)
          case "$prev" in
            --scope)
              _mforge_complete_from_list $(_mforge_scopes)
              return
              ;;
          esac
          _mforge_complete_from_list --scope
          return
          ;;
      esac
      ;;
    context)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list get set unset list
        return
      fi
      if [[ "$sub" == "set" && CURRENT == 4 ]]; then
        _mforge_complete_from_list $(_mforge_rigs)
        return
      fi
      return
      ;;
    rig)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list list delete rename backup restore
        return
      fi
      case "$sub" in
        list)
          return
          ;;
        delete|backup)
          if (( CURRENT == 4 )); then
            _mforge_complete_from_list $(_mforge_rigs)
            return
          fi
          return
          ;;
        rename)
          if (( CURRENT == 4 )); then
            _mforge_complete_from_list $(_mforge_rigs)
            return
          fi
          return
          ;;
        restore)
          if [[ "$prev" == "--name" ]]; then
            return
          fi
          _mforge_complete_from_list --name --force
          return
          ;;
      esac
      ;;
    engine)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list run emit drain
        return
      fi
      case "$sub" in
        run)
          _mforge_complete_from_list --wait --rounds --completion-promise
          return
          ;;
        emit)
          case "$prev" in
            --scope)
              _mforge_complete_from_list $(_mforge_scopes)
              return
              ;;
            --type|--title|--source|--payload)
              return
              ;;
          esac
          _mforge_complete_from_list --type --scope --title --source --payload
          return
          ;;
        drain)
      _mforge_complete_from_list --keep
      return
      ;;
    convoy)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list start
        return
      fi
      case "$prev" in
        --epic)
          _mforge_complete_from_list $(_mforge_bead_ids "" epic)
          return
          ;;
        --role)
          _mforge_complete_from_list builder monitor reviewer architect cell
          return
          ;;
        --title)
          return
          ;;
      esac
      _mforge_complete_from_list --epic --role --title
      return
      ;;
      esac
      ;;
    monitor)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list run-tests run
        return
      fi
      if (( CURRENT == 4 )); then
        _mforge_complete_from_list $(_mforge_cells)
        return
      fi
      case "$prev" in
        --severity)
          _mforge_complete_from_list low med high
          return
          ;;
        --priority)
          _mforge_complete_from_list p1 p2 p3
          return
          ;;
        --scope)
          _mforge_complete_from_list $(_mforge_scopes)
          return
          ;;
        --observation|--cmd)
          return
          ;;
      esac
      _mforge_complete_from_list --cmd --severity --priority --scope --observation
      return
      ;;
    epic)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list create add-task assign status close conflict design tree
        return
      fi
      case "$sub" in
        create)
          case "$prev" in
            --title|--body|--short-id)
              return
              ;;
          esac
          _mforge_complete_from_list --title --body --short-id
          return
          ;;
        add-task)
          case "$prev" in
            --epic)
              _mforge_complete_from_list $(_mforge_bead_ids "" epic)
              return
              ;;
            --task)
              _mforge_complete_from_pairs $(_mforge_bead_pairs "" task)
              return
              ;;
          esac
          _mforge_complete_from_list --epic --task
          return
          ;;
        assign)
          case "$prev" in
            --epic)
              _mforge_complete_from_list $(_mforge_bead_ids "" epic)
              return
              ;;
            --role)
              _mforge_complete_from_list builder monitor reviewer architect cell
              return
              ;;
          esac
          _mforge_complete_from_list --epic --role
          return
          ;;
        status|close)
          if [[ "$prev" == "--epic" ]]; then
            _mforge_complete_from_list $(_mforge_bead_ids "" epic)
            return
          fi
          _mforge_complete_from_list --epic
          return
          ;;
        conflict)
          case "$prev" in
            --epic)
              _mforge_complete_from_list $(_mforge_bead_ids "" epic)
              return
              ;;
            --cell)
              _mforge_complete_from_list $(_mforge_cells)
              return
              ;;
            --details)
              return
              ;;
          esac
          _mforge_complete_from_list --epic --cell --details
          return
          ;;
        design|tree)
          if (( CURRENT == 4 )); then
            _mforge_complete_from_pairs $(_mforge_bead_pairs "" epic)
            return
          fi
          return
          ;;
      esac
      ;;
    architect)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list docs contract design
        return
      fi
      case "$prev" in
        --cell)
          _mforge_complete_from_list $(_mforge_cells)
          return
          ;;
        --scope)
          _mforge_complete_from_list $(_mforge_scopes)
          return
          ;;
        --details)
          return
          ;;
      esac
      _mforge_complete_from_list --cell --details --scope
      return
      ;;
    report)
      if [[ "$prev" == "--cell" ]]; then
        _mforge_complete_from_list $(_mforge_cells)
        return
      fi
      _mforge_complete_from_list --cell
      return
      ;;
    library)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list start query
        return
      fi
      case "$sub" in
        start)
          _mforge_complete_from_list --addr
          return
          ;;
        query)
          case "$prev" in
            --q|--service|--addr)
              return
              ;;
          esac
          _mforge_complete_from_list --q --service --addr
          return
          ;;
      esac
      ;;
    watch)
      _mforge_complete_from_list --interval --role --fswatch --tui
      return
      ;;
    tui)
      _mforge_complete_from_list --interval --remote --watch --role
      return
      ;;
    migrate)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list beads rig
        return
      fi
      _mforge_complete_from_list --all
      return
      ;;
    completions)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list install path bash zsh
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
      _mforge_complete_from_list --cmd --tty
      return
      ;;
    manager)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list tick assign
        return
      fi
      if [[ "$prev" == "--role" ]]; then
        _mforge_complete_from_list builder monitor reviewer architect cell
        return
      fi
      _mforge_complete_from_list --watch --role
      return
      ;;
    turn)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list start status end slate run list diff
        return
      fi
      if [[ "$sub" == "start" ]]; then
        _mforge_complete_from_list --name
        return
      fi
      if [[ "$sub" == "end" ]]; then
        _mforge_complete_from_list --report
        return
      fi
      if [[ "$sub" == "diff" ]]; then
        _mforge_complete_from_list --id
        return
      fi
      if [[ "$sub" == "run" ]]; then
        if [[ "$prev" == "--role" ]]; then
          _mforge_complete_from_list builder monitor reviewer architect cell
          return
        fi
        _mforge_complete_from_list --role --wait
        return
      fi
      return
      ;;
    round)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list start review merge
        return
      fi
      if [[ "$sub" == "review" ]]; then
        _mforge_complete_from_list --wait --all --changes-only --base
        return
      fi
      if [[ "$sub" == "merge" ]]; then
        _mforge_complete_from_list --feature --base
        return
      fi
      _mforge_complete_from_list --wait
      return
      ;;
    checkpoint)
      _mforge_complete_from_list --message
      return
      ;;
    bead)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list create list show close status triage dep template
        return
      fi
      if [[ "$sub" == "dep" ]]; then
        if (( CURRENT == 4 )); then
          _mforge_complete_from_list add
          return
        fi
        if (( CURRENT == 6 )); then
          _mforge_complete_from_list $(_mforge_bead_ids "")
          return
        fi
        return
      fi
      case "$prev" in
        --type)
          local types
          types=$(_mforge_bead_types "")
          if [[ -n "$types" ]]; then
            _mforge_complete_from_list ${(f)types}
          else
            _mforge_complete_from_list task request observation assignment pr review epic contract decision build deploy conflictresolution
          fi
          return
          ;;
        --status)
          _mforge_complete_from_list open in_progress blocked done closed ready queued
          return
          ;;
        --priority)
          _mforge_complete_from_list p1 p2 p3
          return
          ;;
        --cell)
          _mforge_complete_from_list $(_mforge_cells)
          return
          ;;
        --role)
          _mforge_complete_from_list builder monitor reviewer architect cell
          return
          ;;
        --scope)
          _mforge_complete_from_list $(_mforge_scopes)
          return
          ;;
        --turn)
          _mforge_complete_from_list $(_mforge_turn_id)
          return
          ;;
        --severity)
          _mforge_complete_from_list low med high
          return
          ;;
        --deps|--description|--acceptance|--compat|--links|--title)
          return
          ;;
      esac
      if [[ "$sub" == "show" || "$sub" == "close" || "$sub" == "status" ]]; then
        if (( CURRENT == 4 )); then
          _mforge_complete_from_list $(_mforge_bead_ids "")
          return
        fi
      fi
      if [[ "$sub" == "status" ]]; then
        if (( CURRENT == 5 )); then
          _mforge_complete_from_list open in_progress blocked done closed ready queued
          return
        fi
      fi
      if [[ "$sub" == "triage" ]]; then
        case "$prev" in
          --id)
            _mforge_complete_from_list $(_mforge_bead_ids "")
            return
            ;;
          --cell)
            _mforge_complete_from_list $(_mforge_cells)
            return
            ;;
          --role)
            _mforge_complete_from_list builder monitor reviewer architect cell
            return
            ;;
          --turn)
            _mforge_complete_from_list $(_mforge_turn_id)
            return
            ;;
        esac
      fi
      return
      ;;
    review)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list create
        return
      fi
      case "$prev" in
        --cell)
          _mforge_complete_from_list $(_mforge_cells)
          return
          ;;
        --scope)
          _mforge_complete_from_list $(_mforge_scopes)
          return
          ;;
        --title)
          return
          ;;
      esac
      _mforge_complete_from_list --title --cell --scope --turn
      return
      ;;
    pr)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list create ready link-review
        return
      fi
      case "$sub" in
        create)
          case "$prev" in
            --cell)
              _mforge_complete_from_list $(_mforge_cells)
              return
              ;;
            --title|--url)
              return
              ;;
          esac
          _mforge_complete_from_list --title --cell --url --turn --status
          return
          ;;
        ready)
          if (( CURRENT == 4 )); then
            _mforge_complete_from_list $(_mforge_bead_ids "" pr)
            return
          fi
          return
          ;;
        link-review)
          if (( CURRENT == 4 )); then
            _mforge_complete_from_list $(_mforge_bead_ids "" pr)
            return
          fi
          if (( CURRENT == 5 )); then
            _mforge_complete_from_list $(_mforge_bead_ids "" review)
            return
          fi
          return
          ;;
      esac
      ;;
    merge)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list run
        return
      fi
      case "$prev" in
        --turn)
          _mforge_complete_from_list $(_mforge_turn_id)
          return
          ;;
        --as)
          _mforge_complete_from_list merge-manager
          return
          ;;
      esac
      _mforge_complete_from_list --turn --as --dry-run
      return
      ;;
    wait)
      case "$prev" in
        --turn)
          _mforge_complete_from_list $(_mforge_turn_id)
          return
          ;;
        --interval)
          return
          ;;
      esac
      _mforge_complete_from_list --turn --interval
      return
      ;;
    coordinator)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list sync
        return
      fi
      if [[ "$prev" == "--turn" ]]; then
        _mforge_complete_from_list $(_mforge_turn_id)
        return
      fi
      _mforge_complete_from_list --turn
      return
      ;;
    digest)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list render
        return
      fi
      if [[ "$prev" == "--turn" ]]; then
        _mforge_complete_from_list $(_mforge_turn_id)
        return
      fi
      _mforge_complete_from_list --turn
      return
      ;;
    build)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list record
        return
      fi
      _mforge_complete_from_list --service --image --status --turn
      return
      ;;
    deploy)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list record
        return
      fi
      _mforge_complete_from_list --env --service --status --turn
      return
      ;;
    contract)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list create
        return
      fi
      case "$prev" in
        --cell)
          _mforge_complete_from_list $(_mforge_cells)
          return
          ;;
        --scope)
          _mforge_complete_from_list $(_mforge_scopes)
          return
          ;;
        --title|--acceptance|--compat|--links)
          return
          ;;
      esac
      _mforge_complete_from_list --title --cell --scope --acceptance --compat --links
      return
      ;;
    hook)
      if (( CURRENT == 3 )); then
        _mforge_complete_from_list stop guardrails emit
        return
      fi
      if [[ "$sub" == "emit" ]]; then
        _mforge_complete_from_list --event
        return
      fi
      return
      ;;
  esac
}

compdef _mforge mforge
