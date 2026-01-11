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
        'list:List all worktrees'
        'cleanup:Clean up merged worktrees'
        'exit:Return to the main repository'
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
        cd|delete)
          # Complete worktree names
          local repo_root worktree_dir worktrees
          repo_root=$(git rev-parse --show-toplevel 2>/dev/null)
          if [[ -n "$repo_root" ]]; then
            # Check if in worktree and get main repo
            if [[ -f "$repo_root/.git" ]]; then
              local gitdir=$(grep "^gitdir:" "$repo_root/.git" | cut -d' ' -f2)
              if [[ -n "$gitdir" ]]; then
                repo_root=$(dirname $(dirname $(dirname "$gitdir")))
              fi
            fi
            if [[ -f "$repo_root/.wt.yaml" ]]; then
              worktree_dir=$(grep "^worktree_dir:" "$repo_root/.wt.yaml" | cut -d' ' -f2 | tr -d '"' | tr -d "'")
              [[ -z "$worktree_dir" ]] && worktree_dir="worktrees"
              if [[ -d "$repo_root/$worktree_dir" ]]; then
                worktrees=(${(f)"$(ls -1 "$repo_root/$worktree_dir" 2>/dev/null)"})
                _describe 'worktree' worktrees
              fi
            fi
          fi
          ;;
        create)
          _arguments \
            '-b[Use existing branch]:branch:->branches' \
            '--branch[Use existing branch]:branch:->branches'
          if [[ $state == branches ]]; then
            local branches
            branches=(${(f)"$(git branch --format='%(refname:short)' 2>/dev/null)"})
            _describe 'branch' branches
          fi
          ;;
        init|completion)
          local shells=(zsh bash fish)
          _describe 'shell' shells
          ;;
        delete|cleanup)
          _arguments \
            '-f[Force deletion]' \
            '--force[Force deletion]' \
            '-k[Keep the associated branch]' \
            '--keep-branch[Keep the associated branch]'
          ;;
        cleanup)
          _arguments \
            '-n[Dry run - show what would be deleted]' \
            '--dry-run[Dry run - show what would be deleted]'
          ;;
        list)
          _arguments \
            '-v[Show detailed status]' \
            '--verbose[Show detailed status]'
          ;;
      esac
      ;;
  esac
}

# Register completion for wt function
compdef _wt wt

wt() {
  # Check if we're in a git repo with .wt.yaml
  local repo_root
  repo_root=$(git rev-parse --show-toplevel 2>/dev/null)

  if [[ -z "$repo_root" ]]; then
    # Not in a git repo - try to run wt anyway (might be a global command)
    command wt "$@"
    return $?
  fi

  # Check for .wt.yaml in repo root or if we're in a worktree, check main repo
  local config_found=false
  if [[ -f "$repo_root/.wt.yaml" ]]; then
    config_found=true
  else
    # Check if this is a worktree and look for config in main repo
    local git_file="$repo_root/.git"
    if [[ -f "$git_file" ]]; then
      local gitdir=$(grep "^gitdir:" "$git_file" | cut -d' ' -f2)
      if [[ -n "$gitdir" ]]; then
        local main_repo=$(dirname $(dirname $(dirname "$gitdir")))
        if [[ -f "$main_repo/.wt.yaml" ]]; then
          config_found=true
          repo_root="$main_repo"
        fi
      fi
    fi
  fi

  if [[ "$config_found" != "true" ]]; then
    command wt "$@"
    return $?
  fi

  # Commands that need cd handling
  case "$1" in
    create)
      local output
      output=$(command wt "$@" 2>&1)
      local exit_code=$?

      if [[ $exit_code -eq 0 ]]; then
        # Print all but last line
        echo "$output" | sed '$d'
        # cd to path on last line
        local target=$(echo "$output" | tail -1)
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

  local commands="create delete cd list cleanup exit init root completion version help"

  if [[ $COMP_CWORD -eq 1 ]]; then
    COMPREPLY=($(compgen -W "$commands" -- "$cur"))
    return 0
  fi

  local cmd="${COMP_WORDS[1]}"
  case "$cmd" in
    cd|delete)
      # Complete worktree names
      local repo_root worktree_dir worktrees
      repo_root=$(git rev-parse --show-toplevel 2>/dev/null)
      if [[ -n "$repo_root" ]]; then
        if [[ -f "$repo_root/.git" ]]; then
          local gitdir=$(grep "^gitdir:" "$repo_root/.git" | cut -d' ' -f2)
          if [[ -n "$gitdir" ]]; then
            repo_root=$(dirname $(dirname $(dirname "$gitdir")))
          fi
        fi
        if [[ -f "$repo_root/.wt.yaml" ]]; then
          worktree_dir=$(grep "^worktree_dir:" "$repo_root/.wt.yaml" | cut -d' ' -f2 | tr -d '"' | tr -d "'")
          [[ -z "$worktree_dir" ]] && worktree_dir="worktrees"
          if [[ -d "$repo_root/$worktree_dir" ]]; then
            worktrees=$(ls -1 "$repo_root/$worktree_dir" 2>/dev/null)
            COMPREPLY=($(compgen -W "$worktrees" -- "$cur"))
          fi
        fi
      fi
      ;;
    create)
      case "$prev" in
        -b|--branch)
          local branches=$(git branch --format='%(refname:short)' 2>/dev/null)
          COMPREPLY=($(compgen -W "$branches" -- "$cur"))
          ;;
        *)
          COMPREPLY=($(compgen -W "-b --branch" -- "$cur"))
          ;;
      esac
      ;;
    init|completion)
      COMPREPLY=($(compgen -W "zsh bash fish" -- "$cur"))
      ;;
    delete)
      COMPREPLY=($(compgen -W "-f --force -k --keep-branch" -- "$cur"))
      ;;
    cleanup)
      COMPREPLY=($(compgen -W "-n --dry-run -f --force -k --keep-branch" -- "$cur"))
      ;;
    list)
      COMPREPLY=($(compgen -W "-v --verbose" -- "$cur"))
      ;;
  esac
  return 0
}

