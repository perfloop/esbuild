package api_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/evanw/esbuild/pkg/api"
)

func TestFormatMessagesBatchMatchesIndividual(t *testing.T) {
	messages := make([]api.Message, 256)
	for i := range messages {
		messages[i] = api.Message{
			ID:         fmt.Sprintf("message-%d", i),
			PluginName: fmt.Sprintf("plugin-%d", i%3),
			Text:       fmt.Sprintf("diagnostic %d", i),
		}
		if i%2 == 0 {
			messages[i].Location = &api.Location{
				File:     fmt.Sprintf("file-%d.js", i),
				Line:     i + 1,
				Column:   i % 11,
				Length:   1,
				LineText: "let value = diagnostic",
			}
		}
		if i%3 == 0 {
			messages[i].Notes = []api.Note{{
				Text: fmt.Sprintf("note %d", i),
			}}
		}
	}

	opts := api.FormatMessagesOptions{Kind: api.WarningMessage, TerminalWidth: 80}
	want := make([]string, len(messages))
	for i, message := range messages {
		want[i] = api.FormatMessages([]api.Message{message}, opts)[0]
	}

	if got := api.FormatMessages(messages, opts); !reflect.DeepEqual(got, want) {
		t.Fatalf("batch formatting differed from formatting each message individually")
	}
}
