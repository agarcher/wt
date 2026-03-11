package shell

import "fmt"

// GenerateZsh generates the zsh shell integration script
func GenerateZsh() string {
	return `# wt shell integration for zsh
# Add this to your ~/.zshrc: eval "$(wt init zsh)"

# Completion function for wt
_wt() {
  local curcontext="$curcontext" state line
  typeset -A opt_args

  _arguments -C \
    '1: :->command' \
    '*: :->args'

  case $state in
    command)
      local commands=(
        'create:Create a new worktree'
        'delete:Delete a worktree'
        'cd:Change to a worktree directory'
        'info:Show detailed information about a worktree'
        'list:List all worktrees'
        'cleanup:Clean up merged worktrees'
        'exit:Return to the main repository'
        'setup:Interactive setup for worktree management'
        'config:Manage user configuration'
        'init:Generate shell integration script'
        'root:Print the main repository root path'
        'completion:Generate shell completion script'
        'version:Print the version number'
        'help:Help about any command'
      )
      _describe 'command' commands
      ;;
    args)
      case $words[2] in
        cd|info)
          # Only complete worktree names for the first argument
          local has_name=false
          for ((i=3; i < $CURRENT; i++)); do
            if [[ ${words[$i]} != -* ]]; then
              has_name=true
              break
            fi
          done
          if ! $has_name; then
            # Use cobra's built-in completion
            local completions
            completions=(${(f)"$(command wt __complete "$words[2]" -- "$words[$CURRENT]" 2>/dev/null | grep -v '^:')"})
            if [[ ${#completions[@]} -gt 0 ]]; then
              _describe 'worktree' completions
            fi
          fi
          ;;
        delete)
          # Check if a worktree name (non-flag argument) has already been provided
          local has_name=false
          for ((i=3; i < $CURRENT; i++)); do
            if [[ ${words[$i]} != -* ]]; then
              has_name=true
              break
            fi
          done
          if $has_name || [[ ${words[$CURRENT]} == -* ]]; then
            # Worktree name already provided or typing a flag - complete flags
            local -a flags=(
              '-f:Force deletion'
              '--force:Force deletion'
              '-k:Keep the associated branch'
              '--keep-branch:Keep the associated branch'
            )
            _describe 'flag' flags
          else
            # Use cobra's built-in completion
            local completions
            completions=(${(f)"$(command wt __complete delete -- "$words[$CURRENT]" 2>/dev/null | grep -v '^:')"})
            if [[ ${#completions[@]} -gt 0 ]]; then
              _describe 'worktree' completions
            fi
          fi
          ;;
        create)
          _arguments \
            '-b[Use existing branch]:branch:->branches' \
            '--branch[Use existing branch]:branch:->branches'
          if [[ $state == branches ]]; then
            local branches
            # Try --format first (Git 2.13+), fall back to parsing git branch output
            branches=(${(f)"$(git branch --format='%(refname:short)' 2>/dev/null || git branch 2>/dev/null | sed 's/^[* ] //')"})
            _describe 'branch' branches
          fi
          ;;
        setup)
          _arguments \
            '--personal[Save to personal config]' \
            '--shared[Save to repository .wt.yaml]'
          ;;
        init|completion)
          local shells=(zsh bash fish)
          _describe 'shell' shells
          ;;
        cleanup)
          _arguments \
            '-f[Force deletion]' \
            '--force[Force deletion]' \
            '-k[Keep the associated branch]' \
            '--keep-branch[Keep the associated branch]' \
            '-n[Dry run - show what would be deleted]' \
            '--dry-run[Dry run - show what would be deleted]'
          ;;
        list)
          _arguments \
            '-v[Show detailed status]' \
            '--verbose[Show detailed status]'
          ;;
        config)
          _arguments \
            '--global[Set/get global configuration]' \
            '--unset[Remove a per-repo configuration value]' \
            '--list[List all configuration values]' \
            '--show-origin[Show where each value comes from]'
          ;;
      esac
      ;;
  esac
}

# Register completion for wt function
compdef _wt wt

wt() {
  # Check if we're in a git repo
  local repo_root
  repo_root=$(git rev-parse --show-toplevel 2>/dev/null)

  if [[ -z "$repo_root" ]]; then
    # Not in a git repo - run wt directly (might be a global command)
    command wt "$@"
    return $?
  fi

  # Commands that need cd handling
  case "$1" in
    create|delete|cleanup)
      # Use temp file to communicate cd target from Go
      local cdfile=$(mktemp)
      trap "rm -f '$cdfile'" EXIT
      WT_CD_FILE="$cdfile" command wt "$@"
      local exit_code=$?

      # cd to path if one was written
      if [[ -s "$cdfile" ]]; then
        local target=$(cat "$cdfile")
        if [[ -d "$target" ]]; then
          cd "$target"
        fi
      fi
      rm -f "$cdfile"
      trap - EXIT
      return $exit_code
      ;;
    cd)
      local output
      output=$(command wt "$@" 2>&1)
      local exit_code=$?

      if [[ $exit_code -eq 0 ]]; then
        local target="$output"
        if [[ -d "$target" ]]; then
          cd "$target"
        else
          echo "$output"
        fi
      else
        echo "$output"
      fi
      return $exit_code
      ;;
    exit)
      local target
      target=$(command wt root 2>/dev/null)
      if [[ -d "$target" ]]; then
        cd "$target"
      else
        echo "Could not find repository root"
        return 1
      fi
      ;;
    *)
      command wt "$@"
      ;;
  esac
}
`
}

