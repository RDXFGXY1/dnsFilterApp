package nettest

import (
	"bytes"
	"testing"
)

func TestLatencyCalculations(t *testing.T) {
	values := []float64{10, 20, 15}
	if got := average(values); got != 15 {
		t.Fatalf("average = %v", got)
	}
	if got := jitter(values); got != 7.5 {
		t.Fatalf("jitter = %v", got)
	}
}

func TestProgressReaderReportsCompletion(t *testing.T) {
	data := []byte("network-test")
	last := int64(0)
	reader := &progressReader{reader: bytes.NewReader(data), total: int64(len(data)), update: func(value int64) { last = value }}
	buf := make([]byte, len(data))
	if _, err := reader.Read(buf); err != nil {
		t.Fatal(err)
	}
	if last != int64(len(data)) {
		t.Fatalf("progress = %d", last)
	}
}

func TestStartNormalizesMode(t *testing.T) {
	m := New()
	run := m.Start("unexpected")
	defer m.Cancel(run.ID)
	if run.Mode != "quick" || run.Status != "running" {
		t.Fatalf("unexpected snapshot: %+v", run)
	}
}
