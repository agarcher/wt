package shell

import "fmt"

// GenerateZsh generates the zsh shell integration script
func GenerateZsh() string {
	return `# wt shell integration for zsh
# Add this to your ~/.zshrc: eval "$(wt init zsh)"

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
