package sheet

import "testing"

func TestRenderLine(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		width     int
		want      string
		wantEmpty bool
	}{
		{
			name:      "empty line",
			input:     "",
			wantEmpty: true,
		},
		{
			name:      "whitespace-only line",
			input:     "   \t",
			wantEmpty: true,
		},
		{
			name:  "comment line",
			input: "# or: git push origin :refs/tags/v1.2.3",
			want:  "\x1b[90m\x1b[3m# or: git push origin :refs/tags/v1.2.3\x1b[0m",
		},
		{
			name:  "section header single word",
			input: "List",
			want:  "\x1b[36;1mList\x1b[0m",
		},
		{
			name:  "section header multi-word phrase",
			input: "Move / rename (avoid on published tags)",
			want:  "\x1b[36;1mMove / rename (avoid on published tags)\x1b[0m",
		},
		{
			name:  "command with description, no alignment",
			input: "git status # Show working tree status",
			want:  "\x1b[32;1mgit status\x1b[0m\x1b[90m · \x1b[0m\x1b[3mShow working tree status\x1b[0m",
		},
		{
			name:  "command with alignment padding in source",
			input: "git tag                      # all tags, alphabetical",
			want:  "\x1b[32;1mgit tag\x1b[0m\x1b[90m · \x1b[0m\x1b[3mall tags, alphabetical\x1b[0m",
		},
		{
			name:  "command padded to width when shorter",
			input: "git tag # all tags",
			width: 15,
			want:  "\x1b[32;1mgit tag\x1b[0m        \x1b[90m · \x1b[0m\x1b[3mall tags\x1b[0m",
		},
		{
			name:  "command at width takes no extra padding",
			input: "git rebase main # rebase",
			width: 15,
			want:  "\x1b[32;1mgit rebase main\x1b[0m\x1b[90m · \x1b[0m\x1b[3mrebase\x1b[0m",
		},
		{
			name:  "source padding stripped before width padding applied",
			input: "git tag        # all tags",
			width: 15,
			want:  "\x1b[32;1mgit tag\x1b[0m        \x1b[90m · \x1b[0m\x1b[3mall tags\x1b[0m",
		},
		{
			name:  "bare command without description",
			input: "git tag -a v1.2.3 -m \"v1.2.3\"",
			want:  "\x1b[32;1mgit tag -a v1.2.3 -m \"v1.2.3\"\x1b[0m",
		},
		{
			name:  "first hash wins inside command",
			input: "curl https://example.com/page#section # fetch with fragment",
			want:  "\x1b[32;1mcurl https://example.com/page\x1b[0m\x1b[90m · \x1b[0m\x1b[3msection # fetch with fragment\x1b[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderLine(tt.input, tt.width)
			if tt.wantEmpty && got != "" {
				t.Errorf("RenderLine(%q, %d) = %q, want empty", tt.input, tt.width, got)
			}
			if !tt.wantEmpty && got != tt.want {
				t.Errorf("RenderLine(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
			}
		})
	}
}

func TestMaxCommandWidth(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  int
	}{
		{
			name: "picks longest command-with-description",
			lines: []string{
				"git tag # short",
				"git rebase main # long",
				"git push origin --tags # longer",
			},
			want: len("git push origin --tags"),
		},
		{
			name: "ignores headers, comments, blanks, bare commands",
			lines: []string{
				"",
				"List",
				"# a comment",
				"git status",
				"git tag # described",
			},
			want: len("git tag"),
		},
		{
			name:  "no command-with-description lines",
			lines: []string{"List", "# comment", "git status"},
			want:  0,
		},
		{
			name:  "trims alignment padding from command",
			lines: []string{"git tag      # all tags"},
			want:  len("git tag"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaxCommandWidth(tt.lines)
			if got != tt.want {
				t.Errorf("MaxCommandWidth() = %d, want %d", got, tt.want)
			}
		})
	}
}
