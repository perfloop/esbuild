package bundler_tests

import (
	"strings"
	"testing"
)

func cssConditionalImportRepresentativeGraphFiles() map[string]string {
	return map[string]string{
		"/entry.css": `
			@import "./base.css";
			@import "./theme.css" layer(theme) supports(display: grid);
			@import "./responsive.css" screen;
			@import "./print.css" print;
			.entry { color: black }
		`,
		"/base.css": `
			@import "./reset.css";
			@import "./shared.css";
			.base { color: gray }
		`,
		"/theme.css": `
			@import "./shared.css" layer(theme-inner);
			@import "./palette.css";
			.theme { color: blue }
		`,
		"/responsive.css": `
			@import "./shared.css" supports(container-type: inline-size);
			@import "./layout.css";
			.responsive { color: green }
		`,
		"/print.css": `
			@import "./shared.css";
			.print { color: black }
		`,
		"/reset.css":   `.reset { box-sizing: border-box }`,
		"/shared.css":  `.shared { color: red }`,
		"/palette.css": `.palette { color: purple }`,
		"/layout.css":  `.layout { display: grid }`,
	}
}

func BenchmarkCSSConditionalImportDedupRepresentativeGraph(b *testing.B) {
	bundle := scanCSSConditionalImportFixture(b, cssConditionalImportRepresentativeGraphFiles())
	expected := compileCSSConditionalImportFixture(b, &bundle)
	for _, marker := range []string{".base", ".theme", ".responsive", ".print", ".shared", ".entry"} {
		if !strings.Contains(expected, marker) {
			b.Fatalf("representative CSS graph output is missing %q", marker)
		}
	}
	if actual := compileCSSConditionalImportFixture(b, &bundle); actual != expected {
		b.Fatal("repeated compilation changed representative CSS graph output")
	}

	b.ResetTimer()
	outputBytes := 0
	for i := 0; i < b.N; i++ {
		outputBytes += len(compileCSSConditionalImportFixture(b, &bundle))
	}
	b.StopTimer()

	if outputBytes != b.N*len(expected) {
		b.Fatal("representative CSS graph benchmark output was not consumed")
	}
}
