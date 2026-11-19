import React, { useState } from 'react';
import { UploadCloud, FileAudio, CheckCircle2, AlertOctagon, AlertTriangle, X, Loader2 } from 'lucide-react';
import { FileAnalysisResult, RiskLevel } from '../types/telemetry';

interface FileAnalyzerModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const FileAnalyzerModal: React.FC<FileAnalyzerModalProps> = ({ isOpen, onClose }) => {
  const [analyzing, setAnalyzing] = useState(false);
  const [result, setResult] = useState<FileAnalysisResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  if (!isOpen) return null;

  const handleFileUpload = async (file: File) => {
    setAnalyzing(true);
    setError(null);
    setResult(null);

    const formData = new FormData();
    formData.append('audio', file);

    try {
      const resp = await fetch('/api/v1/analyze', {
        method: 'POST',
        body: formData,
      });

      if (!resp.ok) {
        throw new Error(`Server returned HTTP ${resp.status}`);
      }

      const data: FileAnalysisResult = await resp.json();
      setResult(data);
    } catch (err: any) {
      setError(err.message || 'Failed to analyze audio file');
    } finally {
      setAnalyzing(false);
    }
  };

  const handlePresetSample = async (type: 'human' | 'cloned') => {
    setAnalyzing(true);
    setError(null);
    setResult(null);

    try {
      const response = await fetch(`/api/v1/simulate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type, duration_sec: 5 }),
      });
      if (!response.ok) {
        throw new Error('Simulation failed to trigger');
      }

      setError(`Stream simulation started for ${type} sample. Inspect live telemetry stream in cockpit.`);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setAnalyzing(false);
    }
  };

  const getVerdictStyle = (level: RiskLevel) => {
    switch (level) {
      case 'CRITICAL':
        return {
          bg: 'bg-rose-950/80 border-rose-800 text-rose-300',
          title: 'CRITICAL: SYNTHETIC DEEPFAKE DETECTED',
          icon: AlertOctagon,
        };
      case 'SUSPICIOUS':
        return {
          bg: 'bg-amber-950/80 border-amber-800 text-amber-300',
          title: 'SUSPICIOUS: ACOUSTIC IRREGULARITIES DETECTED',
          icon: AlertTriangle,
        };
      default:
        return {
          bg: 'bg-emerald-950/80 border-emerald-800 text-emerald-300',
          title: 'CLEAN: NATURAL HUMAN PHONATION',
          icon: CheckCircle2,
        };
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="w-full max-w-3xl rounded-xl bg-[#0f111a] border border-slate-800 shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
        {/* Modal Header */}
        <div className="p-4 border-b border-slate-800 flex items-center justify-between bg-slate-950/80">
          <div className="flex items-center gap-2">
            <FileAudio className="w-5 h-5 text-cyan-400" />
            <h2 className="text-sm font-mono font-bold text-white tracking-wide">
              FORENSIC AUDIO FILE ANALYZER (WAV PCM 8kHz / 16kHz)
            </h2>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded text-slate-400 hover:text-white hover:bg-slate-800 transition cursor-pointer"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Modal Content */}
        <div className="p-5 overflow-y-auto space-y-4">
          {/* Upload Area */}
          <div
            onDragOver={(e) => e.preventDefault()}
            onDrop={(e) => {
              e.preventDefault();
              if (e.dataTransfer.files && e.dataTransfer.files[0]) {
                handleFileUpload(e.dataTransfer.files[0]);
              }
            }}
            className="border-2 border-dashed border-slate-700 hover:border-cyan-500/80 rounded-xl p-6 text-center bg-slate-950/50 transition cursor-pointer flex flex-col items-center justify-center relative group"
          >
            <input
              type="file"
              accept=".wav"
              onChange={(e) => {
                if (e.target.files && e.target.files[0]) {
                  handleFileUpload(e.target.files[0]);
                }
              }}
              className="absolute inset-0 opacity-0 cursor-pointer"
            />
            <UploadCloud className="w-10 h-10 text-cyan-400 mb-2 group-hover:scale-110 transition" />
            <p className="text-sm font-mono text-slate-200 mb-1">
              Drag & Drop WAV audio recording or click to browse
            </p>
            <p className="text-xs text-slate-500 font-mono">
              Accepts 8000Hz / 16000Hz 16-bit PCM uncompressed WAV files
            </p>
          </div>

          {/* Quick Presets */}
          <div className="flex items-center justify-center gap-3">
            <span className="text-xs font-mono text-slate-400">Quick Test:</span>
            <button
              onClick={() => handlePresetSample('human')}
              className="px-2.5 py-1 rounded bg-slate-900 hover:bg-slate-800 text-emerald-400 border border-slate-700 text-xs font-mono cursor-pointer"
            >
              Test Human Phonation
            </button>
            <button
              onClick={() => handlePresetSample('cloned')}
              className="px-2.5 py-1 rounded bg-slate-900 hover:bg-slate-800 text-rose-400 border border-slate-700 text-xs font-mono cursor-pointer"
            >
              Test Cloned Deepfake
            </button>
          </div>

          {analyzing && (
            <div className="py-6 flex flex-col items-center justify-center text-cyan-400 font-mono gap-2">
              <Loader2 className="w-6 h-6 animate-spin" />
              <span className="text-xs">Processing DSP STFT, Mel filterbank and neuromuscular micro-tremor...</span>
            </div>
          )}

          {error && (
            <div className="p-3 rounded-lg bg-rose-950/60 border border-rose-800 text-rose-300 text-xs font-mono">
              {error}
            </div>
          )}

          {/* Results Display */}
          {result && (
            <div className="space-y-4 animate-in fade-in">
              {/* Verdict Banner */}
              {(() => {
                const style = getVerdictStyle(result.overall_threat_level);
                const Icon = style.icon;
                return (
                  <div className={`p-4 rounded-xl border flex items-center justify-between ${style.bg}`}>
                    <div className="flex items-center gap-3">
                      <Icon className="w-6 h-6 flex-shrink-0" />
                      <div>
                        <div className="text-sm font-mono font-bold">{style.title}</div>
                        <div className="text-xs opacity-90 font-mono">
                          File: {result.file_name} ({result.duration_sec.toFixed(2)}s, {result.sample_rate}Hz)
                        </div>
                      </div>
                    </div>
                    <div className="text-right font-mono">
                      <div className="text-2xl font-extrabold">{(result.peak_score * 100).toFixed(1)}%</div>
                      <div className="text-[10px] opacity-75">PEAK SCORE</div>
                    </div>
                  </div>
                );
              })()}

              {/* Key Indicators */}
              <div className="p-4 rounded-xl bg-slate-950 border border-slate-800">
                <div className="text-xs font-mono font-semibold text-slate-300 mb-2">
                  ACOUSTIC DIAGNOSTIC SUMMARY:
                </div>
                <ul className="space-y-1.5 font-mono text-xs text-slate-400">
                  {result.summary_indicators.map((ind, i) => (
                    <li key={i} className="flex items-start gap-2">
                      <span className="text-cyan-400 font-bold">&bull;</span>
                      <span>{ind}</span>
                    </li>
                  ))}
                </ul>
              </div>

              {/* Sliding Window Frames Chart */}
              <div className="p-4 rounded-xl bg-slate-950 border border-slate-800">
                <div className="flex items-center justify-between mb-3 text-xs font-mono text-slate-300">
                  <span>300ms SLIDING ANALYSIS FRAMES ({result.frames.length} TOTAL)</span>
                  <span>AVG SCORE: {(result.average_score * 100).toFixed(1)}%</span>
                </div>
                <div className="grid grid-cols-12 sm:grid-cols-16 md:grid-cols-24 gap-1.5 max-h-40 overflow-y-auto p-1">
                  {result.frames.map((frame, idx) => {
                    const scorePct = frame.instant_score * 100;
                    let barColor = 'bg-emerald-500';
                    if (frame.instant_score >= 0.8) barColor = 'bg-rose-500';
                    else if (frame.instant_score >= 0.35) barColor = 'bg-amber-500';
                    if (!frame.is_voiced) barColor = 'bg-slate-800';

                    return (
                      <div
                        key={idx}
                        title={`Frame #${frame.frame_index} (${frame.timestamp_offset_ms}ms): ${scorePct.toFixed(1)}% ${frame.risk_level}`}
                        className={`h-9 rounded-sm flex items-end justify-center p-0.5 text-[8px] font-mono text-white transition hover:scale-110 ${barColor}`}
                      >
                        {frame.is_voiced ? Math.round(scorePct) : '-'}
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
