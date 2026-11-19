package action

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/alexandrmotologa/voiceprobe/internal/classifier"
)

// AlertPayload defines the structured notification sent to enterprise endpoints.
type AlertPayload struct {
	Event         string                   `json:"event"`
	CallID        string                   `json:"call_id"`
	Timestamp     time.Time                `json:"timestamp"`
	ThreatLevel   classifier.RiskLevel     `json:"threat_level"`
	PeakScore     float64                  `json:"peak_score"`
	Confidence    float64                  `json:"confidence"`
	KeyIndicators []string                 `json:"key_indicators"`
	AudioWAVPath  string                   `json:"audio_wav_path,omitempty"`
}

// WebhookDispatcher delivers real-time notifications to external systems without blocking audio threads.
type WebhookDispatcher struct {
	enabled    bool
	targetURL  string
	client     *http.Client
	queue      chan AlertPayload
	wg         sync.WaitGroup
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewWebhookDispatcher starts background dispatch workers.
func NewWebhookDispatcher(targetURL string, enabled bool, timeout time.Duration) *WebhookDispatcher {
	ctx, cancel := context.WithCancel(context.Background())
	d := &WebhookDispatcher{
		enabled:    enabled,
		targetURL:  targetURL,
		client:     &http.Client{Timeout: timeout},
		queue:      make(chan AlertPayload, 256),
		ctx:        ctx,
		cancelFunc: cancel,
	}

	if enabled && targetURL != "" {
		d.wg.Add(1)
		go d.workerLoop()
	}

	return d
}

// Dispatch queues a threat alert for asynchronous delivery.
func (d *WebhookDispatcher) Dispatch(payload AlertPayload) {
	if !d.enabled || d.targetURL == "" {
		return
	}

	select {
	case d.queue <- payload:
	default:
		// Queue full: drop oldest or log to prevent blocking audio packet ingress
	}
}

func (d *WebhookDispatcher) workerLoop() {
	defer d.wg.Done()

	for {
		select {
		case <-d.ctx.Done():
			return
		case payload, ok := <-d.queue:
			if !ok {
				return
			}
			d.sendWithRetry(payload)
		}
	}
}

func (d *WebhookDispatcher) sendWithRetry(payload AlertPayload) {
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	maxRetries := 3
	backoff := 100 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(d.ctx, http.MethodPost, d.targetURL, bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "voiceprobe/1.0")

		resp, err := d.client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return
			}
		}

		select {
		case <-d.ctx.Done():
			return
		case <-time.After(backoff):
			backoff *= 2
		}
	}
}

// Close gracefully flushes pending tasks.
func (d *WebhookDispatcher) Close() {
	d.cancelFunc()
	close(d.queue)
	d.wg.Wait()
}
