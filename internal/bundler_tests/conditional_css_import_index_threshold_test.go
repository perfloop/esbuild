package bundler_tests

import (
	"fmt"
	"strings"
	"testing"
)

func conditionalCSSImportIndexThresholdInternal(count int, distinct bool) map[string]string {
	var entry strings.Builder
	for i := 0; i < count; i++ {
		layer := "shared"
		if distinct {
			layer = fmt.Sprintf("l%d", i)
		}
		fmt.Fprintf(&entry, "@import \"./shared.css\" layer(%s);\n", layer)
	}
	return map[string]string{
		"/entry.css":  entry.String(),
		"/shared.css": `.shared { color: black }`,
	}
}

func conditionalCSSImportIndexThresholdExternal(count int) map[string]string {
	var entry strings.Builder
	for i := 0; i < count; i++ {
		fmt.Fprintf(&entry, "@import \"https://example.com/shared.css\" layer(l%d);\n", i)
	}
	return map[string]string{"/entry.css": entry.String()}
}

func BenchmarkConditionalCSSImportIndexThreshold(b *testing.B) {
	for _, count := range []int{63, 64, 65, 66} {
		b.Run(fmt.Sprintf("internal-distinct-%d", count), func(b *testing.B) {
			benchmarkConditionalCSSImportIndex(b, conditionalCSSImportIndexThresholdInternal(count, true))
		})
		b.Run(fmt.Sprintf("external-distinct-%d", count), func(b *testing.B) {
			benchmarkConditionalCSSImportIndex(b, conditionalCSSImportIndexThresholdExternal(count))
		})
	}
	b.Run("internal-compatible-66", func(b *testing.B) {
		benchmarkConditionalCSSImportIndex(b, conditionalCSSImportIndexThresholdInternal(66, false))
	})
}