# Register completion for wt function
complete -F _wt_completions wt

wt() {
  # Check if we're in a git repo with .wt.yaml
  local repo_root
  repo_root=$(git rev-parse --show-toplevel 2>/dev/null)

  if [[ -z "$repo_root" ]]; then
    command wt "$@"
    return $?
  fi

  # Check for .wt.yaml in repo root or if we're in a worktree, check main repo
  local config_found=false
  if [[ -f "$repo_root/.wt.yaml" ]]; then
    config_found=true
  else
    local git_file="$repo_root/.git"
    if [[ -f "$git_file" ]]; then
      local gitdir=$(grep "^gitdir:" "$git_file" | cut -d' ' -f2)
      if [[ -n "$gitdir" ]]; then
        local main_repo=$(dirname $(dirname $(dirname "$gitdir")))
        if [[ -f "$main_repo/.wt.yaml" ]]; then
          config_found=true
          repo_root="$main_repo"
        fi
      fi
    fi
  fi

  if [[ "$config_found" != "true" ]]; then
    command wt "$@"
    return $?
  fi

  case "$1" in
    create)
      local output
      output=$(command wt "$@" 2>&1)
      local exit_code=$?

      if [[ $exit_code -eq 0 ]]; then
        echo "$output" | sed '$d'
        local target=$(echo "$output" | tail -1)
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
complete -c wt -n "__fish_use_subcommand" -a "cleanup" -d "Clean up merged worktrees"
complete -c wt -n "__fish_use_subcommand" -a "exit" -d "Return to the main repository"
complete -c wt -n "__fish_use_subcommand" -a "init" -d "Generate shell integration script"
complete -c wt -n "__fish_use_subcommand" -a "root" -d "Print the main repository root path"
complete -c wt -n "__fish_use_subcommand" -a "completion" -d "Generate shell completion script"
complete -c wt -n "__fish_use_subcommand" -a "version" -d "Print the version number"
complete -c wt -n "__fish_use_subcommand" -a "help" -d "Help about any command"

# Helper function to get worktree names
function __wt_worktrees
  set -l repo_root (git rev-parse --show-toplevel 2>/dev/null)
  if test -z "$repo_root"
    return
  end
  if test -f "$repo_root/.git"
    set -l gitdir (grep "^gitdir:" "$repo_root/.git" | cut -d' ' -f2)
    if test -n "$gitdir"
      set repo_root (dirname (dirname (dirname "$gitdir")))
    end
  end
  if test -f "$repo_root/.wt.yaml"
    set -l worktree_dir (grep "^worktree_dir:" "$repo_root/.wt.yaml" | cut -d' ' -f2 | tr -d '"' | tr -d "'")
    test -z "$worktree_dir"; and set worktree_dir "worktrees"
    if test -d "$repo_root/$worktree_dir"
      ls -1 "$repo_root/$worktree_dir" 2>/dev/null
    end
  end
end

# Worktree name completion for cd and delete
complete -c wt -n "__fish_seen_subcommand_from cd delete" -a "(__wt_worktrees)"

# Branch completion for create --branch
complete -c wt -n "__fish_seen_subcommand_from create" -s b -l branch -d "Use existing branch" -a "(git branch --format='%(refname:short)' 2>/dev/null)"

# Shell completion for init and completion commands
complete -c wt -n "__fish_seen_subcommand_from init completion" -a "zsh bash fish"

# Flags for delete
complete -c wt -n "__fish_seen_subcommand_from delete" -s f -l force -d "Force deletion"
complete -c wt -n "__fish_seen_subcommand_from delete" -s k -l keep-branch -d "Keep the associated branch"

# Flags for cleanup
complete -c wt -n "__fish_seen_subcommand_from cleanup" -s n -l dry-run -d "Show what would be deleted"
complete -c wt -n "__fish_seen_subcommand_from cleanup" -s f -l force -d "Skip confirmation"
complete -c wt -n "__fish_seen_subcommand_from cleanup" -s k -l keep-branch -d "Keep the associated branches"

# Flags for list
complete -c wt -n "__fish_seen_subcommand_from list" -s v -l verbose -d "Show detailed status"

function wt
  # Check if we're in a git repo
  set -l repo_root (git rev-parse --show-toplevel 2>/dev/null)

  if test -z "$repo_root"
    command wt $argv
    return $status
  end

  # Check for .wt.yaml
  set -l config_found false
  if test -f "$repo_root/.wt.yaml"
    set config_found true
  else
    set -l git_file "$repo_root/.git"
    if test -f "$git_file"
      set -l gitdir (grep "^gitdir:" "$git_file" | cut -d' ' -f2)
      if test -n "$gitdir"
        set -l main_repo (dirname (dirname (dirname "$gitdir")))
        if test -f "$main_repo/.wt.yaml"
          set config_found true
          set repo_root "$main_repo"
        end
      end
    end
  end

  if test "$config_found" != "true"
    command wt $argv
    return $status
  end

  switch $argv[1]
    case create
      set -l output (command wt $argv 2>&1)
      set -l exit_code $status

      if test $exit_code -eq 0
        echo "$output" | sed '$d'
        set -l target (echo "$output" | tail -1)
        if test -d "$target"
          cd "$target"
        else
          echo "$output"
        end
      else
        echo "$output"
      end
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
