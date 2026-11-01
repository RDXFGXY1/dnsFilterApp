package nettest

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	quickDownloadBytes = int64(5 * 1024 * 1024)
	quickUploadBytes   = int64(2 * 1024 * 1024)
	fullDownloadBytes  = int64(25 * 1024 * 1024)
	fullUploadBytes    = int64(5 * 1024 * 1024)
)

type Sample struct {
	At       int64   `json:"at_ms"`
	PingMS   float64 `json:"ping_ms,omitempty"`
	DownMbps float64 `json:"download_mbps,omitempty"`
	UpMbps   float64 `json:"upload_mbps,omitempty"`
}

type Snapshot struct {
	ID              string    `json:"id"`
	Status          string    `json:"status"`
	Phase           string    `json:"phase"`
	Mode            string    `json:"mode"`
	LimitMB         int       `json:"limit_mb,omitempty"`
	DurationSeconds int       `json:"duration_seconds,omitempty"`
	Server          string    `json:"server"`
	Progress        int       `json:"progress"`
	PingMS          float64   `json:"ping_ms"`
	JitterMS        float64   `json:"jitter_ms"`
	PacketLoss      float64   `json:"packet_loss"`
	DownloadMbps    float64   `json:"download_mbps"`
	UploadMbps      float64   `json:"upload_mbps"`
	BytesDownloaded int64     `json:"bytes_downloaded"`
	BytesUploaded   int64     `json:"bytes_uploaded"`
	Samples         []Sample  `json:"samples"`
	Error           string    `json:"error,omitempty"`
	StartedAt       time.Time `json:"started_at"`
	FinishedAt      time.Time `json:"finished_at,omitempty"`
}

type Options struct {
	Mode            string `json:"mode"`
	LimitMB         int    `json:"limit_mb"`
	DurationSeconds int    `json:"duration_seconds"`
}

type run struct {
	mu     sync.RWMutex
	snap   Snapshot
	cancel context.CancelFunc
}

type Manager struct {
	mu       sync.RWMutex
	runs     map[string]*run
	client   *http.Client
	onFinish func(Snapshot)
}

func (m *Manager) SetOnFinish(callback func(Snapshot)) { m.onFinish = callback }

func New() *Manager {
	return &Manager{runs: make(map[string]*run), client: &http.Client{Timeout: 60 * time.Second}}
}

func (m *Manager) Start(mode string) Snapshot {
	result, _ := m.StartWithOptions(Options{Mode: mode})
	return result
}

func (m *Manager) StartWithOptions(options Options) (Snapshot, error) {
	switch options.Mode {
	case "full", "quick":
		options.LimitMB, options.DurationSeconds = 0, 0
	case "custom_data":
		if options.LimitMB < 5 || options.LimitMB > 500 {
			return Snapshot{}, errors.New("Data limit must be between 5 and 500 MB")
		}
		options.DurationSeconds = 0
	case "custom_time":
		if options.DurationSeconds < 5 || options.DurationSeconds > 60 {
			return Snapshot{}, errors.New("Test duration must be between 5 and 60 seconds")
		}
		options.LimitMB = 0
	default:
		options = Options{Mode: "quick"}
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := &run{snap: Snapshot{ID: newID(), Status: "running", Phase: "latency", Mode: options.Mode, LimitMB: options.LimitMB, DurationSeconds: options.DurationSeconds, Server: "Cloudflare", StartedAt: time.Now().UTC()}, cancel: cancel}
	m.mu.Lock()
	for _, old := range m.runs {
		old.mu.RLock()
		running := old.snap.Status == "running"
		old.mu.RUnlock()
		if running {
			old.cancel()
		}
	}
	for id, old := range m.runs {
		old.mu.RLock()
		expired := time.Since(old.snap.StartedAt) > time.Hour
		old.mu.RUnlock()
		if expired {
			delete(m.runs, id)
		}
	}
	m.runs[r.snap.ID] = r
	m.mu.Unlock()
	go m.execute(ctx, r)
	return copySnapshot(r), nil
}

func (m *Manager) Get(id string) (Snapshot, bool) {
	m.mu.RLock()
	r, ok := m.runs[id]
	m.mu.RUnlock()
	if !ok {
		return Snapshot{}, false
	}
	return copySnapshot(r), true
}

func (m *Manager) Cancel(id string) bool {
	m.mu.RLock()
	r, ok := m.runs[id]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	r.cancel()
	return true
}

func copySnapshot(r *run) Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s := r.snap
	s.Samples = append([]Sample(nil), r.snap.Samples...)
	return s
}