// GenerateBash generates the bash shell integration script
func GenerateBash() string {
	return `# wt shell integration for bash
# Add this to your ~/.bashrc: eval "$(wt init bash)"

# Completion function for wt
_wt_completions() {
  local cur prev words cword
  _init_completion 2>/dev/null || {
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
  }

  local commands="create delete cd info list cleanup exit setup config init root completion version help"

  if [[ $COMP_CWORD -eq 1 ]]; then
    COMPREPLY=($(compgen -W "$commands" -- "$cur"))
    return 0
  fi

  local cmd="${COMP_WORDS[1]}"
  case "$cmd" in
    cd|info)
      # Use cobra's built-in completion
      local completions
      completions=$(command wt __complete "$cmd" -- "$cur" 2>/dev/null | grep -v '^:')
      if [[ -n "$completions" ]]; then
        COMPREPLY=($(compgen -W "$completions" -- "$cur"))
      fi
      ;;
    delete)
      # Use cobra's built-in completion for worktree names and flags
      local completions
      completions=$(command wt __complete delete -- "$cur" 2>/dev/null | grep -v '^:')
      if [[ -n "$completions" ]]; then
        COMPREPLY=($(compgen -W "$completions -f --force -k --keep-branch" -- "$cur"))
      else
        COMPREPLY=($(compgen -W "-f --force -k --keep-branch" -- "$cur"))
      fi
      ;;
    create)
      case "$prev" in
        -b|--branch)
          # Try --format first (Git 2.13+), fall back to parsing git branch output
          local branches=$(git branch --format='%(refname:short)' 2>/dev/null || git branch 2>/dev/null | sed 's/^[* ] //')
          COMPREPLY=($(compgen -W "$branches" -- "$cur"))
          ;;
        *)
          COMPREPLY=($(compgen -W "-b --branch" -- "$cur"))
          ;;
      esac
      ;;
    setup)
      COMPREPLY=($(compgen -W "--personal --shared" -- "$cur"))
      ;;
    init|completion)
      COMPREPLY=($(compgen -W "zsh bash fish" -- "$cur"))
      ;;
    cleanup)
      COMPREPLY=($(compgen -W "-n --dry-run -f --force -k --keep-branch" -- "$cur"))
      ;;
    list)
      COMPREPLY=($(compgen -W "-v --verbose" -- "$cur"))
      ;;
    config)
      COMPREPLY=($(compgen -W "--global --unset --list --show-origin remote fetch_interval worktree_dir branch_pattern default_branch" -- "$cur"))
      ;;
  esac
  return 0
}

# Register completion for wt function
complete -F _wt_completions wt

wt() {
  # Check if we're in a git repo
  local repo_root
  repo_root=$(git rev-parse --show-toplevel 2>/dev/null)

  if [[ -z "$repo_root" ]]; then
    command wt "$@"
    return $?
  fi

  case "$1" in
    create|delete|cleanup)
      # Use temp file to communicate cd target from Go
      local cdfile=$(mktemp)
      trap "rm -f '$cdfile'" EXIT
      WT_CD_FILE="$cdfile" command wt "$@"
      local exit_code=$?

      # cd to path if one was written
      if [[ -s "$cdfile" ]]; then
        local target=$(cat "$cdfile")
        if [[ -d "$target" ]]; then
          cd "$target"
        fi
      fi
      rm -f "$cdfile"
      trap - EXIT
      return $exit_code
      ;;
    cd)
      local output
      output=$(command wt "$@" 2>&1)
      local exit_code=$?

      if [[ $exit_code -eq 0 ]]; then
        local target="$output"
        if [[ -d "$target" ]]; then
          cd "$target"
        else
          echo "$output"
        fi
      else
        echo "$output"
      fi
      return $exit_code
      ;;
    exit)
      local target
      target=$(command wt root 2>/dev/null)
      if [[ -d "$target" ]]; then
        cd "$target"
      else
        echo "Could not find repository root"
        return 1
      fi
      ;;
    *)
      command wt "$@"
      ;;
  esac
}
`
}

