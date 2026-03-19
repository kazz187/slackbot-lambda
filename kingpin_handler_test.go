package slackbot

import (
	"testing"

	"github.com/alecthomas/kingpin/v2"
)

func TestExtractArgs(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "mention with command",
			text: "<@U12345> ask how does it work?",
			want: []string{"ask", "how", "does", "it", "work?"},
		},
		{
			name: "mention only",
			text: "<@U12345>",
			want: nil,
		},
		{
			name: "mention with spaces",
			text: "<@U12345>   help  ",
			want: []string{"help"},
		},
		{
			name: "no mention",
			text: "ask something",
			want: []string{"ask", "something"},
		},
		{
			name: "empty string",
			text: "",
			want: nil,
		},
		{
			name: "mention with flags",
			text: "<@UBOT> docs-update --force -t 0.5",
			want: []string{"docs-update", "--force", "-t", "0.5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractArgs(tt.text)
			if len(got) != len(tt.want) {
				t.Errorf("extractArgs(%q) = %v, want %v", tt.text, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractArgs(%q)[%d] = %q, want %q", tt.text, i, got[i], tt.want[i])
				}
			}
		})
	}
}

type testCommand struct {
	called   bool
	argValue *string
}

func (tc *testCommand) Register(app *kingpin.Application) {
	cmd := app.Command("greet", "say hello")
	tc.argValue = cmd.Arg("name", "name to greet").String()
	cmd.Action(func(pc *kingpin.ParseContext) error {
		tc.called = true
		return nil
	})
}

func TestKingpinHandler_AddCommand(t *testing.T) {
	kh := NewKingpinHandler(nil, "bot", "test bot")
	cmd := &testCommand{}
	kh.AddCommand(cmd)

	if len(kh.commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(kh.commands))
	}
}

func TestKingpinHandler_ParseAndDispatch(t *testing.T) {
	// Test that kingpin correctly parses and dispatches commands
	// without actually calling Slack API (slackCli is nil, writer won't flush)
	cmd := &testCommand{}

	app := kingpin.New("bot", "test bot")
	app.Terminate(nil)
	cmd.Register(app)

	_, err := app.Parse([]string{"greet", "world"})
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if !cmd.called {
		t.Error("command action was not called")
	}
	if *cmd.argValue != "world" {
		t.Errorf("arg = %q, want %q", *cmd.argValue, "world")
	}
}

func TestSlackWriter_Write(t *testing.T) {
	sw := newSlackWriter(nil, "C123", "ts123")

	n, err := sw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("wrote %d bytes, want 5", n)
	}

	n, err = sw.Write([]byte(" world"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 6 {
		t.Errorf("wrote %d bytes, want 6", n)
	}

	if sw.buffer.String() != "hello world" {
		t.Errorf("buffer = %q, want %q", sw.buffer.String(), "hello world")
	}
}