func (m *Manager) execute(ctx context.Context, r *run) {
	defer func() {
		if m.onFinish != nil {
			if result := copySnapshot(r); result.Status != "running" {
				m.onFinish(result)
			}
		}
	}()
	latencies := make([]float64, 0, 10)
	lost := 0
	start := time.Now()
	for i := 0; i < 10; i++ {
		if ctx.Err() != nil {
			finish(r, "cancelled", "")
			return
		}
		began := time.Now()
		conn, err := (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "tcp", "speed.cloudflare.com:443")
		if err != nil {
			lost++
		} else {
			_ = conn.Close()
			ms := float64(time.Since(began).Microseconds()) / 1000
			latencies = append(latencies, ms)
			r.mu.Lock()
			r.snap.Samples = append(r.snap.Samples, Sample{At: time.Since(start).Milliseconds(), PingMS: round(ms)})
			r.snap.PingMS = round(average(latencies))
			r.snap.JitterMS = round(jitter(latencies))
			r.snap.PacketLoss = float64(lost) / float64(i+1) * 100
			r.snap.Progress = (i + 1) * 2
			r.mu.Unlock()
		}
		select {
		case <-ctx.Done():
			finish(r, "cancelled", "")
			return
		case <-time.After(250 * time.Millisecond):
		}
	}
	r.mu.Lock()
	r.snap.PacketLoss = round(float64(lost) / 10 * 100)
	r.mu.Unlock()
	if len(latencies) == 0 {
		finish(r, "failed", "The latency server could not be reached.")
		return
	}

	downBytes, upBytes := quickDownloadBytes, quickUploadBytes
	if r.snap.Mode == "full" {
		downBytes, upBytes = fullDownloadBytes, fullUploadBytes
	} else if r.snap.Mode == "custom_data" {
		downBytes = int64(r.snap.LimitMB) * 1024 * 1024 * 4 / 5
		upBytes = int64(r.snap.LimitMB)*1024*1024 - downBytes
	} else if r.snap.Mode == "custom_time" {
		if err := m.downloadTimed(ctx, r, start, time.Duration(r.snap.DurationSeconds)*time.Second/2); err != nil {
			finishFromError(r, ctx, "Download test failed: "+err.Error())
			return
		}
		if err := m.uploadTimed(ctx, r, start, time.Duration(r.snap.DurationSeconds)*time.Second/2); err != nil {
			finishFromError(r, ctx, "Upload test failed: "+err.Error())
			return
		}
		finish(r, "complete", "")
		return
	}
	if err := m.download(ctx, r, start, downBytes); err != nil {
		finishFromError(r, ctx, "Download test failed: "+err.Error())
		return
	}
	if err := m.upload(ctx, r, start, upBytes); err != nil {
		finishFromError(r, ctx, "Upload test failed: "+err.Error())
		return
	}
	finish(r, "complete", "")
}

