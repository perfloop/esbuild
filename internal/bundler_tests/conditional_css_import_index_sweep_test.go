package bundler_tests

import (
	"testing"

	"github.com/evanw/esbuild/internal/bundler"
)

func BenchmarkConditionalCSSImportIndexFanoutSweep(b *testing.B) {
	configs := []struct {
		depth     int
		fullCount int
	}{
		{depth: 4, fullCount: 8},
		{depth: 4, fullCount: 64},
		{depth: 6, fullCount: 16},
		{depth: 6, fullCount: 256},
		{depth: 8, fullCount: 64},
		{depth: 8, fullCount: 1024},
	}
	bundles := make([]*bundler.Bundle, len(configs))
	wants := make([]string, len(configs))
	gots := make([]string, len(configs))
	for i, config := range configs {
		bundles[i] = makeConditionalCSSImportBundle(b, conditionalCSSImportIndexFanout(config.depth, config.fullCount))
		wants[i] = compileConditionalCSSImportBundle(b, bundles[i])
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j, bundle := range bundles {
			gots[j] = compileConditionalCSSImportBundle(b, bundle)
		}
	}
	b.StopTimer()

	for i, got := range gots {
		if got != wants[i] {
			b.Fatal("bundled CSS changed across repeated compiles")
		}
	}
}
