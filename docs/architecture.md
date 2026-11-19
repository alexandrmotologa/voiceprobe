# voiceprobe System Architecture

## Overview
voiceprobe operates as a low-latency audio processing proxy designed for telecommunication environments. It inspects voice streams in real time to detect synthetic speech artifacts produced by neural vocoders and text-to-speech cloning systems.

The core design balances detection latency, CPU usage, and memory consumption. Target operating metrics:
- Stream detection latency: under 150 milliseconds per decision window.
- Resident memory footprint: under 25 megabytes with 50 active audio sessions.
- Ingestion formats: Twilio Media Streams (WebSocket), RTP/UDP (G.711 mu-law, A-law, linear PCM16), and standard WAV files.

## High-Level Data Flow

```
[ Inbound Audio Stream: RTP / Twilio WS / File ]
                       │
                       ▼
          [ Transport Demuxer / Decoder ]
         (PCMU / PCMA -> Linear PCM16)
                       │
                       ▼
             [ Lock-Free Ring Buffer ]
             (300ms window, 50% overlap)
                       │
                       ▼
            [ Voice Activity Detector ]
          (Silence & noise frame gating)
                       │
                       ▼
          [ SIMD-Optimized DSP Pipeline ]
       - Short-Time Fourier Transform (STFT)
       - Mel Filterbank Projection (40/128 bins)
       - Spectral Flux and Rolloff
       - Glottal Phase Discontinuity Index
       - Micro-Tremor (Jitter and Shimmer)
                       │
                       ▼
        [ Quantized Classifier Model ]
        (Multi-layer neural scoring engine)
                       │
                       ▼
       [ Temporal Smoothing and Thresholding ]
         (Exponential Moving Average filter)
                       │
        ┌──────────────┴──────────────┐
        ▼                             ▼
 [ Score < Threshold ]      [ Score >= Threshold ]
  Human verified log          Deepfake alert triggered
                              - SIP Header: X-Deepfake-Alert
                              - HTTP CRM Webhook
                              - Forensic Incident WAV recorder
```

## Core Modules

### 1. Transport Layer (`internal/transport`)
The transport layer accepts inbound real-time audio from multiple telephony gateways:
- `websocket.go`: Handles Twilio Media Stream connections over WebSockets. Twilio encodes telephony packets in base64-encoded G.711 mu-law chunks (typically 20ms chunks at 8000Hz). The transport decodes payloads into 16-bit linear PCM and feeds the ring buffer.
- `rtp.go`: Listens on configurable UDP ports for RTP audio streams (RFC 3550). It decodes payload types 0 (PCMU) and 8 (PCMA), reorders packets using sequence numbers, and compensates for network jitter.
- `sip.go`: Inspects SIP signaling messages (RFC 3261). When a deepfake is detected on an active call session, it mutates outbound SIP INVITE, UPDATE, or 200 OK responses to inject the `X-Deepfake-Alert` header.

### 2. Digital Signal Processing (`internal/dsp`)
The DSP package extracts acoustic features on sliding 300ms windows:
- `ringbuffer.go`: Ring buffer that maintains continuous audio sample history for each call session without dynamic memory allocations in the packet loop.
- `vad.go`: Voice activity detector based on short-term energy and zero-crossing rate. Non-speech frames are flagged and excluded from classifier scoring to avoid false positives during call silence.
- `stft.go`: Radix-2 Cooley-Tukey FFT with pre-calculated twiddle factors and Hanning windowing to minimize spectral leakage.
- `mel.go`: Triangular Mel filterbank matrix projecting spectral magnitudes into 40 Mel-spaced frequency bins.
- `flux.go`: Spectral flux and spectral rolloff measuring energy distribution variations between consecutive frames.
- `phase.go`: Unwrapped phase spectrum second derivative measuring glottal phase irregularities.
- `microtremor.go`: F0 pitch tracking via normalized autocorrelation, calculating jitter and shimmer variances.

### 3. Classification Engine (`internal/classifier`)
The classification layer aggregates DSP features into normalized feature vectors:
- `model.go`: Evaluates the feature vector against calibrated neural network layers using quantized float32/int8 arithmetic.
- `weights.go`: Embedded network weights trained to differentiate between natural human phonation and vocoder synthesis artifacts.
- `scorer.go`: Applies an exponential moving average (EMA) filter across consecutive 150ms hops to stabilize score fluctuations. Classifies calls into three severity levels: LOW (0.0 to 0.40), SUSPICIOUS (0.41 to 0.75), and HIGH (0.76 to 1.0).

### 4. Enterprise Actions (`internal/action`)
- `webhook.go`: Dispatches non-blocking JSON alerts to configured CRM endpoints when the detection score exceeds configured thresholds.
- `incident.go`: Writes raw audio buffers to disk as standard WAV files along with forensic JSON metadata files for incident audits.

### 5. Control Plane and Cockpit (`internal/server`)
- `api.go`: Provides REST endpoints for call management, telemetry queries, and file analysis.
- `sse.go`: Broadcasts real-time spectrogram, waveform, and threat scores to browser clients using Server-Sent Events.
- `embed.go`: Serves the compiled React 19 single-page application directly from the Go binary.