func (m *Manager) download(ctx context.Context, r *run, runStart time.Time, size int64) error {
	r.mu.Lock()
	r.snap.Phase, r.snap.Progress = "download", 20
	r.mu.Unlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://speed.cloudflare.com/__down?bytes="+itoa(size), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept-Encoding", "identity")
	started := time.Now()
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New(resp.Status)
	}
	buf := make([]byte, 64*1024)
	var total int64
	lastSample := time.Time{}
	for {
		n, readErr := resp.Body.Read(buf)
		total += int64(n)
		now := time.Now()
		elapsed := now.Sub(started).Seconds()
		if elapsed > 0 && (lastSample.IsZero() || now.Sub(lastSample) >= 200*time.Millisecond || readErr == io.EOF) {
			mbps := float64(total*8) / elapsed / 1_000_000
			r.mu.Lock()
			r.snap.DownloadMbps = round(mbps)
			r.snap.BytesDownloaded = total
			r.snap.Progress = 20 + int(math.Min(1, float64(total)/float64(size))*45)
			r.snap.Samples = append(r.snap.Samples, Sample{At: time.Since(runStart).Milliseconds(), DownMbps: round(mbps)})
			r.mu.Unlock()
			lastSample = now
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return nil
}

func (m *Manager) upload(ctx context.Context, r *run, runStart time.Time, size int64) error {
	r.mu.Lock()
	r.snap.Phase, r.snap.Progress = "upload", 65
	r.mu.Unlock()
	payload := bytes.Repeat([]byte{0x5a}, int(size))
	started := time.Now()
	reader := &progressReader{reader: bytes.NewReader(payload), total: size, update: func(sent int64) {
		elapsed := time.Since(started).Seconds()
		if elapsed <= 0 {
			return
		}
		mbps := float64(sent*8) / elapsed / 1_000_000
		r.mu.Lock()
		r.snap.UploadMbps = round(mbps)
		r.snap.BytesUploaded = sent
		r.snap.Progress = 65 + int(math.Min(1, float64(sent)/float64(size))*35)
		r.snap.Samples = append(r.snap.Samples, Sample{At: time.Since(runStart).Milliseconds(), UpMbps: round(mbps)})
		r.mu.Unlock()
	}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://speed.cloudflare.com/__up", reader)
	if err != nil {
		return err
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New(resp.Status)
	}
	elapsed := time.Since(started).Seconds()
	if elapsed > 0 {
		mbps := float64(size*8) / elapsed / 1_000_000
		r.mu.Lock()
		r.snap.UploadMbps = round(mbps)
		r.snap.BytesUploaded = size
		r.snap.Samples = append(r.snap.Samples, Sample{At: time.Since(runStart).Milliseconds(), UpMbps: round(mbps)})
		r.mu.Unlock()
	}
	return nil
}

// Timed tests run repeated bounded requests until the chosen phase time ends.
// A phase deadline is a successful stop; a user cancellation remains an error.
func (m *Manager) downloadTimed(ctx context.Context, r *run, runStart time.Time, duration time.Duration) error {
	phaseCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	started := time.Now()
	var total int64
	r.mu.Lock()
	r.snap.Phase, r.snap.Progress = "download", 20
	r.mu.Unlock()
	for phaseCtx.Err() == nil {
		req, err := http.NewRequestWithContext(phaseCtx, http.MethodGet, "https://speed.cloudflare.com/__down?bytes=8388608", nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept-Encoding", "identity")
		resp, err := m.client.Do(req)
		if err != nil {
			if phaseCtx.Err() == context.DeadlineExceeded && ctx.Err() == nil {
				break
			}
			return err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			return errors.New(resp.Status)
		}
		buf := make([]byte, 64*1024)
		lastSample := time.Time{}
		for {
			n, readErr := resp.Body.Read(buf)
			total += int64(n)
			now := time.Now()
			if n > 0 && (lastSample.IsZero() || now.Sub(lastSample) >= 200*time.Millisecond) {
				elapsed := now.Sub(started).Seconds()
				r.mu.Lock()
				r.snap.BytesDownloaded = total
				r.snap.DownloadMbps = round(float64(total*8) / elapsed / 1_000_000)
				r.snap.Progress = 20 + int(math.Min(1, float64(now.Sub(started))/float64(duration))*45)
				r.snap.Samples = appendLimitedSample(r.snap.Samples, Sample{At: time.Since(runStart).Milliseconds(), DownMbps: r.snap.DownloadMbps})
				r.mu.Unlock()
				lastSample = now
			}
			if readErr != nil {
				if readErr != io.EOF && phaseCtx.Err() == nil {
					resp.Body.Close()
					return readErr
				}
				break
			}
		}
		resp.Body.Close()
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if total == 0 {
		return errors.New("no download data received")
	}
	r.mu.Lock()
	r.snap.BytesDownloaded, r.snap.Progress = total, 65
	r.mu.Unlock()
	return nil
}

func (m *Manager) uploadTimed(ctx context.Context, r *run, runStart time.Time, duration time.Duration) error {
	phaseCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	started := time.Now()
	payload := bytes.Repeat([]byte{0x5a}, 2*1024*1024)
	var total int64
	r.mu.Lock()
	r.snap.Phase, r.snap.Progress = "upload", 65
	r.mu.Unlock()
	for phaseCtx.Err() == nil {
		base := total
		reader := &progressReader{reader: bytes.NewReader(payload), total: int64(len(payload)), update: func(sent int64) {
			now := time.Now()
			elapsed := now.Sub(started).Seconds()
			if elapsed <= 0 {
				return
			}
			r.mu.Lock()
			r.snap.BytesUploaded = base + sent
			r.snap.UploadMbps = round(float64((base+sent)*8) / elapsed / 1_000_000)
			r.snap.Progress = 65 + int(math.Min(1, float64(now.Sub(started))/float64(duration))*35)
			r.snap.Samples = appendLimitedSample(r.snap.Samples, Sample{At: time.Since(runStart).Milliseconds(), UpMbps: r.snap.UploadMbps})
			r.mu.Unlock()
		}}
		req, err := http.NewRequestWithContext(phaseCtx, http.MethodPost, "https://speed.cloudflare.com/__up", reader)
		if err != nil {
			return err
		}
		req.ContentLength = int64(len(payload))
		req.Header.Set("Content-Type", "application/octet-stream")
		resp, err := m.client.Do(req)
		if err != nil {
			total = base + reader.sent
			if phaseCtx.Err() == context.DeadlineExceeded && ctx.Err() == nil {
				break
			}
			return err
		}
		_, copyErr := io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if copyErr != nil && phaseCtx.Err() == nil {
			return copyErr
		}
		if phaseCtx.Err() == context.DeadlineExceeded && ctx.Err() == nil {
			total = base + reader.sent
			break
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return errors.New(resp.Status)
		}
		total = base + int64(len(payload))
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if total == 0 {
		return errors.New("no upload data sent")
	}
	r.mu.Lock()
	r.snap.BytesUploaded, r.snap.Progress = total, 100
	r.mu.Unlock()
	return nil
}

func appendLimitedSample(samples []Sample, sample Sample) []Sample {
	if len(samples) >= 300 {
		copy(samples, samples[1:])
		samples[len(samples)-1] = sample
		return samples
	}
	return append(samples, sample)
}

type progressReader struct {
	reader *bytes.Reader
	total  int64
	sent   int64
	last   time.Time
	update func(int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.reader.Read(b)
	p.sent += int64(n)
	if p.last.IsZero() || time.Since(p.last) >= 200*time.Millisecond || p.sent == p.total {
		p.update(p.sent)
		p.last = time.Now()
	}
	return n, err
}

func finishFromError(r *run, ctx context.Context, message string) {
	if ctx.Err() != nil {
		finish(r, "cancelled", "")
		return
	}
	finish(r, "failed", message)
}

func finish(r *run, status, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snap.Status = status
	r.snap.Error = message
	r.snap.FinishedAt = time.Now().UTC()
	if status == "complete" {
		r.snap.Phase, r.snap.Progress = "complete", 100
	} else {
		r.snap.Phase = status
	}
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var total float64
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func jitter(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	var total float64
	for i := 1; i < len(values); i++ {
		total += math.Abs(values[i] - values[i-1])
	}
	return total / float64(len(values)-1)
}

func round(value float64) float64 { return math.Round(value*10) / 10 }

func newID() string {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format("150405.000000")))
	}
	return hex.EncodeToString(value)
}

func itoa(value int64) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0, 20)
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}
