package slackbot

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

// Command is the interface that kingpin subcommands must implement.
type Command interface {
	// Register registers the command and its flags/args to the kingpin application.
	Register(app *kingpin.Application)
}

// KingpinHandler parses Slack app_mention messages as CLI commands using kingpin.
type KingpinHandler struct {
	commandName string
	commandDesc string
	commands    []Command
	slackCli    *slack.Client
}

// NewKingpinHandler creates a new KingpinHandler.
func NewKingpinHandler(slackCli *slack.Client, commandName, commandDesc string) *KingpinHandler {
	return &KingpinHandler{
		commandName: commandName,
		commandDesc: commandDesc,
		slackCli:    slackCli,
	}
}

// AddCommand registers a Command to be available for parsing.
func (kh *KingpinHandler) AddCommand(cmd Command) {
	kh.commands = append(kh.commands, cmd)
}

// HandleAppMention is an EventHandlerFunc that handles app_mention events.
func (kh *KingpinHandler) HandleAppMention(ctx context.Context, ev *slackevents.AppMentionEvent) error {
	args := extractArgs(ev.Text)

	threadTS := ev.ThreadTimeStamp
	if threadTS == "" {
		threadTS = ev.TimeStamp
	}

	app := kingpin.New(kh.commandName, kh.commandDesc)
	app.HelpFlag.Short('h')
	app.Terminate(nil)

	writer := newSlackWriter(kh.slackCli, ev.Channel, threadTS)
	app.UsageWriter(writer)
	app.ErrorWriter(writer)

	for _, cmd := range kh.commands {
		cmd.Register(app)
	}

	if _, err := app.Parse(args); err != nil {
		writer.Flush(ctx)
		return fmt.Errorf("failed to parse command: %w", err)
	}

	return writer.Flush(ctx)
}

// extractArgs removes the leading mention (e.g. "<@U12345>") from the message
// text and splits the remaining string into arguments.
func extractArgs(text string) []string {
	s := strings.TrimSpace(text)
	// Remove leading Slack mention like "<@U12345>"
	if strings.HasPrefix(s, "<@") {
		if idx := strings.Index(s, ">"); idx >= 0 {
			s = strings.TrimSpace(s[idx+1:])
		}
	}
	if s == "" {
		return nil
	}
	return strings.Fields(s)
}

// slackWriter is an io.Writer that buffers output and sends it to Slack.
type slackWriter struct {
	slackCli *slack.Client
	channel  string
	threadTS string
	buffer   bytes.Buffer
}

func newSlackWriter(slackCli *slack.Client, channel, threadTS string) *slackWriter {
	return &slackWriter{
		slackCli: slackCli,
		channel:  channel,
		threadTS: threadTS,
	}
}

func (sw *slackWriter) Write(p []byte) (n int, err error) {
	return sw.buffer.Write(p)
}

// Flush sends the buffered content to Slack as a code block.
// If the buffer is empty, it does nothing.
func (sw *slackWriter) Flush(ctx context.Context) error {
	content := sw.buffer.String()
	if content == "" {
		return nil
	}
	sw.buffer.Reset()

	msg := fmt.Sprintf("```\n%s\n```", content)
	_, _, err := sw.slackCli.PostMessageContext(ctx, sw.channel,
		slack.MsgOptionText(msg, false),
		slack.MsgOptionTS(sw.threadTS),
	)
	if err != nil {
		return fmt.Errorf("failed to post message to slack: %w", err)
	}
	return nil
}
