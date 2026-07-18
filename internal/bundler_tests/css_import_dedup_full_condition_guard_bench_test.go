package bundler_tests

import (
	"fmt"
	"strings"
	"testing"
)

// cssConditionalImportFullConditionBenchmarkFiles creates every combination of
// supports-only and media-only conditions at each depth. The alternating probe
// chains then put an incompatible newest condition before each compatible probe,
// which exercises the full-condition fallback instead of the direct latest-entry
// match.
func cssConditionalImportFullConditionBenchmarkFiles(depth int, probeCount int) map[string]string {
	files := map[string]string{
		"/shared.css": `.shared { color: red }`,
	}
	var entry strings.Builder

	for index := 0; index < 2; index++ {
		condition := "screen"
		if index == 0 {
			condition = "supports(display: common)"
		}
		fmt.Fprintf(&entry, "@import \"./tree-01-%04d.css\" layer(shared) %s;\n", index, condition)
	}

	for level := 1; level <= depth; level++ {
		for index := 0; index < 1<<level; index++ {
			path := fmt.Sprintf("/tree-%02d-%04d.css", level, index)
			if level == depth {
				files[path] = `@import "./shared.css";`
				continue
			}
			files[path] = fmt.Sprintf(`
				@import "./tree-%02d-%04d.css" layer(shared) supports(display: common);
				@import "./tree-%02d-%04d.css" layer(shared) screen;
			`, level+1, index*2, level+1, index*2+1)
		}
	}

	for i := 0; i < probeCount; i++ {
		for _, name := range []string{"incompatible", "probe"} {
			for level := 1; level <= depth; level++ {
				path := fmt.Sprintf("/%s-%03d-%02d.css", name, i, level)
				if level == depth {
					layer := "shared"
					if name == "incompatible" {
						layer = fmt.Sprintf("incompatible-%03d", i)
					}
					files[path] = fmt.Sprintf(`@import "./shared.css" layer(%s) supports(display: %s-%03d) screen;`, layer, name, i)
					continue
				}
				files[path] = fmt.Sprintf(`@import "./%s-%03d-%02d.css" layer(shared) supports(display: common) screen;`, name, i, level+1)
			}
		}
		fmt.Fprintf(&entry, `@import "./incompatible-%03d-01.css" layer(incompatible-%03d) supports(display: common) screen;`+"\n", i, i)
		fmt.Fprintf(&entry, `@import "./probe-%03d-01.css" layer(shared) supports(display: common) screen;`+"\n", i)
	}
	entry.WriteString(".entry { color: black }\n")
	files["/entry.css"] = entry.String()
	return files
}

func BenchmarkCSSConditionalImportDedupFullCondition(b *testing.B) {
	const depth = 12
	bundle := scanCSSConditionalImportFixture(b, cssConditionalImportFullConditionBenchmarkFiles(depth, 1000))
	expected := compileCSSConditionalImportFixture(b, &bundle)
	if sharedCopies := strings.Count(expected, ".shared {"); sharedCopies < 1<<depth {
		b.Fatalf("full-condition benchmark retained only %d of %d distinct shared imports", sharedCopies, 1<<depth)
	}
	for _, marker := range []string{"@supports (display: common)", "@supports (display: incompatible-000)", "@supports (display: probe-000)", "@media screen", ".entry"} {
		if !strings.Contains(expected, marker) {
			b.Fatalf("full-condition conditional import output is missing %q", marker)
		}
	}
	if actual := compileCSSConditionalImportFixture(b, &bundle); actual != expected {
		b.Fatal("repeated compilation changed full-condition conditional import output")
	}

	b.ReportAllocs()
	b.ResetTimer()
	outputBytes := 0
	for i := 0; i < b.N; i++ {
		outputBytes += len(compileCSSConditionalImportFixture(b, &bundle))
	}
	b.StopTimer()

	if outputBytes != b.N*len(expected) {
		b.Fatal("full-condition conditional import benchmark output was not consumed")
	}
}
