package shell

import (
	"strings"
	"testing"
)

func TestGenerateZsh(t *testing.T) {
	script := GenerateZsh()

	// Check for required elements
	requiredStrings := []string{
		"wt()",
		"git rev-parse --show-toplevel",
		"command wt",
		"create)",
		"exit)",
		"cd \"$target\"",
		"setup",
	}

	for _, s := range requiredStrings {
		if !strings.Contains(script, s) {
			t.Errorf("zsh script missing required string: %q", s)
		}
	}

	// Should NOT gate on .wt.yaml in wrapper function
	// Use \nwt() to avoid matching _wt() completion function
	wrapperStart := strings.Index(script, "\nwt() {")
	if wrapperStart == -1 {
		t.Fatal("zsh script missing wt() wrapper function")
	}
	wrapper := script[wrapperStart:]
	if strings.Contains(wrapper, ".wt.yaml") {
		t.Error("zsh wrapper function should not gate on .wt.yaml")
	}
}

func TestGenerateBash(t *testing.T) {
	script := GenerateBash()

	// Check for required elements
	requiredStrings := []string{
		"wt()",
		"git rev-parse --show-toplevel",
		"command wt",
		"create)",
		"exit)",
		"setup",
	}

	for _, s := range requiredStrings {
		if !strings.Contains(script, s) {
			t.Errorf("bash script missing required string: %q", s)
		}
	}

	// Should NOT gate on .wt.yaml in wrapper function
	wrapperStart := strings.Index(script, "\nwt() {")
	if wrapperStart == -1 {
		t.Fatal("bash script missing wt() wrapper function")
	}
	wrapper := script[wrapperStart:]
	if strings.Contains(wrapper, ".wt.yaml") {
		t.Error("bash wrapper function should not gate on .wt.yaml")
	}
}

func TestGenerateFish(t *testing.T) {
	script := GenerateFish()

	// Check for required elements
	requiredStrings := []string{
		"function wt",
		"git rev-parse --show-toplevel",
		"command wt",
		"case create",
		"case exit",
		"setup",
	}

	for _, s := range requiredStrings {
		if !strings.Contains(script, s) {
			t.Errorf("fish script missing required string: %q", s)
		}
	}

	// Should NOT gate on .wt.yaml in wrapper function
	// Use \nfunction wt\n to avoid matching __wt_worktrees
	wrapperStart := strings.Index(script, "\nfunction wt\n")
	if wrapperStart == -1 {
		t.Fatal("fish script missing wt wrapper function")
	}
	wrapper := script[wrapperStart:]
	if strings.Contains(wrapper, ".wt.yaml") {
		t.Error("fish wrapper function should not gate on .wt.yaml")
	}
}

func TestGenerate(t *testing.T) {
	tests := []struct {
		shell   string
		wantErr bool
	}{
		{"zsh", false},
		{"bash", false},
		{"fish", false},
		{"invalid", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			script, err := Generate(tt.shell)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if script == "" {
				t.Error("expected non-empty script")
			}
		})
	}
}

func TestShellScriptsHaveGlobalSetupCompletion(t *testing.T) {
	tests := []struct {
		name   string
		script string
		substr string // shell-specific substring to look for
	}{
		{"zsh", GenerateZsh(), "'--global[Configure global defaults]'"},
		{"bash", GenerateBash(), `"--global --personal --shared"`},
		{"fish", GenerateFish(), `"__fish_seen_subcommand_from setup" -l global`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.script, tt.substr) {
				t.Errorf("%s script missing global setup completion (expected %q)", tt.name, tt.substr)
			}
		})
	}
}

func TestShellScriptsHaveComments(t *testing.T) {
	shells := []string{"zsh", "bash", "fish"}

	for _, sh := range shells {
		t.Run(sh, func(t *testing.T) {
			script, _ := Generate(sh)
			if !strings.HasPrefix(script, "#") {
				t.Error("script should start with a comment")
			}
		})
	}
}
