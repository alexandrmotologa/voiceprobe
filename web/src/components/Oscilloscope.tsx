import React, { useEffect, useRef } from 'react';
import { Activity } from 'lucide-react';
import { RiskLevel } from '../types/telemetry';

interface OscilloscopeProps {
  f0Hz: number;
  vadEnergy: number;
  isVoiced: boolean;
  jitter: number;
  shimmer: number;
  riskLevel: RiskLevel;
}

export const Oscilloscope: React.FC<OscilloscopeProps> = ({
  f0Hz,
  vadEnergy,
  isVoiced,
  jitter,
  shimmer,
  riskLevel,
}) => {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const phaseRef = useRef<number>(0);

  useEffect(() => {
    let animId: number;

    const render = () => {
      const canvas = canvasRef.current;
      if (!canvas) return;
      const ctx = canvas.getContext('2d');
      if (!ctx) return;

      const width = canvas.width;
      const height = canvas.height;
      const midY = height / 2;

      // Dark background
      ctx.fillStyle = '#0a0c14';
      ctx.fillRect(0, 0, width, height);

      // Technical oscilloscope grid
      ctx.strokeStyle = '#141824';
      ctx.lineWidth = 1;
      ctx.beginPath();
      for (let x = 0; x < width; x += 40) {
        ctx.moveTo(x, 0);
        ctx.lineTo(x, height);
      }
      for (let y = 0; y < height; y += 30) {
        ctx.moveTo(0, y);
        ctx.lineTo(width, y);
      }
      ctx.stroke();

      // Zero-crossing center line
      ctx.strokeStyle = '#1e2438';
      ctx.setLineDash([2, 4]);
      ctx.beginPath();
      ctx.moveTo(0, midY);
      ctx.lineTo(width, midY);
      ctx.stroke();
      ctx.setLineDash([]);

      // Waveform parameters
      const amplitude = isVoiced ? Math.min(Math.max(vadEnergy * 600, 15), height * 0.42) : 2.0;
      const frequency = f0Hz > 60 ? f0Hz : 140.0;
      const waveSpeed = 0.08;
      phaseRef.current += waveSpeed;

      // Color based on threat level
      let strokeColor = '#00f2fe'; // Cyan default
      let glowColor = 'rgba(0, 242, 254, 0.4)';
      if (riskLevel === 'CRITICAL') {
        strokeColor = '#f43f5e'; // Rose deepfake
        glowColor = 'rgba(244, 63, 94, 0.5)';
      } else if (riskLevel === 'SUSPICIOUS') {
        strokeColor = '#f59e0b'; // Amber
        glowColor = 'rgba(245, 158, 11, 0.4)';
      }

      ctx.save();
      ctx.strokeStyle = strokeColor;
      ctx.lineWidth = 2;
      ctx.shadowColor = glowColor;
      ctx.shadowBlur = 10;
      ctx.beginPath();

      const numPoints = width;
      for (let x = 0; x < numPoints; x++) {
        const t = (x / width) * 4.0 * Math.PI;
        // Jitter perturbation factor
        const jit = jitter > 0 ? (Math.sin(t * 11.0 + phaseRef.current) * jitter * 12.0) : 0;
        // Shimmer perturbation
        const shim = 1.0 + (shimmer > 0 ? (Math.sin(t * 3.5) * shimmer * 2.0) : 0);

        // Multi-harmonic acoustic waveform: fundamental + 2nd + 3rd harmonics
        const harmonic1 = Math.sin(t * (frequency / 80.0) + phaseRef.current + jit);
        const harmonic2 = 0.4 * Math.sin(2.0 * (t * (frequency / 80.0) + phaseRef.current));
        const harmonic3 = 0.15 * Math.sin(3.0 * (t * (frequency / 80.0) + phaseRef.current));

        const rawSample = (harmonic1 + harmonic2 + harmonic3) * amplitude * shim;
        const y = midY - rawSample;

        if (x === 0) {
          ctx.moveTo(x, y);
        } else {
          ctx.lineTo(x, y);
        }
      }
      ctx.stroke();
      ctx.restore();

      animId = requestAnimationFrame(render);
    };

    animId = requestAnimationFrame(render);
    return () => cancelAnimationFrame(animId);
  }, [f0Hz, vadEnergy, isVoiced, jitter, shimmer, riskLevel]);

  return (
    <div className="p-4 rounded-xl bg-[#0f111a] border border-slate-800/90 flex flex-col justify-between">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <Activity className="w-4 h-4 text-cyan-400" />
          <span className="text-xs font-mono font-semibold tracking-wider text-slate-300 uppercase">
            Real-Time Oscilloscope & VAD Waveform
          </span>
        </div>
        <div className="flex items-center gap-2 text-[10px] font-mono">
          <span className={`px-2 py-0.5 rounded border ${
            isVoiced
              ? 'bg-emerald-950/80 text-emerald-300 border-emerald-800'
              : 'bg-slate-900 text-slate-500 border-slate-800'
          }`}>
            {isVoiced ? 'VOICED SPEECH' : 'PAUSE / SILENCE'}
          </span>
          <span className="text-slate-400">
            F0: <strong className="text-white">{f0Hz > 0 ? `${f0Hz.toFixed(1)} Hz` : '---'}</strong>
          </span>
        </div>
      </div>

      <div className="relative w-full h-44 rounded-lg overflow-hidden border border-slate-800 bg-slate-950">
        <canvas
          ref={canvasRef}
          width={600}
          height={180}
          className="w-full h-full block"
        />
        {/* Scale labels */}
        <div className="absolute right-2 top-1.5 text-[9px] font-mono text-slate-500">
          +1.0 FS
        </div>
        <div className="absolute right-2 bottom-1.5 text-[9px] font-mono text-slate-500">
          -1.0 FS
        </div>
      </div>
    </div>
  );
};
