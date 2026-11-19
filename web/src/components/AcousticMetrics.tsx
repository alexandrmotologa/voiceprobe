import React from 'react';
import { Fingerprint, Gauge, Zap, Disc3, Radio, FileText } from 'lucide-react';
import { TelemetryEvent } from '../types/telemetry';

interface AcousticMetricsProps {
  telemetry?: TelemetryEvent | null;
}

export const AcousticMetrics: React.FC<AcousticMetricsProps> = ({ telemetry }) => {
  if (!telemetry) {
    return (
      <div className="p-5 rounded-xl bg-[#0f111a] border border-slate-800/90 flex items-center justify-center text-slate-500 font-mono text-xs">
        No active stream telemetry available.
      </div>
    );
  }

  const {
    jitter,
    shimmer,
    phase_discontinuity,
    spectral_flux,
    spectral_centroid,
    f0_hz,
    spectral_tilt = -6.0,
    formant_dispersion = 1000.0,
    thd_percent = 5.2,
    loudspeaker_anomaly = false,
    vad_energy,
    key_indicators = [],
    risk_level,
  } = telemetry;

  // Jitter evaluation: natural human is typically 0.010 - 0.025 (1.0% - 2.5%)
  // Cloned vocoder is typically < 0.004 (< 0.4%)
  const jitterPct = (jitter * 100);
  const isJitterDeficit = jitter < 0.006;

  // Shimmer evaluation: natural human is typically 0.025 - 0.060 (2.5% - 6.0%)
  // Cloned vocoder is typically < 0.012 (< 1.2%)
  const shimmerPct = (shimmer * 100);
  const isShimmerDeficit = shimmer < 0.012;

  // Glottal phase discontinuity: natural < 0.25, cloned > 0.45
  const isPhaseAbnormal = phase_discontinuity > 0.40;

  // Spectral tilt anomaly: neural vocoder is typically flatter than -3.5 dB/octave
  const isTiltAbnormal = spectral_tilt > -3.5 && spectral_tilt < 0.0;
  // Loudspeaker replay anomaly
  const isReplayAnomaly = loudspeaker_anomaly || thd_percent > 18.0;

  return (
    <div className="p-5 rounded-xl bg-[#0f111a] border border-slate-800/90 flex flex-col justify-between">
      <div className="flex items-center justify-between mb-3 border-b border-slate-800/80 pb-2">
        <div className="flex items-center gap-2">
          <Fingerprint className="w-4 h-4 text-cyan-400" />
          <span className="text-xs font-mono font-semibold tracking-wider text-slate-300 uppercase">
            Acoustic Biometric Diagnostics
          </span>
        </div>
        <span className="text-[10px] font-mono text-slate-400">
          DSP SLIDING WINDOW: 300ms / 150ms HOP
        </span>
      </div>

      {/* Grid of Key Features */}
      <div className="grid grid-cols-2 md:grid-cols-3 gap-3 mb-4">
        {/* Laryngeal Jitter */}
        <div className={`p-2.5 rounded-lg border bg-slate-950/70 ${
          isJitterDeficit
            ? 'border-rose-900/80 text-rose-300 shadow-[0_0_10px_rgba(244,63,94,0.15)]'
            : 'border-slate-800 text-slate-300'
        }`}>
          <div className="flex items-center justify-between text-[11px] font-mono mb-1 text-slate-400">
            <span className="flex items-center gap-1">
              <Gauge className="w-3 h-3 text-cyan-400" /> Pitch Jitter
            </span>
            <span className={isJitterDeficit ? 'text-rose-400 font-bold' : 'text-emerald-400'}>
              {isJitterDeficit ? 'DEFICIT' : 'NATURAL'}
            </span>
          </div>
          <div className="text-lg font-mono font-bold tracking-tight">
            {jitterPct.toFixed(2)}%
          </div>
          <div className="text-[10px] text-slate-500 font-mono mt-0.5">
            Baseline: &gt; 1.00%
          </div>
        </div>

        {/* Aerodynamic Shimmer */}
        <div className={`p-2.5 rounded-lg border bg-slate-950/70 ${
          isShimmerDeficit
            ? 'border-rose-900/80 text-rose-300 shadow-[0_0_10px_rgba(244,63,94,0.15)]'
            : 'border-slate-800 text-slate-300'
        }`}>
          <div className="flex items-center justify-between text-[11px] font-mono mb-1 text-slate-400">
            <span className="flex items-center gap-1">
              <Zap className="w-3 h-3 text-amber-400" /> Shimmer
            </span>
            <span className={isShimmerDeficit ? 'text-rose-400 font-bold' : 'text-emerald-400'}>
              {isShimmerDeficit ? 'DEFICIT' : 'NATURAL'}
            </span>
          </div>
          <div className="text-lg font-mono font-bold tracking-tight">
            {shimmerPct.toFixed(2)}%
          </div>
          <div className="text-[10px] text-slate-500 font-mono mt-0.5">
            Baseline: &gt; 2.50%
          </div>
        </div>

        {/* Glottal Phase Discontinuity */}
        <div className={`p-2.5 rounded-lg border bg-slate-950/70 ${
          isPhaseAbnormal
            ? 'border-rose-900/80 text-rose-300 shadow-[0_0_10px_rgba(244,63,94,0.15)]'
            : 'border-slate-800 text-slate-300'
        }`}>
          <div className="flex items-center justify-between text-[11px] font-mono mb-1 text-slate-400">
            <span className="flex items-center gap-1">
              <Disc3 className="w-3 h-3 text-indigo-400" /> Phase Curvature
            </span>
            <span className={isPhaseAbnormal ? 'text-rose-400 font-bold' : 'text-emerald-400'}>
              {isPhaseAbnormal ? 'ANOMALY' : 'COHERENT'}
            </span>
          </div>
          <div className="text-lg font-mono font-bold tracking-tight">
            {phase_discontinuity.toFixed(3)}
          </div>
          <div className="text-[10px] text-slate-500 font-mono mt-0.5">
            Threshold: &lt; 0.250
          </div>
        </div>

        {/* Fundamental Frequency F0 */}
        <div className="p-2.5 rounded-lg border border-slate-800 bg-slate-950/70 text-slate-300">
          <div className="text-[11px] font-mono text-slate-400 mb-1 flex items-center gap-1">
            <Radio className="w-3 h-3 text-cyan-400" /> Fundamental F0
          </div>
          <div className="text-lg font-mono font-bold tracking-tight">
            {f0_hz > 0 ? `${f0_hz.toFixed(1)} Hz` : '---'}
          </div>
          <div className="text-[10px] text-slate-500 font-mono mt-0.5">
            Telephony range: 75-350Hz
          </div>
        </div>

        {/* Spectral Flux */}
        <div className="p-2.5 rounded-lg border border-slate-800 bg-slate-950/70 text-slate-300">
          <div className="text-[11px] font-mono text-slate-400 mb-1">
            Spectral Flux
          </div>
          <div className="text-lg font-mono font-bold tracking-tight">
            {spectral_flux.toFixed(3)}
          </div>
          <div className="text-[10px] text-slate-500 font-mono mt-0.5">
            MPEG-7 Sub-band Delta
          </div>
        </div>

        {/* Spectral Centroid & VAD Energy */}
        <div className="p-2.5 rounded-lg border border-slate-800 bg-slate-950/70 text-slate-300">
          <div className="text-[11px] font-mono text-slate-400 mb-1">
            Centroid / VAD
          </div>
          <div className="text-lg font-mono font-bold tracking-tight">
            {spectral_centroid.toFixed(0)} Hz
          </div>
          <div className="text-[10px] text-slate-500 font-mono mt-0.5">
            RMS: {(vad_energy * 1000).toFixed(1)} mPa | {risk_level}
          </div>
        </div>

        {/* Spectral Tilt */}
        <div className={`p-2.5 rounded-lg border bg-slate-950/70 ${
          isTiltAbnormal
            ? 'border-rose-900/80 text-rose-300 shadow-[0_0_10px_rgba(244,63,94,0.15)]'
            : 'border-slate-800 text-slate-300'
        }`}>
          <div className="text-[11px] font-mono text-slate-400 mb-1 flex items-center justify-between">
            <span>Spectral Tilt</span>
            <span className={isTiltAbnormal ? 'text-rose-400 font-bold' : 'text-emerald-400'}>
              {isTiltAbnormal ? 'FLAT' : 'NATURAL'}
            </span>
          </div>
          <div className="text-lg font-mono font-bold tracking-tight">
            {spectral_tilt.toFixed(1)} dB/oct
          </div>
          <div className="text-[10px] text-slate-500 font-mono mt-0.5">
            Dispersion: {formant_dispersion.toFixed(0)}Hz
          </div>
        </div>

        {/* Loudspeaker Replay THD */}
        <div className={`p-2.5 rounded-lg border bg-slate-950/70 ${
          isReplayAnomaly
            ? 'border-rose-900/80 text-rose-300 shadow-[0_0_10px_rgba(244,63,94,0.15)]'
            : 'border-slate-800 text-slate-300'
        }`}>
          <div className="text-[11px] font-mono text-slate-400 mb-1 flex items-center justify-between">
            <span>Replay THD</span>
            <span className={isReplayAnomaly ? 'text-rose-400 font-bold' : 'text-emerald-400'}>
              {isReplayAnomaly ? 'SPEAKER' : 'DIRECT'}
            </span>
          </div>
          <div className="text-lg font-mono font-bold tracking-tight">
            {thd_percent.toFixed(1)}%
          </div>
          <div className="text-[10px] text-slate-500 font-mono mt-0.5">
            Threshold: &lt; 18.0%
          </div>
        </div>
      </div>

      {/* Forensic Diagnostics Explanations List */}
      <div className="p-3 rounded-lg bg-slate-950 border border-slate-800">
        <div className="text-[11px] font-mono font-semibold text-slate-400 mb-1.5 flex items-center gap-1.5">
          <FileText className="w-3.5 h-3.5 text-cyan-400" />
          <span>Active Forensic Anomaly Log:</span>
        </div>
        {key_indicators.length > 0 ? (
          <ul className="space-y-1">
            {key_indicators.map((ind, i) => (
              <li
                key={i}
                className="text-xs font-mono text-rose-300/90 flex items-start gap-1.5"
              >
                <span className="text-rose-500 font-bold">&bull;</span>
                <span>{ind}</span>
              </li>
            ))}
          </ul>
        ) : (
          <div className="text-xs font-mono text-emerald-400/90 flex items-center gap-1.5">
            <span className="w-2 h-2 rounded-full bg-emerald-500" />
            <span>Phonation characteristics match human anatomical baseline; no synthetic vocoder artifacts detected.</span>
          </div>
        )}
      </div>
    </div>
  );
};
