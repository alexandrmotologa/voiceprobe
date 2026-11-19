# voiceprobe: Architecture and Engineering Master Plan
Real-Time Deepfake Voice and Synthetic Audio Fraud Detector for Telephony and Voice AI Streams
Target: Low-Latency Go DSP Engine (< 25MB RAM, < 150ms detection) with Embedded React 19 Cockpit

---

## 1. Problem Statement and Scope

### 1.1 Neural Voice Cloning in Financial Services and Telephony
Commercial speech synthesis models (such as ElevenLabs, F5-TTS, CosyVoice, XTTS-v2, and StyleTTS2) enable low-cost, zero-shot voice cloning from short reference audio clips. In banking and enterprise communications, this creates critical exposure vectors:
1. Executive impersonation (vishing) directed at finance desks to authorize wire transfers.
2. Voice biometric bypasses on automated call center IVR systems.
3. Telephony codec degradation (G.711 mu-law and A-law at 8kHz, Opus at 16kHz) stripping high-frequency acoustic cues that human ears rely on to spot synthetic speech.
4. Latency bounds: live phone calls require automated threat scoring within 150 milliseconds to alert agents or alter routing before transactional commitments.

### 1.2 Acoustic Markers of Neural Vocoders
Generative speech pipelines rely on neural vocoders (such as HiFi-GAN, BigVGAN, and Vocos) to reconstruct time-domain waveforms from acoustic embeddings. These vocoders leave measurable acoustic signatures:
- Glottal Phase Discontinuities: Natural vocal cord phonation exhibits continuous phase progression. Neural vocoders introduce micro-phase steps across adjacent frequency sub-bands.
- Absence of Neuromuscular Micro-Tremor: Human vocal folds exhibit continuous physiological jitter (pitch period variation) and shimmer (amplitude variation) between 3Hz and 8Hz. Synthetic voices produce over-smoothed pitch contours.
- Spectral Flux Anomalies: Convolutional upsampling layers generate characteristic energy variances across short-time Fourier frames during phonetic transitions.

### 1.3 The voiceprobe Solution
voiceprobe runs as a low-overhead proxy and analysis engine written in Go:
- Streaming packet ingestion across Twilio Media Streams (WebSocket), raw RTP (RFC 3550), and SIP signaling (RFC 3261).
- Sliding 300ms analysis windows with 50% overlap and voice activity detection to bypass non-speech intervals.
- Pure Go DSP routines: Short-Time Fourier Transform (STFT), 40/128-bin Mel filterbank, glottal phase unwrapping, spectral flux, and micro-tremor variance tracking.
- Quantized acoustic feature classification executing under 10ms per frame on standard CPU cores.
- Telephony actions: SIP warning header injection (`X-Deepfake-Alert: HIGH`), asynchronous HTTP webhooks for CRM screen pops, and forensic WAV snippet logging.
- Embedded React 19 single-binary dashboard serving real-time spectrogram waterfalls, live oscilloscopes, and active call diagnostics on port 7979.

---

## 2. System Architecture

```
+-------------------------------------------------------------+
|               Inbound Audio Streams                         |
|   Twilio Media Streams (WS) | RTP / UDP | Offline WAV Files |
+-------------------------------------------------------------+
                              |
              PCM 16-bit Samples (8kHz / 16kHz)
                              v
+-------------------------------------------------------------+
|                     voiceprobe Core Engine                  |
|                                                             |
|  +---------------------+        +------------------------+  |
|  | Packet Decapsulator |------->| Lock-Free Ring Buffer  |  |
|  | (RTP / Base64 WS)   |        | (300ms Sliding Window) |  |
|  +---------------------+        +------------------------+  |
|                                              |              |
|                                              v              |
|  +-------------------------------------------------------+  |
|  |                 Acoustic DSP Pipeline                 |  |
|  |  - Energy VAD Gating (Skip Silence Frames)            |  |
|  |  - Hanning Window + Cooley-Tukey FFT                  |  |
|  |  - Mel Filterbank Projection (Log Energies)           |  |
|  |  - Spectral Flux and Rolloff Calculation              |  |
|  |  - Glottal Phase Discontinuity Index                  |  |
|  |  - F0 Autocorrelation Jitter and Shimmer Variance     |  |
|  +-------------------------------------------------------+  |
|                              |                              |
|                              v                              |
|  +-------------------------------------------------------+  |
|  |         Quantized Acoustic Classifier Engine          |  |
|  |  - Multi-Layer Feature Classifier (CPU INT8 / Float32)|  |
|  |  - Exponential Moving Average (EMA) Temporal Filter   |  |
|  |  - Output: Synthetic Probability Score (0.0 to 1.0)   |  |
|  +-------------------------------------------------------+  |
|               /                               \             |
|     Score < Threshold                   Score >= Threshold  |
|             /                                   \           |
|            v                                     v          |
|   +------------------+                 +-----------------+  |
|   | Human Baseline   |                 | Deepfake Alert  |  |
|   | Passthrough Log  |                 | Action Trigger  |  |
|   +------------------+                 +-----------------+  |
|            \                                     /          |
|             +-----------------+-----------------+           |
|                               |                             |
+-------------------------------|-----------------------------+
                                |
                                v
+-------------------------------------------------------------+
|                  Enterprise Action Surfaces                 |
|  1. Injected SIP Warning Header (X-Deepfake-Alert: HIGH)    |
|  2. Async CRM Webhook Screen Pop Dispatcher                 |
|  3. Threat Incident WAV and Forensic Metadata Recorder      |
|  4. Embedded React 19 Audio Cockpit (HTTP :7979)            |
+-------------------------------------------------------------+
```

