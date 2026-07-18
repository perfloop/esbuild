package api

import (
	"fmt"
	"testing"

	"github.com/evanw/esbuild/internal/logger"
)

var convertMessagesBenchmarkResult []logger.Msg
var formatMessagesBenchmarkResult []string
var convertErrorsAndWarningsBenchmarkResult []logger.Msg

func benchmarkMessages(count int) []Message {
	messages := make([]Message, count)
	for i := range messages {
		messages[i] = Message{
			ID:         fmt.Sprintf("message-%d", i),
			PluginName: "benchmark",
			Text:       fmt.Sprintf("diagnostic %d", i),
		}
	}
	return messages
}

func BenchmarkConvertMessagesBatch256(b *testing.B) {
	messages := benchmarkMessages(256)
	result := convertMessagesToInternal(nil, logger.Error, messages)
	if len(result) != len(messages) || result[0].Data.Text == "" || result[len(result)-1].Data.Text == "" {
		b.Fatal("unexpected converted messages")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		convertMessagesBenchmarkResult = convertMessagesToInternal(nil, logger.Error, messages)
	}
}

func benchmarkFormatMessages(b *testing.B, count int) {
	messages := benchmarkMessages(count)
	opts := FormatMessagesOptions{Kind: ErrorMessage}
	result := FormatMessages(messages, opts)
	if len(result) != len(messages) || result[0] == "" || result[len(result)-1] == "" {
		b.Fatal("unexpected formatted messages")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatMessagesBenchmarkResult = FormatMessages(messages, opts)
	}
}

func BenchmarkFormatMessagesBatch1(b *testing.B) {
	benchmarkFormatMessages(b, 1)
}

func BenchmarkFormatMessagesBatch32(b *testing.B) {
	benchmarkFormatMessages(b, 32)
}

func BenchmarkFormatMessagesBatch256(b *testing.B) {
	benchmarkFormatMessages(b, 256)
}

func BenchmarkFormatMessagesBatch1024(b *testing.B) {
	benchmarkFormatMessages(b, 1024)
}

func BenchmarkConvertErrorsAndWarningsBatch256(b *testing.B) {
	errors := benchmarkMessages(128)
	warnings := benchmarkMessages(128)
	result := convertErrorsAndWarningsToInternal(errors, warnings)
	if len(result) != len(errors)+len(warnings) {
		b.Fatal("unexpected converted messages")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		convertErrorsAndWarningsBenchmarkResult = convertErrorsAndWarningsToInternal(errors, warnings)
	}
}
