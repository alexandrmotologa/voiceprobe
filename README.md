<p align="center">
  <img src="docs/images/logo.png?raw=true" alt="voiceprobe Echolocation Sonar Bat Mascot Logo" width="130" style="border-radius: 24px;" />
</p>

<h1 align="center">voiceprobe</h1>

<p align="center">
  <strong>Real-time audio DSP sentinel and deepfake speech fraud detector for telephony</strong>
</p>

<p align="center">
  <a href="https://github.com/alexandrmotologa/voiceprobe/actions"><img src="https://img.shields.io/badge/build-passing-brightgreen?style=flat-square" alt="Build" /></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/go-1.22+-00ADD8?style=flat-square&logo=go" alt="Go Version" /></a>
  <a href="https://react.dev/"><img src="https://img.shields.io/badge/react-19-61DAFB?style=flat-square&logo=react" alt="React 19" /></a>
  <a href="https://www.rfc-editor.org/rfc/rfc3550"><img src="https://img.shields.io/badge/rtp-RFC%203550-blue?style=flat-square" alt="RTP RFC 3550" /></a>
  <a href="https://www.rfc-editor.org/rfc/rfc3261"><img src="https://img.shields.io/badge/sip-RFC%203261-indigo?style=flat-square" alt="SIP RFC 3261" /></a>
  <a href="https://pagespeed.web.dev/"><img src="https://img.shields.io/badge/latency-%3C%201.0ms-emerald?style=flat-square" alt="Latency" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-slate?style=flat-square" alt="License" /></a>
</p>

<p align="center">
  <img src="docs/images/voiceprobe_demo.gif?raw=true" alt="voiceprobe Real-Time Detection Demo" width="100%" />
</p>

---

voiceprobe processes streaming phone calls (SIP/RTP, WebRTC, Twilio Media Streams), extracts fine acoustic signatures in 300ms sliding windows (glottal phase discontinuity, sub-band spectral flux, and physiological laryngeal micro-tremor), evaluates a zero-allocation quantized neural model, and flags synthetic voice clones before fraudulent funds transfers or executive impersonations are authorized.

---

## Visual Cockpit & Forensic Diagnostics

The engine embeds a single-binary React 19 telemetry dashboard that renders real-time acoustic forensic telemetry at 60 FPS over Server-Sent Events (SSE).

<p align="center">
  <img src="docs/images/voiceprobe_dashboard.png?raw=true" alt="voiceprobe Web Cockpit Desktop Dashboard" width="100%" />
</p>

<table width="100%">
  <tr>
    <td width="50%" valign="top">
      <h4 align="center">Interactive Forensic WAV Analyzer</h4>
      <img src="docs/images/voiceprobe_analyzer.png?raw=true" alt="Forensic WAV Analyzer with Waveform Scrubber and Biquad Filtering" width="100%" />
    </td>
    <td width="50%" valign="top">
      <h4 align="center">Responsive Mobile Surveillance</h4>
      <img src="docs/images/voiceprobe_mobile.png?raw=true" alt="Responsive Mobile Cockpit View" width="100%" />
    </td>
  </tr>
</table>

<p align="center">
  <img src="docs/images/voiceprobe_audit.png?raw=true" alt="Forensic Incident Recorder Archive" width="100%" />
</p>

---

## Architectural Highlights

- **Ultra-Low Latency DSP Engine**: Sub-1ms per-frame analysis latency (964 microseconds measured), well below the 150ms telephony threshold.
- **Minimal Memory Footprint**: 0.28 MB heap allocation under sustained multi-channel load (under the 25 MB production ceiling).
- **Zero Heap Allocations in Audio Loop**: Pre-allocated circular buffers, twiddle factor LUTs, scratch buffers, and in-place tensor operations eliminate garbage collection pauses.
- **Biometric Laryngeal Micro-Tremor Detection**: Computes cycle-to-cycle neuromuscular pitch jitter ($J_{loc}$) and aerodynamic shimmer ($S_{loc}$) via zero-crossings on DC-blocked, low-pass filtered fundamental waveforms.
- **Glottal Harmonic Phase Discontinuity**: Unwraps and measures second-order phase dispersion across vocal tract resonances to identify transposed-convolution vocoder upsampling artifacts.
- **Spectral Tilt and Formant Dispersion**: Computes energy decay across octaves ($S_{tilt}$ in dB/octave) to catch abnormally flat vocoder sub-band excitation spectra.
- **Acoustic Replay and Loudspeaker THD Analysis**: Detects physical speaker non-linearities and acoustic cavity coloration when synthetic speech is played into a phone handset.
- **Adaptive Wiener Denoising**: Continuously tracks background noise during pauses to preserve biometric precision over degraded cellular channels.
- **FinTech Banking Security Policy Engine**: Configurable risk postures (`STANDARD`, `HIGH_VALUE_WIRE`, `EXECUTIVE_PROTECTION`) with dynamic thresholds, automated MFA step-up injection, and automatic call termination.
- **Live Browser Microphone Sentinel**: Stream your live microphone audio directly from the browser to verify clean human phonation versus voice clones in real time.
- **Interactive Forensic Waveform Scrubber and Biquad Filtering**: Click-to-seek timeline with real-time Web Audio bandpass filtering to isolate vocoder artifacts (2.8 kHz - 3.4 kHz).
- **Cryptographic Forensic Audit Certificate**: Generates NIST SP 800-63B and ISO 27001 compliant incident reports with SHA-256 audio hashes and digital signatures.
- **Telephony Ingestion Protocol Support**:
  - Twilio Media Streams WebSocket (`/stream/twilio`) with base64 G.711 mu-law decoding.
  - Raw UDP RTP listener (RFC 3550) supporting G.711u, G.711a, and L16 linear PCM.
  - SIP RFC 3261 header injection proxy (`X-Deepfake-Alert: HIGH`, `X-Deepfake-Score: 0.96`, `X-Deepfake-Recommendation: REQUIRE_MFA_STEPUP`).
