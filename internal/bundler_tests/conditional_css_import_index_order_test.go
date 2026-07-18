package bundler_tests

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/evanw/esbuild/internal/css_ast"
	"github.com/evanw/esbuild/internal/css_lexer"
)

func hashConditionalCSSImportIndexLayer(name string) uint32 {
	children := []css_ast.Token{{Kind: css_lexer.TIdent, Text: name}}
	condition := css_ast.ImportConditions{Layers: []css_ast.Token{{
		Kind:     css_lexer.TFunction,
		Text:     "layer",
		Children: &children,
	}}}
	hash := css_ast.HashTokens(0, condition.Layers)
	hash = css_ast.HashTokens(hash, condition.Supports)
	return css_ast.HashMediaQueries(hash, condition.Queries)
}

func conditionalCSSImportIndexDescendingInsertion(count int) map[string]string {
	type layerName struct {
		name string
		hash uint32
	}

	layers := make([]layerName, count)
	for i := range layers {
		name := fmt.Sprintf("adversarial-%d", i)
		layers[i] = layerName{name: name, hash: hashConditionalCSSImportIndexLayer(name)}
	}

	// The linker processes duplicate imports backward. Ascending source hashes
	// therefore force descending hashes into the index, which inserted every new
	// hash at the front of the old sorted slice.
	sort.Slice(layers, func(i int, j int) bool {
		if layers[i].hash != layers[j].hash {
			return layers[i].hash < layers[j].hash
		}
		return layers[i].name < layers[j].name
	})

	var entry strings.Builder
	for _, layer := range layers {
		fmt.Fprintf(&entry, "@import \"./shared.css\" layer(%s);\n", layer.name)
	}
	return map[string]string{
		"/entry.css":  entry.String(),
		"/shared.css": ".shared { color: black }",
	}
}

func BenchmarkConditionalCSSImportIndexDescendingInsertion(b *testing.B) {
	for _, count := range []int{127, 128, 129, 130, 1000, 4000, 16000} {
		b.Run(fmt.Sprintf("%d", count), func(b *testing.B) {
			benchmarkConditionalCSSImportIndex(b, conditionalCSSImportIndexDescendingInsertion(count))
		})
	}
}
