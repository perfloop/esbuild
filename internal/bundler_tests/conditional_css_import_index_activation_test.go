package bundler_tests

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

type conditionalCSSImportIndexLayerName struct {
	name string
	hash uint32
}

func conditionalCSSImportIndexDescendingLayers(count int, prefix string, payloadBytes int) []string {
	payload := strings.Repeat("a", payloadBytes)
	layers := make([]conditionalCSSImportIndexLayerName, count)
	for i := range layers {
		name := fmt.Sprintf("%s-%08d-%s", prefix, i, payload)
		layers[i] = conditionalCSSImportIndexLayerName{
			name: name,
			hash: hashConditionalCSSImportIndexLayer(name),
		}
	}

	sort.Slice(layers, func(i int, j int) bool {
		if layers[i].hash != layers[j].hash {
			return layers[i].hash < layers[j].hash
		}
		return layers[i].name < layers[j].name
	})

	names := make([]string, len(layers))
	for i, layer := range layers {
		names[i] = layer.name
	}
	return names
}

func conditionalCSSImportIndexDescendingExternalInsertion(count int) map[string]string {
	var entry strings.Builder
	for _, layer := range conditionalCSSImportIndexDescendingLayers(count, "external-adversarial", 0) {
		fmt.Fprintf(&entry, "@import \"https://example.com/shared.css\" layer(%s);\n", layer)
	}
	return map[string]string{"/entry.css": entry.String()}
}

func conditionalCSSImportIndexLayerActivation(count int) map[string]string {
	var entry strings.Builder
	for i := 0; i < count; i++ {
		fmt.Fprintf(&entry, "@import \"https://example.com/layer-%d.css\" layer(layer-%d);\n", i, i)
	}
	return map[string]string{"/entry.css": entry.String()}
}

func conditionalCSSImportIndexHashPayload(count int, payloadBytes int) map[string]string {
	var entry strings.Builder
	for _, layer := range conditionalCSSImportIndexDescendingLayers(count, "payload", payloadBytes) {
		fmt.Fprintf(&entry, "@import \"./shared.css\" layer(%s);\n", layer)
	}
	return map[string]string{
		"/entry.css":  entry.String(),
		"/shared.css": ".shared { color: black }\n",
	}
}

func BenchmarkConditionalCSSImportIndexActivationThreshold(b *testing.B) {
	for _, count := range []int{127, 128, 129, 130} {
		b.Run(fmt.Sprintf("internal-descending-%d", count), func(b *testing.B) {
			benchmarkConditionalCSSImportIndex(b, conditionalCSSImportIndexDescendingInsertion(count))
		})
		b.Run(fmt.Sprintf("external-descending-%d", count), func(b *testing.B) {
			benchmarkConditionalCSSImportIndex(b, conditionalCSSImportIndexDescendingExternalInsertion(count))
		})
	}
	for _, count := range []int{1023, 1024, 1025, 1026} {
		b.Run(fmt.Sprintf("layer-%d", count), func(b *testing.B) {
			benchmarkConditionalCSSImportIndex(b, conditionalCSSImportIndexLayerActivation(count))
		})
	}
}

func BenchmarkConditionalCSSImportIndexHashPayload(b *testing.B) {
	for _, sample := range []struct {
		name         string
		count        int
		payloadBytes int
	}{
		{name: "16B-129", count: 129, payloadBytes: 16},
		{name: "1KiB-129", count: 129, payloadBytes: 1 << 10},
		{name: "16KiB-129", count: 129, payloadBytes: 16 << 10},
		{name: "16B-1000", count: 1000, payloadBytes: 16},
	} {
		b.Run(sample.name, func(b *testing.B) {
			benchmarkConditionalCSSImportIndex(b, conditionalCSSImportIndexHashPayload(sample.count, sample.payloadBytes))
		})
	}
}