---

## 3. Mathematical DSP Formulation

### 3.1 Spectral Flux ($S_{\text{flux}}$)
Calculated across successive short-time Fourier frames $X_t$ and $X_{t-1}$ for $K$ frequency bins:

$$S_{\text{flux}}(t) = \sum_{k=1}^K H\Big( |X_t(k)| - |X_{t-1}(k)| \Big)$$

Here $H(x) = \frac{x + |x|}{2}$ represents the half-wave rectifier function. Neural vocoders exhibit abnormal step transitions in high-frequency spectral flux.

### 3.2 Glottal Phase Discontinuity ($\Phi_{\text{disc}}$)
Let $\phi(t, \omega)$ denote the unwrapped phase spectrum at frame $t$ across discrete frequency indices $\omega$:

$$\Phi_{\text{disc}}(t) = \frac{1}{\Omega} \sum_{\omega=2}^{\Omega-1} \left| \phi(t, \omega+1) - 2\phi(t, \omega) + \phi(t, \omega-1) \right|$$

Vocoder synthesis causes phase disruptions across harmonic multiples compared to the continuous phase trajectories of natural vocal tract resonances.

### 3.3 Micro-Tremor Variance (Pitch Perturbation)
Calculated from short-time fundamental frequency $F_0$ estimates across successive voiced frames:

$$\text{Jitter}_{\text{local}} = \frac{\frac{1}{N-1} \sum_{i=1}^{N-1} |T_i - T_{i+1}|}{\frac{1}{N} \sum_{i=1}^N T_i}$$

Synthetic audio engines lack the organic neuromuscular micro-tremor of natural human phonation, resulting in abnormally low jitter values over sustained vowels.

---

## 4. Repository Layout