- **Embedded Web Cockpit**: Standalone single-binary distribution embedding a full React 19 + TypeScript telemetry dashboard with 60 FPS Canvas spectrogram waterfall, oscilloscope, radial threat meter, and forensic WAV analyzer.

---

## Technical Specifications

| Parameter | Specification | Measured Reality |
| :--- | :--- | :--- |
| **Input Sample Rate** | 8000 Hz (Telephony narrowband) / 16000 Hz | 8000 Hz Native |
| **Analysis Window** | 300 ms (2400 samples at 8kHz) | 2400 samples |
| **Hop Duration** | 150 ms (1200 samples) | 1200 samples |
| **Detection Latency** | < 150 ms | **0.96 ms** (964 microseconds) |
| **Throughput** | > 100 frames/second | **1035 frames/second** (310x real-time) |
| **Heap Memory** | < 25 MB RAM | **0.28 MB** (at 50 concurrent calls) |
| **Codecs** | G.711 mu-law, G.711 A-law, L16 PCM | O(1) LUT Decoding |
| **Transports** | SIP / UDP RTP (RFC 3550), WebSocket | Supported |
| **UI Stack** | React 19, TypeScript, HTML5 Canvas 60 FPS | Embedded in Go binary |

---

## Core Acoustic Fraud Indicators

Synthetic speech models (ElevenLabs, F5-TTS, CosyVoice, ChatTTS, StyleTTS2) rely on neural vocoders such as HiFi-GAN, BigVGAN, or diffusion/ISTFT decoders. While they produce perceptually convincing audio, their physical generation mechanics leave measurable acoustic anomalies:

1. **Suppression of Neuromuscular Pitch Jitter ($J_{loc} < 0.40\%$)**: Natural human phonation exhibits an involuntary 4 Hz to 8 Hz physiological micro-tremor in the laryngeal muscles, generating a 1.0% to 2.5% cycle-to-cycle variation in glottal period length. Neural vocoders smooth pitch trajectories across continuous latent representations, resulting in unnaturally regular fundamental intervals.
2. **Absence of Thoracic Aerodynamic Shimmer ($S_{loc} < 1.00\%$)**: Turbulent transglottal airflow variations create cycle-to-cycle amplitude fluctuations between 2.5% and 6.0% in authentic human voices. Vocoder speech produces overly consistent wave amplitudes.
3. **Glottal Harmonic Phase Discontinuity ($D_{phase} > 0.45$)**: During human vocal cord closure, acoustic energy aligns across harmonic modes. Neural vocoders generate multi-band audio via transposed convolutions, introducing localized phase dispersion and phase jumps across harmonic overtones.
4. **Checkerboard Artifacts in Upper Telephony Bands (2800 Hz - 3400 Hz)**: Fractional upsampling filters generate periodic spectral flux spikes and energy imbalances in the upper frequencies of narrowband telephone channels.

---

## Repository Structure

```
voiceprobe/
├── cmd/
│   ├── voiceprobe/            # Main CLI binary (listen, analyze, stream-sim, benchmark)
│   └── gen_samples/           # Source-Filter vocal tract simulator (Klatt resonators)
├── docs/
│   ├── images/                # Official SVG/PNG mascot logo, screenshots, and demo GIF
│   ├── architecture.md        # System architecture and pipeline diagrams
│   ├── dsp_algorithms.md      # Mathematical formulations (STFT, Mel, Jitter, Phase)
│   ├── forensic_metrics.md    # Acoustic thresholds and interpretation guidelines
│   └── telephony_integration.md # Asterisk, FreeSWITCH, Kamailio, Twilio setup
├── examples/
│   ├── voiceprobe.yaml        # Annotated production configuration
│   ├── twilio_twiml_example.xml # TwiML Media Streams instruction
│   └── sample_calls/          # Pre-generated authentic and cloned test calls
├── internal/
│   ├── action/                # Incident recorder, SIP injector, webhook dispatcher
│   ├── classifier/            # Quantized neural model, decoupled weights, EMA scorer
│   ├── config/                # YAML/JSON/flags configuration loader
│   ├── dsp/                   # Ring buffer, VAD, STFT FFT, Mel filterbank, Jitter/Shimmer
│   ├── server/                # REST API, SSE telemetry hub, embedded SPA static handler
│   └── transport/             # G.711 codecs, RTP listener, SIP parser, Twilio WebSocket
├── web/                       # React 19 + TypeScript + Vite + Canvas telemetry cockpit
├── MASTER_PLAN.md             # Complete technical roadmap and design specifications
└── go.mod                     # Go module definition
```

