import React from 'react';
import { Shield, ShieldAlert, Activity, Cpu, Clock, PhoneCall, Radio, UploadCloud, Archive, Sliders } from 'lucide-react';
import { ServerStatus } from '../types/telemetry';
import { MicrophoneProbe } from './MicrophoneProbe';

interface HeaderProps {
  status: ServerStatus | null;
  sseConnected: boolean;
  onSimulate: (type: 'cloned' | 'human') => void;
  onOpenUpload: () => void;
  onOpenIncidents: () => void;
  activeCount: number;
  currentPolicy: string;
  onSelectPolicy: (policy: string) => void;
  onMicStreamActive: (active: boolean) => void;
}

export const Header: React.FC<HeaderProps> = ({
  status,
  sseConnected,
  onSimulate,
  onOpenUpload,
  onOpenIncidents,
  activeCount,
  currentPolicy,
  onSelectPolicy,
  onMicStreamActive,
}) => {
  return (
    <header className="border-b border-slate-800/80 bg-[#0c0e15]/95 backdrop-blur sticky top-0 z-40 px-4 py-3">
      <div className="max-w-7xl mx-auto flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        {/* Brand identity */}
        <div className="flex items-center gap-3">
          <div className="relative flex items-center justify-center w-10 h-10 rounded-lg bg-gradient-to-br from-cyan-500/20 via-blue-600/20 to-indigo-600/30 border border-cyan-500/40 text-cyan-400 shadow-[0_0_15px_rgba(0,242,254,0.15)]">
            <Shield className="w-5 h-5 animate-pulse-subtle" />
            <div className="absolute -top-1 -right-1 w-2.5 h-2.5 rounded-full bg-cyan-400 animate-ping" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-lg font-bold tracking-tight text-white flex items-center gap-1.5 font-mono">
                VOICE<span className="text-cyan-400">PROBE</span>
              </h1>
              <span className="px-1.5 py-0.5 text-[10px] font-mono tracking-wider font-semibold rounded bg-cyan-950/80 text-cyan-400 border border-cyan-800/50">
                SENTINEL {status?.version || '1.0.0'}
              </span>
            </div>
            <p className="text-xs text-slate-400 hidden sm:block">
              Real-time deepfake & voice cloning fraud detection proxy for telephony
            </p>
          </div>
        </div>

        {/* Engine Telemetry HUD */}
        <div className="flex items-center flex-wrap gap-2 text-xs">
          {/* SSE Connection */}
          <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-slate-900/80 border border-slate-800 text-slate-300 font-mono">
            <Radio className={`w-3.5 h-3.5 ${sseConnected ? 'text-emerald-400 animate-pulse' : 'text-rose-400'}`} />
            <span>{sseConnected ? 'STREAM LIVE' : 'CONNECTING...'}</span>
          </div>

          {/* Active Policy Selector */}
          <div className="flex items-center gap-1 px-2.5 py-1 rounded-md bg-slate-900/80 border border-slate-800 text-slate-300 font-mono">
            <Sliders className="w-3.5 h-3.5 text-cyan-400" />
            <select
              value={currentPolicy}
              onChange={(e) => onSelectPolicy(e.target.value)}
              className="bg-transparent text-white font-semibold outline-none cursor-pointer text-xs"
              title="FinTech Banking Security Policy"
            >
              <option value="STANDARD" className="bg-slate-950 text-slate-200">Policy: Standard Surveillance</option>
              <option value="HIGH_VALUE_WIRE" className="bg-slate-950 text-amber-400 font-bold">Policy: High-Value Wire (&gt; 10k EUR)</option>
              <option value="EXECUTIVE_PROTECTION" className="bg-slate-950 text-rose-400 font-bold">Policy: Executive VIP (Auto-Kill)</option>
            </select>
          </div>

          {/* Active Calls */}
          <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-slate-900/80 border border-slate-800 text-slate-300 font-mono">
            <PhoneCall className="w-3.5 h-3.5 text-cyan-400" />
            <span>CALLS: <strong className="text-white">{activeCount}</strong></span>
          </div>

          {/* Detection Latency Benchmark */}
          <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-slate-900/80 border border-slate-800 text-slate-300 font-mono">
            <Clock className="w-3.5 h-3.5 text-amber-400" />
            <span>LATENCY: <strong className="text-emerald-400">&lt; 1.0ms</strong></span>
          </div>

          {/* RAM Footprint */}
          <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-slate-900/80 border border-slate-800 text-slate-300 font-mono hidden lg:flex">
            <Cpu className="w-3.5 h-3.5 text-indigo-400" />
            <span>RAM: <strong className="text-cyan-400">0.28MB</strong></span>
          </div>
        </div>

        {/* Quick Actions & Live Mic Probe */}
        <div className="flex items-center flex-wrap gap-2">
          {/* Live Browser Mic Probe */}
          <MicrophoneProbe onStreamActive={onMicStreamActive} />

          <button
            onClick={() => onSimulate('cloned')}
            className="flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-semibold rounded-md bg-rose-950/60 hover:bg-rose-900/80 text-rose-300 border border-rose-800/60 transition-all hover:shadow-[0_0_12px_rgba(244,63,94,0.3)] cursor-pointer"
            title="Simulate incoming synthetic cloned voice call (ElevenLabs / F5-TTS)"
          >
            <ShieldAlert className="w-3.5 h-3.5 text-rose-400" />
            <span>Simulate Attack</span>
          </button>

          <button
            onClick={() => onSimulate('human')}
            className="flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-semibold rounded-md bg-emerald-950/60 hover:bg-emerald-900/80 text-emerald-300 border border-emerald-800/60 transition-all hover:shadow-[0_0_12px_rgba(16,185,129,0.3)] cursor-pointer"
            title="Simulate authentic organic human customer call"
          >
            <Activity className="w-3.5 h-3.5 text-emerald-400" />
            <span>Simulate Clean</span>
          </button>

          <button
            onClick={onOpenUpload}
            className="p-1.5 text-slate-400 hover:text-white bg-slate-900 hover:bg-slate-800 border border-slate-800 rounded-md transition cursor-pointer"
            title="Inspect audio file (WAV)"
          >
            <UploadCloud className="w-4 h-4" />
          </button>

          <button
            onClick={onOpenIncidents}
            className="p-1.5 text-slate-400 hover:text-white bg-slate-900 hover:bg-slate-800 border border-slate-800 rounded-md transition cursor-pointer"
            title="Incident archives"
          >
            <Archive className="w-4 h-4" />
          </button>
        </div>
      </div>
    </header>
  );
};
