import React, { useState, useRef, useEffect } from 'react';
import { UploadCloud, FileAudio, CheckCircle2, AlertOctagon, AlertTriangle, X, Loader2, Play, Pause, Filter, Info } from 'lucide-react';
import { FileAnalysisResult, RiskLevel } from '../types/telemetry';

interface FileAnalyzerModalProps {
  isOpen: boolean;
  onClose: () => void;
}

type AudioFilterMode = 'FULL' | 'VOCODER_BAND' | 'LOWPASS_F0';

export const FileAnalyzerModal: React.FC<FileAnalyzerModalProps> = ({ isOpen, onClose }) => {
  const [analyzing, setAnalyzing] = useState(false);
  const [result, setResult] = useState<FileAnalysisResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Audio Playback & Scrubbing
  const [audioUrl, setAudioUrl] = useState<string | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [selectedFrameIdx, setSelectedFrameIdx] = useState<number | null>(null);
  const [filterMode, setFilterMode] = useState<AudioFilterMode>('FULL');

  const audioRef = useRef<HTMLAudioElement | null>(null);
  const audioCtxRef = useRef<AudioContext | null>(null);
  const filterNodeRef = useRef<BiquadFilterNode | null>(null);
  const sourceNodeRef = useRef<MediaElementAudioSourceNode | null>(null);

  useEffect(() => {
    return () => {
      if (audioUrl) {
        URL.revokeObjectURL(audioUrl);
      }
      if (audioCtxRef.current) {
        audioCtxRef.current.close();
      }
    };
  }, [audioUrl]);

  if (!isOpen) return null;

  // Initialize Web Audio Filter Graph
  const initAudioGraph = () => {
    if (!audioRef.current || audioCtxRef.current) return;

    const audioCtx = new (window.AudioContext || (window as any).webkitAudioContext)();
    audioCtxRef.current = audioCtx;

    const source = audioCtx.createMediaElementSource(audioRef.current);
    sourceNodeRef.current = source;

    const filterNode = audioCtx.createBiquadFilter();
    filterNodeRef.current = filterNode;

    source.connect(filterNode);
    filterNode.connect(audioCtx.destination);
    applyFilter(filterMode);
  };

  const applyFilter = (mode: AudioFilterMode) => {
    setFilterMode(mode);
    const filter = filterNodeRef.current;
    if (!filter) return;

    if (mode === 'VOCODER_BAND') {
      filter.type = 'bandpass';
      filter.frequency.value = 3100;
      filter.Q.value = 3.5;
    } else if (mode === 'LOWPASS_F0') {
      filter.type = 'lowpass';
      filter.frequency.value = 400;
      filter.Q.value = 1.0;
    } else {
      filter.type = 'allpass';
    }
  };

  const handleFileUpload = async (file: File) => {
    setAnalyzing(true);
    setError(null);
    setResult(null);

    // Create playable object URL
    if (audioUrl) {
      URL.revokeObjectURL(audioUrl);
    }
    const url = URL.createObjectURL(file);
    setAudioUrl(url);

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
      const resp = await fetch(`/examples/sample_calls/${type === 'human' ? 'human_customer_call.wav' : 'cloned_elevenlabs_vishing.wav'}`);
      if (resp.ok) {
        const blob = await resp.blob();
        const file = new File([blob], `${type}_sample.wav`, { type: 'audio/wav' });
        await handleFileUpload(file);
      } else {
        // Fallback to simulate
        await fetch(`/api/v1/simulate`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ type, duration_sec: 5 }),
        });
        setError(`Stream simulation started for ${type} sample. Telemetry active on main cockpit.`);
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setAnalyzing(false);
    }
  };

  const handleSeekToFrame = (frameIdx: number, offsetMs: number) => {
    setSelectedFrameIdx(frameIdx);
    if (audioRef.current) {
      audioRef.current.currentTime = offsetMs / 1000;
      if (!isPlaying) {
        initAudioGraph();
        audioRef.current.play();
        setIsPlaying(true);
      }
    }
  };

  const togglePlay = () => {
    if (!audioRef.current) return;
    initAudioGraph();

    if (isPlaying) {
      audioRef.current.pause();
      setIsPlaying(false);
    } else {
      audioRef.current.play();
      setIsPlaying(true);
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
      <div className="w-full max-w-4xl rounded-xl bg-[#0f111a] border border-slate-800 shadow-2xl overflow-hidden flex flex-col max-h-[92vh]">
        {/* Modal Header */}
        <div className="p-4 border-b border-slate-800 flex items-center justify-between bg-slate-950/80">
          <div className="flex items-center gap-2">
            <FileAudio className="w-5 h-5 text-cyan-400" />
            <h2 className="text-sm font-mono font-bold text-white tracking-wide">
              FORENSIC AUDIO FILE ANALYZER & INTERACTIVE WAVEFORM SCRUBBER
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
              className="px-2.5 py-1 rounded bg-slate-900 hover:bg-slate-800 text-emerald-400 border border-slate-700 text-xs font-mono cursor-pointer transition"
            >
              Test Human Phonation
            </button>
            <button
              onClick={() => handlePresetSample('cloned')}
              className="px-2.5 py-1 rounded bg-slate-900 hover:bg-slate-800 text-rose-400 border border-slate-700 text-xs font-mono cursor-pointer transition"
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

          {/* Audio Player & Forensic Filter Controls */}
          {audioUrl && (
            <div className="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-3">
              <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
                <div className="flex items-center gap-3">
                  <button
                    onClick={togglePlay}
                    className="w-9 h-9 rounded-full bg-cyan-500 hover:bg-cyan-400 text-black flex items-center justify-center transition cursor-pointer shadow-[0_0_12px_rgba(0,242,254,0.3)]"
                  >
                    {isPlaying ? <Pause className="w-4 h-4 fill-black" /> : <Play className="w-4 h-4 fill-black ml-0.5" />}
                  </button>
                  <div className="font-mono text-xs text-slate-300">
                    <span className="text-white font-bold">{currentTime.toFixed(2)}s</span>
                    <span className="text-slate-500"> / {result ? `${result.duration_sec.toFixed(2)}s` : '--'}</span>
                  </div>
                </div>

                {/* Filter Band Toggles */}
                <div className="flex items-center flex-wrap gap-1.5 text-xs font-mono">
                  <span className="text-slate-500 flex items-center gap-1 mr-1">
                    <Filter className="w-3.5 h-3.5 text-cyan-400" /> Filter:
                  </span>
                  <button
                    onClick={() => applyFilter('FULL')}
                    className={`px-2 py-0.5 rounded border transition cursor-pointer ${
                      filterMode === 'FULL'
                        ? 'bg-cyan-950 text-cyan-300 border-cyan-800'
                        : 'bg-slate-900 text-slate-400 border-slate-800'
                    }`}
                  >
                    Full Audio
                  </button>
                  <button
                    onClick={() => applyFilter('VOCODER_BAND')}
                    className={`px-2 py-0.5 rounded border transition cursor-pointer ${
                      filterMode === 'VOCODER_BAND'
                        ? 'bg-rose-950 text-rose-300 border-rose-800 font-bold'
                        : 'bg-slate-900 text-slate-400 border-slate-800'
                    }`}
                    title="Bandpass 2.8kHz - 3.4kHz: Hear only the vocoder transposed-convolution artifacts"
                  >
                    Vocoder Band (2.8 - 3.4 kHz)
                  </button>
                  <button
                    onClick={() => applyFilter('LOWPASS_F0')}
                    className={`px-2 py-0.5 rounded border transition cursor-pointer ${
                      filterMode === 'LOWPASS_F0'
                        ? 'bg-indigo-950 text-indigo-300 border-indigo-800'
                        : 'bg-slate-900 text-slate-400 border-slate-800'
                    }`}
                    title="Lowpass 400Hz: Hear the isolated vocal cord fundamental phonation"
                  >
                    Glottal F0 (&lt; 400 Hz)
                  </button>
                </div>
              </div>

              <audio
                ref={audioRef}
                src={audioUrl}
                onTimeUpdate={() => {
                  if (audioRef.current) {
                    setCurrentTime(audioRef.current.currentTime);
                    if (result) {
                      const curMs = audioRef.current.currentTime * 1000;
                      const active = result.frames.findIndex(
                        f => curMs >= f.timestamp_offset_ms && curMs < f.timestamp_offset_ms + 150
                      );
                      if (active !== -1) {
                        setSelectedFrameIdx(active);
                      }
                    }
                  }
                }}
                onEnded={() => setIsPlaying(false)}
                className="hidden"
              />
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

              {/* Sliding Window Frames Scrubber Chart */}
              <div className="p-4 rounded-xl bg-slate-950 border border-slate-800">
                <div className="flex items-center justify-between mb-2 text-xs font-mono text-slate-300">
                  <span className="flex items-center gap-1.5">
                    <Info className="w-3.5 h-3.5 text-cyan-400" />
                    CLICK ANY WINDOW TO JUMP AUDIO (300ms SLICES, 150ms HOP)
                  </span>
                  <span>AVG: {(result.average_score * 100).toFixed(1)}%</span>
                </div>
                <div className="grid grid-cols-12 sm:grid-cols-16 md:grid-cols-24 gap-1.5 max-h-44 overflow-y-auto p-1 bg-black/40 rounded border border-slate-900">
                  {result.frames.map((frame, idx) => {
                    const scorePct = frame.instant_score * 100;
                    const isSelected = selectedFrameIdx === idx;
                    let barColor = 'bg-emerald-500';
                    if (frame.instant_score >= 0.8) barColor = 'bg-rose-500';
                    else if (frame.instant_score >= 0.35) barColor = 'bg-amber-500';
                    if (!frame.is_voiced) barColor = 'bg-slate-800';

                    return (
                      <button
                        key={idx}
                        onClick={() => handleSeekToFrame(idx, frame.timestamp_offset_ms)}
                        title={`Frame #${frame.frame_index} (${frame.timestamp_offset_ms}ms): ${scorePct.toFixed(1)}% ${frame.risk_level}`}
                        className={`h-11 rounded-sm flex flex-col items-center justify-between p-0.5 text-[8px] font-mono text-white transition cursor-pointer hover:scale-110 ${barColor} ${
                          isSelected ? 'ring-2 ring-white scale-110 z-10 shadow-[0_0_10px_white]' : 'opacity-85'
                        }`}
                      >
                        <span className="opacity-75">{frame.frame_index}</span>
                        <span className="font-bold">{frame.is_voiced ? Math.round(scorePct) : '-'}</span>
                      </button>
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