// GenerateFish generates the fish shell integration script
func GenerateFish() string {
	return `# wt shell integration for fish
# Add this to your ~/.config/fish/config.fish: wt init fish | source

# Completions for wt
complete -c wt -f  # Disable file completion by default

# Subcommands
complete -c wt -n "__fish_use_subcommand" -a "create" -d "Create a new worktree"
complete -c wt -n "__fish_use_subcommand" -a "delete" -d "Delete a worktree"
complete -c wt -n "__fish_use_subcommand" -a "cd" -d "Change to a worktree directory"
complete -c wt -n "__fish_use_subcommand" -a "list" -d "List all worktrees"
complete -c wt -n "__fish_use_subcommand" -a "info" -d "Show detailed information about a worktree"
complete -c wt -n "__fish_use_subcommand" -a "cleanup" -d "Clean up merged worktrees"
complete -c wt -n "__fish_use_subcommand" -a "exit" -d "Return to the main repository"
complete -c wt -n "__fish_use_subcommand" -a "setup" -d "Interactive setup for worktree management"
complete -c wt -n "__fish_use_subcommand" -a "config" -d "Manage user configuration"
complete -c wt -n "__fish_use_subcommand" -a "init" -d "Generate shell integration script"
complete -c wt -n "__fish_use_subcommand" -a "root" -d "Print the main repository root path"
complete -c wt -n "__fish_use_subcommand" -a "completion" -d "Generate shell completion script"
complete -c wt -n "__fish_use_subcommand" -a "version" -d "Print the version number"
complete -c wt -n "__fish_use_subcommand" -a "help" -d "Help about any command"

# Helper function to get worktree names via cobra completions
function __wt_worktrees
  command wt __complete cd -- "" 2>/dev/null | grep -v '^:'
end

# Worktree name completion for cd, delete, and info
complete -c wt -n "__fish_seen_subcommand_from cd delete info" -a "(__wt_worktrees)"

# Branch completion for create --branch
# Try --format first (Git 2.13+), fall back to parsing git branch output
complete -c wt -n "__fish_seen_subcommand_from create" -s b -l branch -d "Use existing branch" -a "(git branch --format='%(refname:short)' 2>/dev/null; or git branch 2>/dev/null | sed 's/^[* ] //')"

# Shell completion for init and completion commands
complete -c wt -n "__fish_seen_subcommand_from init completion" -a "zsh bash fish"

# Flags for setup
complete -c wt -n "__fish_seen_subcommand_from setup" -l personal -d "Save to personal config"
complete -c wt -n "__fish_seen_subcommand_from setup" -l shared -d "Save to repository .wt.yaml"

# Flags for delete
complete -c wt -n "__fish_seen_subcommand_from delete" -s f -l force -d "Force deletion"
complete -c wt -n "__fish_seen_subcommand_from delete" -s k -l keep-branch -d "Keep the associated branch"

# Flags for cleanup
complete -c wt -n "__fish_seen_subcommand_from cleanup" -s n -l dry-run -d "Show what would be deleted"
complete -c wt -n "__fish_seen_subcommand_from cleanup" -s f -l force -d "Skip confirmation"
complete -c wt -n "__fish_seen_subcommand_from cleanup" -s k -l keep-branch -d "Keep the associated branches"

# Flags for list
complete -c wt -n "__fish_seen_subcommand_from list" -s v -l verbose -d "Show detailed status"

# Flags for config
complete -c wt -n "__fish_seen_subcommand_from config" -l global -d "Set/get global configuration"
complete -c wt -n "__fish_seen_subcommand_from config" -l unset -d "Remove a per-repo configuration value"
complete -c wt -n "__fish_seen_subcommand_from config" -l list -d "List all configuration values"
complete -c wt -n "__fish_seen_subcommand_from config" -l show-origin -d "Show where each value comes from"

function wt
  # Check if we're in a git repo
  set -l repo_root (git rev-parse --show-toplevel 2>/dev/null)

  if test -z "$repo_root"
    command wt $argv
    return $status
  end

  switch $argv[1]
    case create delete cleanup
      # Use temp file to communicate cd target from Go
      set -l cdfile (mktemp)
      WT_CD_FILE="$cdfile" command wt $argv
      set -l exit_code $status

      # cd to path if one was written
      if test -s "$cdfile"
        set -l target (cat "$cdfile")
        if test -d "$target"
          cd "$target"
        end
      end
      rm -f "$cdfile"
      return $exit_code

    case cd
      set -l output (command wt $argv 2>&1)
      set -l exit_code $status

      if test $exit_code -eq 0
        if test -d "$output"
          cd "$output"
        else
          echo "$output"
        end
      else
        echo "$output"
      end
      return $exit_code

    case exit
      set -l target (command wt root 2>/dev/null)
      if test -d "$target"
        cd "$target"
      else
        echo "Could not find repository root"
        return 1
      end

    case '*'
      command wt $argv
  end
end
`
}

// Generate returns the shell integration script for the given shell
func Generate(shell string) (string, error) {
	switch shell {
	case "zsh":
		return GenerateZsh(), nil
	case "bash":
		return GenerateBash(), nil
	case "fish":
		return GenerateFish(), nil
	default:
		return "", fmt.Errorf("unsupported shell: %s (supported: zsh, bash, fish)", shell)
	}
}