```
voiceprobe/
├── cmd/
│   └── voiceprobe/
│       └── main.go                 # CLI commands: listen, analyze, stream-sim, benchmark, studio
├── internal/
│   ├── config/
│   │   ├── config.go               # YAML, env, and flag configuration loader
│   │   └── schema.go               # Thresholds, telemetry, and network settings
│   ├── dsp/
│   │   ├── ringbuffer.go           # Concurrency-safe sliding PCM sample window
│   │   ├── vad.go                  # Energy and zero-crossing voice activity detector
│   │   ├── stft.go                 # Cooley-Tukey FFT and Hanning windowing
│   │   ├── mel.go                  # Mel filterbank projection matrices
│   │   ├── flux.go                 # Spectral flux and rolloff extractors
│   │   ├── phase.go                # Phase unwrapping and discontinuity index
│   │   └── microtremor.go          # F0 pitch tracker, jitter, and shimmer estimators
│   ├── classifier/
│   │   ├── model.go                # Quantized neural inference network
│   │   ├── weights.go              # Pre-trained acoustic detection weights
│   │   ├── onnx_loader.go          # ONNX parser and parameter importer
│   │   └── scorer.go               # Temporal smoothing, confidence, and alert states
│   ├── transport/
│   │   ├── websocket.go            # Twilio Media Streams JSON decoder and handler
│   │   ├── rtp.go                  # UDP RTP packet listener with G.711 decoder
│   │   └── sip.go                  # SIP message parser and header injector
│   ├── action/
│   │   ├── webhook.go              # Asynchronous HTTP notification client
│   │   ├── sip_header.go           # SIP alert header injection routines
│   │   └── incident.go             # Threat WAV audio and JSON metadata recorder
│   └── server/
│       ├── api.go                  # REST management endpoints (/api/v1/...)
│       ├── sse.go                  # Server-Sent Events real-time telemetry stream
│       └── embed.go                # Embedded React 19 UI assets
├── web/
│   ├── src/
│   │   ├── App.tsx                 # Application shell and state provider
│   │   ├── components/
│   │   │   ├── SpectrogramWaterfall.tsx # Canvas 60fps real-time spectrogram
│   │   │   ├── WaveformOscilloscope.tsx # Time-domain audio oscilloscope
│   │   │   ├── SyntheticScoreGauge.tsx  # Radial threat probability meter
│   │   │   ├── CallActiveList.tsx       # Live calls table with threat badges
│   │   │   ├── FileAnalyzer.tsx         # Interactive drag-and-drop audio inspector
│   │   │   ├── StreamSimulator.tsx      # On-demand call injection test controls
│   │   │   └── ForensicPlayer.tsx       # Audio player highlighting anomaly windows
│   │   └── pages/
│   │       └── Dashboard.tsx            # Main operations console
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── docs/
│   ├── architecture.md             # Deep system design and data flow
│   ├── dsp_algorithms.md           # Mathematical DSP derivations and references
│   ├── telephony_integration.md    # Twilio, Asterisk, and FreeSWITCH integration
│   └── forensic_metrics.md         # Interpretation guide for acoustic metrics
├── examples/
│   ├── voiceprobe.yaml             # Full configuration example
│   ├── sample_calls/               # Test audio samples (organic vs. synthetic)
│   └── twilio_twiml_example.xml    # Sample Twilio Media Stream TwiML payload
├── Makefile
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

---

## 5. Implementation Roadmap and Verification Checklist

### Phase 1: Core DSP and Circular Buffer
- Implement `internal/dsp/ringbuffer.go` for sliding PCM windows.
- Implement `internal/dsp/stft.go` Cooley-Tukey FFT with pre-computed twiddles.
- Implement `internal/dsp/vad.go` for energy and zero-crossing detection.
- Implement `internal/dsp/mel.go`, `flux.go`, `phase.go`, and `microtremor.go`.
- Unit tests verifying FFT accuracy against reference signals and phase discontinuity detection on synthetic waveforms.

### Phase 2: Classification Engine and Scorer
- Build `internal/classifier/model.go` with multi-layer feedforward evaluation.
- Embed calibrated pre-trained weights for synthetic speech detection.
- Implement `internal/classifier/scorer.go` with exponential moving average smoothing.
- Unit tests validating score sensitivity on known organic vs. synthetic test vectors.

### Phase 3: Telephony Ingestion and Network Transports
- Implement `internal/transport/websocket.go` with Twilio Media Streams compatibility.
- Implement `internal/transport/rtp.go` UDP listener with PCMU/PCMA decompression.
- Implement `internal/transport/sip.go` message parser and header injector.
- End-to-end tests streaming mock Twilio WebSocket messages and RTP packets.

### Phase 4: Action Engine and Telemetry Server
- Implement `internal/action/webhook.go` async dispatcher with retry logic.
- Implement `internal/action/incident.go` WAV writer and metadata logger.
- Implement `internal/server/api.go` and `internal/server/sse.go` telemetry stream.

### Phase 5: React 19 Audio Cockpit
- Build modern React 19 UI with Vite, TypeScript, and Canvas visualizers.
- Implement real-time spectrogram waterfall, oscilloscope, and radial threat gauge.
- Build interactive file upload analyzer and call simulator control.
- Embed production build in Go binary via `embed.FS`.

### Phase 6: CLI Tooling, Benchmarks, and Verification
- Wire CLI subcommands (`listen`, `analyze`, `stream-sim`, `benchmark`, `studio`).
- Generate deterministic sample WAV audio files for test calls.
- Benchmark concurrent stream capacity and verify latency under 150ms and RAM under 25MB.
- Final documentation and humanized README.
