package sheet

import "testing"

func TestRenderLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
		wantEmpty bool
	}{
		{
			name:     "comment ignored",
			input:    "// this is a comment",
			wantEmpty: true,
		},
		{
			name:  "command with description",
			input: "git status > Show working tree status",
			want:  "  \033[32;1mgit status\033[0m\033[90m · \033[0m\033[3mShow working tree status\033[0m",
		},
		{
			name:  "plain line unchanged",
			input: "some plain text",
			want:  "some plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderLine(tt.input)
			if tt.wantEmpty && got != "" {
				t.Errorf("RenderLine(%q) = %q, want empty", tt.input, got)
			}
			if !tt.wantEmpty && got != tt.want {
				t.Errorf("RenderLine(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name       string
		lines      []string
		wantErrors int
	}{
		{
			name:       "valid format",
			lines:      []string{"cmd > desc", "other > something"},
			wantErrors: 0,
		},
		{
			name:       "invalid missing separator",
			lines:      []string{"cmd > desc", "no separator"},
			wantErrors: 1,
		},
		{
			name:       "empty and comment lines ignored",
			lines:      []string{"", "// comment", "cmd > desc"},
			wantErrors: 0,
		},
		{
			name:       "all invalid",
			lines:      []string{"invalid", "also wrong"},
			wantErrors: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateFormat(tt.lines)
			if len(got) != tt.wantErrors {
				t.Errorf("ValidateFormat() returned %d errors, want %d", len(got), tt.wantErrors)
			}
		})
	}
}