package bundler_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/evanw/esbuild/internal/bundler"
	"github.com/evanw/esbuild/internal/cache"
	"github.com/evanw/esbuild/internal/config"
	"github.com/evanw/esbuild/internal/fs"
	"github.com/evanw/esbuild/internal/linker"
	"github.com/evanw/esbuild/internal/logger"
	"github.com/evanw/esbuild/internal/test"
)

func makeConditionalCSSImportBundle(t testing.TB, files map[string]string) *bundler.Bundle {
	t.Helper()

	log := logger.NewDeferLog(logger.DeferLogNoVerboseOrDebug, nil)
	bundle := bundler.ScanBundle(config.BuildCall, log, fs.MockFS(files, fs.MockUnix, "/"),
		cache.MakeCacheSet(), []bundler.EntryPoint{{InputPath: "/entry.css"}}, config.Options{
			Mode:                config.ModeBundle,
			AbsOutputFile:       "/out.css",
			OmitRuntimeForTests: true,
		}, nil)
	if msgs := log.Done(); len(msgs) != 0 {
		t.Fatalf("unexpected scan log: %v", msgs)
	}
	return &bundle
}

func compileConditionalCSSImportBundle(t testing.TB, bundle *bundler.Bundle) string {
	t.Helper()

	log := logger.NewDeferLog(logger.DeferLogNoVerboseOrDebug, nil)
	results, _ := bundle.Compile(log, nil, nil, linker.Link)
	if msgs := log.Done(); len(msgs) != 0 {
		t.Fatalf("unexpected compile log: %v", msgs)
	}
	if len(results) != 1 {
		t.Fatalf("expected one output file, got %d", len(results))
	}
	return string(results[0].Contents)
}

func TestConditionalCSSImportDeduplication(t *testing.T) {
	got := compileConditionalCSSImportBundle(t, makeConditionalCSSImportBundle(t, map[string]string{
		"/entry.css": `
			@import "./outer-a.css" layer(root) supports(display: grid) screen;
			@import "./outer-b.css" layer(root) supports(display: grid) screen;
			@import "./outer-a.css" layer(root) supports(display: grid) screen;
		`,
		"/outer-a.css": `
			@import "./shared.css" layer(alpha) supports(color: red) print;
		`,
		"/outer-b.css": `
			@import "./shared.css" layer(beta) supports(color: blue) screen;
		`,
		"/shared.css": `.shared { color: black }`,
	}))

	const expected = `@media screen {
  @supports (display: grid) {
    @layer root {
      @media print {
        @supports (color: red) {
          @layer alpha;
        }
      }
    }
  }
}
@media screen {
  @supports (display: grid) {
    @layer root;
  }
}

/* shared.css */
@media screen {
  @supports (display: grid) {
    @layer root {
      @media screen {
        @supports (color: blue) {
          @layer beta {
            .shared {
              color: black;
            }
          }
        }
      }
    }
  }
}

/* outer-b.css */
@media screen {
  @supports (display: grid) {
    @layer root;
  }
}

/* shared.css */
@media screen {
  @supports (display: grid) {
    @layer root {
      @media print {
        @supports (color: red) {
          @layer alpha {
            .shared {
              color: black;
            }
          }
        }
      }
    }
  }
}

/* outer-a.css */
@media screen {
  @supports (display: grid) {
    @layer root;
  }
}

/* entry.css */
`
	test.AssertEqualWithDiff(t, got, expected)
}

func conditionalCSSImportDeduplicationFiles(count int, distinctLayers bool) map[string]string {
	var entry strings.Builder
	for i := 0; i < count; i++ {
		layerName := "shared"
		if distinctLayers {
			layerName = fmt.Sprintf("l%d", i)
		}
		fmt.Fprintf(&entry, "@import \"./shared.css\" layer(%s);\n", layerName)
	}
	return map[string]string{
		"/entry.css":  entry.String(),
		"/shared.css": ".shared { color: black }",
	}
}

func benchmarkConditionalCSSImportDeduplication(b *testing.B, count int, distinctLayers bool) {
	benchmarkConditionalCSSImportIndex(b, conditionalCSSImportDeduplicationFiles(count, distinctLayers))
}

func BenchmarkConditionalCSSImportDeduplication(b *testing.B) {
	for _, count := range []int{1000, 2000, 4000} {
		b.Run(fmt.Sprintf("%d", count), func(b *testing.B) {
			benchmarkConditionalCSSImportDeduplication(b, count, true)
		})
	}

	b.Run("compatible-4000", func(b *testing.B) {
		benchmarkConditionalCSSImportDeduplication(b, 4000, false)
	})
}