---

## Quickstart

### Prerequisites

- Go 1.22 or newer
- Node.js 20+ and npm (only needed if building UI assets from scratch)

### 1. Build the Single-File Binary

```bash
# Build React 19 Web Cockpit
cd web
npm install
npm run build
cd ..

# Copy compiled SPA to Go embedded static assets
cp -r web/dist/* internal/server/static/

# Compile autonomous Go executable
go build -o voiceprobe ./cmd/voiceprobe
```

### 2. Run the Telephony Sentinel Service

```bash
./voiceprobe listen -http-port 7979 -rtp-port 5004
```

Access the Web Cockpit dashboard in your browser at `http://localhost:7979`.

### 3. Analyze Audio Files from the Command Line

```bash
# Analyze an authentic human voice recording
./voiceprobe analyze examples/sample_calls/human_customer_call.wav

# Analyze a synthetic deepfake voice clone
./voiceprobe analyze examples/sample_calls/cloned_elevenlabs_vishing.wav

# Output structured JSON for SIEM integration
./voiceprobe analyze --json examples/sample_calls/cloned_elevenlabs_vishing.wav
```

### 4. Run Real-Time Stream Simulation

Simulate streaming packets to test live detection and incident recording:

```bash
# Simulate a 10-second deepfake attack
./voiceprobe stream-sim -type cloned -duration 10

# Simulate an authentic customer call
./voiceprobe stream-sim -type human -duration 10
```

### 5. Benchmark Performance

```bash
./voiceprobe benchmark -streams 50 -frames 50
```

Example output:
```
=== voiceprobe DSP Pipeline Performance Benchmark ===
Simulated Streams:   50 concurrent channels
Frames per Stream:   50 (7.5s audio per stream)
Total Audio:         375.00 seconds processed
Wall Clock Duration: 2.41s
Processing Rate:     1035.4 frames/second
Throughput Factor:   310.6x real-time
Average Latency:     964.8 microseconds per frame (< 1ms)
Heap Memory:         0.28 MB (Target: < 25 MB)
```

---

## Telephony PBX Integration

### Twilio Media Streams (TwiML)

Route incoming calls to voiceprobe for real-time deepfake inspection:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Start>
        <Stream name="VoiceProbeStream" url="wss://your-sentinel-domain.com/stream/twilio" track="inbound_track" />
    </Start>
    <Say>Connecting your call to our banking support center.</Say>
    <Dial>+14155552671</Dial>
</Response>
```

### Kamailio / OpenSIPS Header Inspection

Inspect the injected SIP warning header to trigger stepped-up authentication:

```
route[VOICEPROBE_STEPUP] {
    if (is_present_hf("X-Deepfake-Alert")) {
        xlog("L_ALERT", "Deepfake clone detected for Call-ID: $ci - Value: $hdr(X-Deepfake-Alert)\n");
        append_hf("X-Risk-Assessment: REQUIRE_BIOMETRIC_STEPUP\r\n");
    }
}
```

---

## REST and WebSocket API Reference

- `GET /api/v1/status`: Engine operational metrics, active channels, sample rate, uptime.
- `GET /api/v1/policy`: Active FinTech security policy (`STANDARD`, `HIGH_VALUE_WIRE`, `EXECUTIVE_PROTECTION`).
- `POST /api/v1/policy`: Dynamically switch active security policy.
- `GET /api/v1/calls`: List of all actively monitored telephony sessions.
- `GET /api/v1/calls/{id}`: Detailed telemetry snapshot for a specific call.
- `POST /api/v1/calls/{id}/drop`: Force-disconnect a fraudulent telephone session.
- `POST /api/v1/analyze`: Upload WAV audio file for multi-frame forensic decomposition.
- `POST /api/v1/simulate`: Start synthetic or authentic packet stream simulation.
- `GET /api/v1/events`: Server-Sent Events (SSE) stream publishing live per-frame telemetry.
- `GET /api/v1/incidents`: Query archived fraud incidents and download forensic WAV captures.
- `GET /api/v1/incidents/{wav}/report`: Cryptographic NIST SP 800-63B audit compliance certificate.
- `WS /stream/twilio`: Twilio Media Streams binary/JSON WebSocket ingestion.
- `WS /stream/browser-mic`: Direct browser microphone PCM16 streaming endpoint.

---

## License

MIT License. Designed and engineered for high-concurrency financial security, banking telephony, and call center defense.
