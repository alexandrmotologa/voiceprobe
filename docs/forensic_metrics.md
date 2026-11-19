# Forensic Acoustic Metrics Guide

This guide explains how security analysts and forensic audio engineers should interpret the acoustic metrics reported by voiceprobe.

## Metric Summary Reference

| Metric Identifier | Normal Human Range | Synthetic / Vocoder Range | Primary Diagnostic Significance |
|---|---|---|---|
| `phase_discontinuity` | 0.005 - 0.035 | 0.055 - 0.180 | Convolutional vocoder upsampling artifacts |
| `spectral_flux_variance`| 0.008 - 0.040 | 0.048 - 0.125 | Frame-to-frame harmonic transition rigidity |
| `pitch_jitter_local` | 0.005 - 0.025 | 0.0005 - 0.0035 | Absence of natural laryngeal micro-tremors |
| `shimmer_local` | 0.020 - 0.075 | 0.003 - 0.015 | Amplitude over-smoothing across pitch periods |
| `spectral_centroid_hz` | 400 - 2800 Hz | Telephony-dependent | Overall brightness and frequency distribution |
| `vad_energy_ratio` | > 0.01 (active) | > 0.01 (active) | Speech presence indicator (silence filter) |

---

## Detailed Metric Analysis

### 1. Glottal Phase Discontinuity (`phase_discontinuity`)
- **Physics**: Natural human phonation exhibits continuous phase transitions across harmonic multiples because of vocal tract resonance continuity.
- **Vocoder Signature**: Generative models (e.g. HiFi-GAN, BigVGAN, MelGAN) reconstruct time-domain waveforms using transposed convolutions or inverse STFT with estimated phase. This introduces micro-steps in the second derivative of the frequency-domain phase spectrum.
- **Threshold**: Values exceeding 0.050 strongly suggest vocoder-based waveform synthesis.

### 2. Spectral Flux Variance (`spectral_flux_variance`)
- **Physics**: Human articulation involves continuous physiological changes in lip, tongue, and jaw geometry. Energy distribution shifts progressively across spectral bands.
- **Vocoder Signature**: Neural vocoders often produce rigid spectral energy profiles during steady-state vowels, punctuated by unnatural high-frequency energy shifts at frame boundaries.
- **Threshold**: Values exceeding 0.048 denote synthetic spectral transition patterns.

### 3. Pitch Jitter (`pitch_jitter_local`)
- **Physics**: Neuromuscular laryngeal tremor introduces continuous micro-variations in vocal fold vibration period ($T_0$), fluctuating naturally between 0.5% and 2.5% in natural speech.
- **Vocoder Signature**: Many text-to-speech architectures predict smooth pitch contours ($F_0$) via deterministic variance adaptors or diffusion estimators, resulting in abnormally smooth pitch trajectories ($< 0.35\%$).
- **Threshold**: Sustained jitter below 0.0035 during voiced speech segments is an indicator of synthetic voice generation.

### 4. Amplitude Shimmer (`shimmer_local`)
- **Physics**: Shimmer quantifies cycle-to-cycle amplitude variations between adjacent glottal pulses, reflecting natural aerodynamic airflow turbulence.
- **Vocoder Signature**: Synthesized speech frequently displays near-perfect cycle-to-cycle amplitude symmetry during vowel phonation.
- **Threshold**: Sustained shimmer below 0.015 on voiced phonemes indicates synthetic origin.

---

## Scoring Logic and Confidence Calculation

1. **Window-Level Score**:
   The feedforward classifier computes an uncalibrated logit $z$, mapped through a sigmoid activation to produce an instant probability $p_t \in [0, 1]$.

2. **Exponential Moving Average (EMA)**:
   To avoid transient spikes from clicks, line noise, or coughs, scores are smoothed across time:
   $$S_t = \alpha \cdot p_t + (1 - \alpha) \cdot S_{t-1}$$
   Where $\alpha = 0.35$ for speech frames.

3. **Threat Classification**:
   - `0.00 - 0.40`: **CLEAN (Human Verified)**
   - `0.41 - 0.75`: **SUSPICIOUS (Elevated Monitoring)**
   - `0.76 - 1.00`: **CRITICAL (Deepfake Voice Detected)**
