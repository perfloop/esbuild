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

func scanCSSConditionalImportFixture(t testing.TB, files map[string]string) bundler.Bundle {
	t.Helper()

	log := logger.NewDeferLog(logger.DeferLogNoVerboseOrDebug, nil)
	bundle := bundler.ScanBundle(
		config.BuildCall,
		log,
		fs.MockFS(files, fs.MockUnix, "/"),
		cache.MakeCacheSet(),
		[]bundler.EntryPoint{{InputPath: "/entry.css"}},
		config.Options{
			Mode:                config.ModeBundle,
			AbsOutputFile:       "/out.css",
			OmitRuntimeForTests: true,
		},
		nil,
	)
	if msgs := log.Done(); len(msgs) != 0 {
		t.Fatalf("unexpected scan messages: %v", msgs)
	}
	return bundle
}

func compileCSSConditionalImportFixture(t testing.TB, bundle *bundler.Bundle) string {
	t.Helper()

	log := logger.NewDeferLog(logger.DeferLogNoVerboseOrDebug, nil)
	results, _ := bundle.Compile(log, nil, nil, linker.Link)
	if msgs := log.Done(); len(msgs) != 0 {
		t.Fatalf("unexpected compile messages: %v", msgs)
	}
	if len(results) != 1 {
		t.Fatalf("expected one output file, got %d", len(results))
	}
	return string(results[0].Contents)
}

func TestCSSConditionalImportDedupConditionPrefixes(t *testing.T) {
	bundle := scanCSSConditionalImportFixture(t, map[string]string{
		"/entry.css": `
			@import "./exact.css" layer(alpha) supports(display: grid) screen;
			@import "./exact.css" layer(alpha) supports(display: grid) screen;
			@import "./supports-a.css" layer(beta) supports(display: grid) screen;
			@import "./supports-b.css" layer(beta) screen;
			@import "./media-a.css" layer(gamma) supports(display: grid) screen;
			@import "./media-b.css" layer(gamma) supports(display: grid);
		`,
		"/exact.css": `
			@import "./shared.css" layer(inner) supports(color: red) (min-width: 1px);
		`,
		"/supports-a.css": `
			@import "./shared.css" layer(inner) supports(color: red) (min-width: 1px);
		`,
		"/supports-b.css": `
			@import "./shared.css" layer(inner) supports(color: red) (min-width: 1px);
		`,
		"/media-a.css": `
			@import "./shared.css" layer(inner) supports(color: red) (min-width: 1px);
		`,
		"/media-b.css": `
			@import "./shared.css" layer(inner) supports(color: red) (min-width: 1px);
		`,
		"/shared.css": `.shared { color: red }`,
	})

	test.AssertEqualWithDiff(t, compileCSSConditionalImportFixture(t, &bundle), `@media screen {
  @supports (display: grid) {
    @layer alpha {
      @media (min-width: 1px) {
        @supports (color: red) {
          @layer inner;
        }
      }
    }
  }
}
@media screen {
  @supports (display: grid) {
    @layer alpha;
  }
}

/* shared.css */
@media screen {
  @supports (display: grid) {
    @layer alpha {
      @media (min-width: 1px) {
        @supports (color: red) {
          @layer inner {
            .shared {
              color: red;
            }
          }
        }
      }
    }
  }
}

/* exact.css */
@media screen {
  @supports (display: grid) {
    @layer alpha;
  }
}
@media screen {
  @supports (display: grid) {
    @layer beta {
      @media (min-width: 1px) {
        @supports (color: red) {
          @layer inner;
        }
      }
    }
  }
}

/* supports-a.css */
@media screen {
  @supports (display: grid) {
    @layer beta;
  }
}

/* shared.css */
@media screen {
  @layer beta {
    @media (min-width: 1px) {
      @supports (color: red) {
        @layer inner {
          .shared {
            color: red;
          }
        }
      }
    }
  }
}

/* supports-b.css */
@media screen {
  @layer beta;
}
@media screen {
  @supports (display: grid) {
    @layer gamma {
      @media (min-width: 1px) {
        @supports (color: red) {
          @layer inner;
        }
      }
    }
  }
}

/* media-a.css */
@media screen {
  @supports (display: grid) {
    @layer gamma;
  }
}

/* shared.css */
@supports (display: grid) {
  @layer gamma {
    @media (min-width: 1px) {
      @supports (color: red) {
        @layer inner {
          .shared {
            color: red;
          }
        }
      }
    }
  }
}

/* media-b.css */
@supports (display: grid) {
  @layer gamma;
}

/* entry.css */
`)
}

func cssConditionalImportBenchmarkFiles(importCount int) map[string]string {
	var entry strings.Builder
	entry.Grow(importCount * 40)
	for i := 0; i < importCount; i++ {
		fmt.Fprintf(&entry, "@import \"./shared.css\" layer(l%04d);\n", i)
	}
	entry.WriteString(".entry { color: black }\n")

	return map[string]string{
		"/entry.css":  entry.String(),
		"/shared.css": `.shared { color: red }`,
	}
}

func BenchmarkCSSConditionalImportDedup(b *testing.B) {
	for _, importCount := range []int{1000, 2000, 4000} {
		b.Run(fmt.Sprintf("%d", importCount), func(b *testing.B) {
			bundle := scanCSSConditionalImportFixture(b, cssConditionalImportBenchmarkFiles(importCount))
			expected := compileCSSConditionalImportFixture(b, &bundle)
			if !strings.Contains(expected, "@layer l0000") || !strings.Contains(expected, fmt.Sprintf("@layer l%04d", importCount-1)) {
				b.Fatal("conditional imports were not retained in benchmark output")
			}
			if actual := compileCSSConditionalImportFixture(b, &bundle); actual != expected {
				b.Fatal("repeated compilation changed conditional import output")
			}

			b.ResetTimer()
			outputBytes := 0
			for i := 0; i < b.N; i++ {
				outputBytes += len(compileCSSConditionalImportFixture(b, &bundle))
			}
			b.StopTimer()

			if outputBytes != b.N*len(expected) {
				b.Fatal("conditional import benchmark output was not consumed")
			}
		})
	}
}
