package metrics_test

import (
	"runtime/metrics"
	"testing"
)

func TestReadAllMetricKinds(t *testing.T) {
	descs := metrics.All()
	if len(descs) == 0 {
		t.Fatal("metrics.All returned no descriptions")
	}

	samples := make([]metrics.Sample, len(descs))
	for i, desc := range descs {
		samples[i].Name = desc.Name
	}
	metrics.Read(samples)

	seen := map[metrics.ValueKind]bool{}
	for i, desc := range descs {
		value := samples[i].Value
		if got := value.Kind(); got != desc.Kind {
			t.Fatalf("Read(%q) kind = %d, want %d", desc.Name, got, desc.Kind)
		}
		checkMetricValue(t, desc.Name, value, desc.Kind)
		seen[desc.Kind] = true
	}

	for _, kind := range []metrics.ValueKind{
		metrics.KindUint64,
		metrics.KindFloat64,
		metrics.KindFloat64Histogram,
	} {
		if !seen[kind] {
			t.Fatalf("metrics.All did not include a metric of kind %d", kind)
		}
	}
}

func TestReadUnknownMetric(t *testing.T) {
	samples := []metrics.Sample{{Name: "/llgo/unknown:things"}}
	metrics.Read(samples)
	if got := samples[0].Value.Kind(); got != metrics.KindBad {
		t.Fatalf("Read unknown metric kind = %d, want %d", got, metrics.KindBad)
	}
}

func checkMetricValue(t *testing.T, name string, value metrics.Value, kind metrics.ValueKind) {
	t.Helper()

	switch kind {
	case metrics.KindUint64:
		_ = value.Uint64()
	case metrics.KindFloat64:
		_ = value.Float64()
	case metrics.KindFloat64Histogram:
		hist := value.Float64Histogram()
		if hist == nil {
			t.Fatalf("Read(%q) returned nil histogram", name)
		}
		if len(hist.Buckets) != len(hist.Counts)+1 {
			t.Fatalf("Read(%q) histogram buckets/counts lengths = %d/%d, want buckets = counts+1",
				name, len(hist.Buckets), len(hist.Counts))
		}
	default:
		t.Fatalf("Read(%q) returned unexpected kind %d", name, kind)
	}
}
