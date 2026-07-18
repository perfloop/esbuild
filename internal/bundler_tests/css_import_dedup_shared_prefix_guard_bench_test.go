package bundler_tests

import (
	"fmt"
	"strings"
	"testing"
)

func cssConditionalImportSharedPrefixBenchmarkFiles(importCount int) map[string]string {
	var entry strings.Builder
	entry.Grow(importCount * 70)
	for i := 0; i < importCount; i++ {
		entry.WriteString(`@import "./outer.css" layer(outer) supports(display: grid) screen;`)
		entry.WriteByte('\n')
	}
	entry.WriteString(".entry { color: black }\n")

	return map[string]string{
		"/entry.css": entry.String(),
		"/outer.css": `
			@import "./shared.css" layer(inner) supports(color: red) (min-width: 1px);
			.outer { color: blue }
		`,
		"/shared.css": `.shared { color: red }`,
	}
}

func BenchmarkCSSConditionalImportDedupSharedPrefix(b *testing.B) {
	for _, importCount := range []int{1000, 2000, 4000} {
		b.Run(fmt.Sprintf("%d", importCount), func(b *testing.B) {
			bundle := scanCSSConditionalImportFixture(b, cssConditionalImportSharedPrefixBenchmarkFiles(importCount))
			expected := compileCSSConditionalImportFixture(b, &bundle)
			for _, marker := range []string{"@layer outer", "@layer inner", ".outer", ".shared", ".entry"} {
				if !strings.Contains(expected, marker) {
					b.Fatalf("shared-prefix conditional import output is missing %q", marker)
				}
			}
			if actual := compileCSSConditionalImportFixture(b, &bundle); actual != expected {
				b.Fatal("repeated compilation changed shared-prefix conditional import output")
			}

			b.ResetTimer()
			outputBytes := 0
			for i := 0; i < b.N; i++ {
				outputBytes += len(compileCSSConditionalImportFixture(b, &bundle))
			}
			b.StopTimer()

			if outputBytes != b.N*len(expected) {
				b.Fatal("shared-prefix conditional import benchmark output was not consumed")
			}
		})
	}
}
