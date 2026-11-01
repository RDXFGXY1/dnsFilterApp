package nettest

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

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

func TestCustomTestLimits(t *testing.T) {
	m := New()
	for _, tc := range []struct {
		options Options
		valid   bool
	}{
		{Options{Mode: "custom_data", LimitMB: 100}, true},
		{Options{Mode: "custom_data", LimitMB: 4}, false},
		{Options{Mode: "custom_data", LimitMB: 501}, false},
		{Options{Mode: "custom_time", DurationSeconds: 15}, true},
		{Options{Mode: "custom_time", DurationSeconds: 61}, false},
	} {
		snap, err := m.StartWithOptions(tc.options)
		if (err == nil) != tc.valid {
			t.Fatalf("options %+v: err=%v", tc.options, err)
		}
		if tc.valid {
			if snap.Mode != tc.options.Mode || snap.LimitMB != tc.options.LimitMB || snap.DurationSeconds != tc.options.DurationSeconds {
				t.Fatalf("wrong limits: %+v", snap)
			}
			m.Cancel(snap.ID)
		}
	}
}

func TestTimedTransferStopsAtDuration(t *testing.T) {
	m := &Manager{client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if err := req.Context().Err(); err != nil {
			return nil, err
		}
		if req.Method == http.MethodPost {
			_, _ = io.Copy(io.Discard, req.Body)
		}
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(bytes.NewReader(make([]byte, 64*1024)))}, nil
	})}}
	r := &run{}
	start := time.Now()
	if err := m.downloadTimed(context.Background(), r, start, 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if err := m.uploadTimed(context.Background(), r, start, 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	snap := copySnapshot(r)
	if snap.BytesDownloaded == 0 || snap.BytesUploaded == 0 || snap.Progress != 100 {
		t.Fatalf("timed result incomplete: %+v", snap)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatalf("timed transfer ran too long: %v", time.Since(start))
	}
}
