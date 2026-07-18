package bundler_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/evanw/esbuild/internal/test"
)

func TestConditionalCSSImportIndexEdgeCases(t *testing.T) {
	got := compileConditionalCSSImportBundle(t, makeConditionalCSSImportBundle(t, map[string]string{
		"/entry.css": `
			@import "https://example.com/shared.css" layer(external) supports(display: grid) screen;
			@import "https://example.com/shared.css" layer(external) screen;
			@import "https://example.com/shared.css" layer(external) supports(display: grid);
			@import "./outer-a.css" layer(root) supports(display: grid) screen;
			@import "./outer-b.css" layer(root) screen;
			@import "./prefix-a.css" layer(prefix-root) supports(display: flex);
			@import "./prefix-b.css";
		`,
		"/outer-a.css": `
			@import "./shared.css" layer(inner) supports(color: red) print;
		`,
		"/outer-b.css": `
			@import "./shared.css" layer(inner) print;
		`,
		"/shared.css":        `.shared { color: black }`,
		"/prefix-a.css":      `@import "./shared-prefix.css" layer(prefix-inner) screen;`,
		"/prefix-b.css":      `@import "./shared-prefix.css";`,
		"/shared-prefix.css": `.prefix { color: blue }`,
	}))
	const expected = `@media screen {
  @supports (display: grid) {
    @layer external;
  }
}
@import "https://example.com/shared.css" layer(external) screen;
@import "https://example.com/shared.css" layer(external) supports(display: grid);
@media screen {
  @supports (display: grid) {
    @layer root {
      @media print {
        @supports (color: red) {
          @layer inner;
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

/* shared.css */
@media screen {
  @layer root {
    @media print {
      @layer inner {
        .shared {
          color: black;
        }
      }
    }
  }
}

/* outer-b.css */
@media screen {
  @layer root;
}
@supports (display: flex) {
  @layer prefix-root {
    @media screen {
      @layer prefix-inner;
    }
  }
}

/* prefix-a.css */
@supports (display: flex) {
  @layer prefix-root;
}

/* shared-prefix.css */
.prefix {
  color: blue;
}

/* prefix-b.css */

/* entry.css */
`
	test.AssertEqualWithDiff(t, got, expected)
}

func benchmarkConditionalCSSImportIndex(b *testing.B, files map[string]string) {
	bundle := makeConditionalCSSImportBundle(b, files)
	want := compileConditionalCSSImportBundle(b, bundle)

	b.ReportAllocs()
	b.ResetTimer()
	var got string
	for i := 0; i < b.N; i++ {
		got = compileConditionalCSSImportBundle(b, bundle)
	}
	b.StopTimer()

	if got != want {
		b.Fatal("bundled CSS changed across repeated compiles")
	}
}

func conditionalCSSImportIndexSmallGroups() map[string]string {
	files := make(map[string]string)
	var entry strings.Builder
	for group := 0; group < 4; group++ {
		for _, count := range []int{1, 2, 3, 63, 64, 65} {
			path := fmt.Sprintf("./shared-%d-%d.css", group, count)
			for i := 0; i < count; i++ {
				condition := ""
				switch group % 3 {
				case 0:
					condition = fmt.Sprintf("layer(distinct-%d-%d)", group, i)
				case 1:
					switch i % 3 {
					case 0:
						condition = fmt.Sprintf("layer(partial-%d) supports(display: grid) screen", group)
					case 1:
						condition = fmt.Sprintf("layer(partial-%d) screen", group)
					default:
						condition = fmt.Sprintf("layer(partial-%d) supports(display: grid)", group)
					}
				case 2:
					condition = fmt.Sprintf("layer(compatible-%d)", group)
				}
				fmt.Fprintf(&entry, "@import %q %s;\n", path, condition)
			}
			files["/"+path[2:]] = fmt.Sprintf(".shared-%d-%d { color: black }\n", group, count)
		}
	}
	files["/entry.css"] = entry.String()
	return files
}

func conditionalCSSImportIndexExternal(count int) map[string]string {
	var entry strings.Builder
	for i := 0; i < count; i++ {
		fmt.Fprintf(&entry, "@import \"https://example.com/shared.css\" layer(l%d);\n", i)
	}
	return map[string]string{"/entry.css": entry.String()}
}

func conditionalCSSImportIndexFanout(depth int, fullCount int) map[string]string {
	files := map[string]string{"/shared.css": `.shared { color: black }`}
	var entry strings.Builder

	writePath := func(prefix string, conditions func(int) string) {
		for level := 0; level < depth; level++ {
			path := fmt.Sprintf("/%s-%d.css", prefix, level)
			next := "./shared.css"
			if level+1 < depth {
				next = fmt.Sprintf("./%s-%d.css", prefix, level+1)
			}
			files[path] = fmt.Sprintf("@import %q %s;\n", next, conditions(level))
		}
		fmt.Fprintf(&entry, "@import %q %s;\n", "./"+prefix+"-0.css", conditions(0))
	}

	fullCondition := func(level int) string {
		return fmt.Sprintf("layer(l%d) supports(display: grid) screen", level)
	}
	for i := 0; i < fullCount; i++ {
		writePath(fmt.Sprintf("full-%d", i), fullCondition)
	}
	for bits := 0; bits < 1<<depth; bits++ {
		bits := bits
		writePath(fmt.Sprintf("partial-%d", bits), func(level int) string {
			if bits&(1<<level) == 0 {
				return fmt.Sprintf("layer(l%d) supports(display: grid)", level)
			}
			return fmt.Sprintf("layer(l%d) screen", level)
		})
	}

	files["/entry.css"] = entry.String()
	return files
}

func BenchmarkConditionalCSSImportIndexCoverage(b *testing.B) {
	b.Run("small-groups", func(b *testing.B) {
		benchmarkConditionalCSSImportIndex(b, conditionalCSSImportIndexSmallGroups())
	})
	b.Run("external-distinct-1000", func(b *testing.B) {
		benchmarkConditionalCSSImportIndex(b, conditionalCSSImportIndexExternal(1000))
	})
	b.Run("fanout-k8-r1024", func(b *testing.B) {
		benchmarkConditionalCSSImportIndex(b, conditionalCSSImportIndexFanout(8, 1024))
	})
}
