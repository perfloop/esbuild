package bundler_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/evanw/esbuild/internal/test"
)

// This has 64 non-redundant entries before the target conditions, so the
// final target imports exercise lookup through cssImportConditionsIndex.
func TestConditionalCSSImportIndexLargeGroupOutput(t *testing.T) {
	var entry strings.Builder
	for i := 0; i < 64; i++ {
		fmt.Fprintf(&entry, "@import \"https://example.com/shared.css\" layer(external-%d);\n", i)
	}
	entry.WriteString(`@import "https://example.com/shared.css" layer(external-target) supports(display: grid) screen;
@import "https://example.com/shared.css" layer(external-target) screen;
@import "https://example.com/shared.css" layer(external-target) supports(display: grid);
@import "https://example.com/shared.css" layer(external-target) supports(display: grid);
`)
	for i := 0; i < 64; i++ {
		fmt.Fprintf(&entry, "@import \"./shared.css\" layer(l%d);\n", i)
	}
	entry.WriteString(`@import "./shared.css" layer(target) supports(display: grid) screen;
@import "./shared.css" layer(target) screen;
@import "./shared.css" layer(target) supports(display: grid);
@import "./shared.css" layer(target) supports(display: grid);
`)

	got := compileConditionalCSSImportBundle(t, makeConditionalCSSImportBundle(t, map[string]string{
		"/entry.css":  entry.String(),
		"/shared.css": `.shared { color: black }`,
	}))

	var expected strings.Builder
	for i := 0; i < 64; i++ {
		fmt.Fprintf(&expected, "@import \"https://example.com/shared.css\" layer(external-%d);\n", i)
	}
	expected.WriteString(`@media screen {
  @supports (display: grid) {
    @layer external-target;
  }
}
@import "https://example.com/shared.css" layer(external-target) screen;
@import "https://example.com/shared.css" layer(external-target) supports(display: grid);

`)
	for i := 0; i < 64; i++ {
		fmt.Fprintf(&expected, `/* shared.css */
@layer l%d {
  .shared {
    color: black;
  }
}
`, i)
		if i+1 < 64 {
			expected.WriteByte('\n')
		}
	}
	expected.WriteString(`@media screen {
  @supports (display: grid) {
    @layer target;
  }
}

/* shared.css */
@media screen {
  @layer target {
    .shared {
      color: black;
    }
  }
}

/* shared.css */
@supports (display: grid) {
  @layer target {
    .shared {
      color: black;
    }
  }
}

/* entry.css */
`)
	test.AssertEqualWithDiff(t, got, expected.String())
}

func TestConditionalCSSImportIndexLargeNestedGroupOutput(t *testing.T) {
	files := map[string]string{"/shared.css": `.shared { color: black }`}
	var entry strings.Builder
	for i := 0; i < 64; i++ {
		fmt.Fprintf(&entry, "@import \"./outer-%d.css\" layer(outer-%d);\n", i, i)
		files[fmt.Sprintf("/outer-%d.css", i)] = fmt.Sprintf("@import \"./shared.css\" layer(inner-%d);\n", i)
	}
	for _, suffix := range []string{"full", "screen", "supports-a", "supports-b"} {
		fmt.Fprintf(&entry, "@import \"./outer-%s.css\" layer(outer-target);\n", suffix)
	}
	files["/outer-full.css"] = `@import "./shared.css" layer(inner-target) supports(display: grid) screen;`
	files["/outer-screen.css"] = `@import "./shared.css" layer(inner-target) screen;`
	files["/outer-supports-a.css"] = `@import "./shared.css" layer(inner-target) supports(display: grid);`
	files["/outer-supports-b.css"] = `@import "./shared.css" layer(inner-target) supports(display: grid);`
	files["/entry.css"] = entry.String()

	got := compileConditionalCSSImportBundle(t, makeConditionalCSSImportBundle(t, files))

	var expected strings.Builder
	for i := 0; i < 64; i++ {
		fmt.Fprintf(&expected, `/* shared.css */
@layer outer-%d {
  @layer inner-%d {
    .shared {
      color: black;
    }
  }
}

/* outer-%d.css */
@layer outer-%d;
`, i, i, i, i)
		if i+1 < 64 {
			expected.WriteByte('\n')
		}
	}
	expected.WriteString(`@layer outer-target {
  @media screen {
    @supports (display: grid) {
      @layer inner-target;
    }
  }
}

/* outer-full.css */
@layer outer-target;

/* shared.css */
@layer outer-target {
  @media screen {
    @layer inner-target {
      .shared {
        color: black;
      }
    }
  }
}

/* outer-screen.css */
@layer outer-target;

/* outer-supports-a.css */
@layer outer-target;

/* shared.css */
@layer outer-target {
  @supports (display: grid) {
    @layer inner-target {
      .shared {
        color: black;
      }
    }
  }
}

/* outer-supports-b.css */
@layer outer-target;

/* entry.css */
`)
	test.AssertEqualWithDiff(t, got, expected.String())
}
