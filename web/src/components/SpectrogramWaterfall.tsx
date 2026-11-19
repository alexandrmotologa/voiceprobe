import React, { useEffect, useRef } from 'react';
import { Waves } from 'lucide-react';

interface SpectrogramWaterfallProps {
  melEnergies?: number[];
  isVoiced?: boolean;
}

export const SpectrogramWaterfall: React.FC<SpectrogramWaterfallProps> = ({
  melEnergies,
  isVoiced = true,
}) => {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const historyRef = useRef<number[][]>([]);
  const maxHistory = 100; // 100 frames of history (~15 seconds at 150ms hop)

  // Map normalized Mel band energy (0..1) to thermal heatmap color (Viridis / Inferno style)
  const getHeatmapColor = (val: number): [number, number, number] => {
    // Clamp to 0..1
    const v = Math.min(Math.max(val, 0), 1);
    if (v < 0.2) {
      // Deep dark blue / purple
      const t = v / 0.2;
      return [Math.floor(10 + t * 40), Math.floor(12 + t * 20), Math.floor(25 + t * 100)];
    } else if (v < 0.45) {
      // Purple to Magenta
      const t = (v - 0.2) / 0.25;
      return [Math.floor(50 + t * 130), Math.floor(32 + t * 20), Math.floor(125 + t * 50)];
    } else if (v < 0.75) {
      // Magenta to Orange
      const t = (v - 0.45) / 0.3;
      return [Math.floor(180 + t * 65), Math.floor(52 + t * 90), Math.floor(175 - t * 140)];
    } else {
      // Orange to Bright Yellow / White
      const t = (v - 0.75) / 0.25;
      return [245 + Math.floor(t * 10), Math.floor(142 + t * 105), Math.floor(35 + t * 210)];
    }
  };

  useEffect(() => {
    if (melEnergies && melEnergies.length > 0) {
      // Push copy of normalized mel energies (40 bands)
      // Normalize values if raw log-energies
      const normalized = melEnergies.map(e => {
        // Typical log energy ranges from -8 to 8
        const norm = (e + 6.0) / 10.0;
        return isVoiced ? Math.min(Math.max(norm, 0), 1) : norm * 0.2;
      });

      historyRef.current.push(normalized);
      if (historyRef.current.length > maxHistory) {
        historyRef.current.shift();
      }
    }
  }, [melEnergies, isVoiced]);

  // Render loop
  useEffect(() => {
    let animId: number;

    const render = () => {
      const canvas = canvasRef.current;
      if (!canvas) return;
      const ctx = canvas.getContext('2d');
      if (!ctx) return;

      const width = canvas.width;
      const height = canvas.height;

      // Clear background
      ctx.fillStyle = '#0a0c14';
      ctx.fillRect(0, 0, width, height);

      const history = historyRef.current;
      const totalColumns = maxHistory;
      const colWidth = width / totalColumns;
      const numBands = 40;
      const bandHeight = height / numBands;

      // Draw spectrogram grid columns
      for (let xIdx = 0; xIdx < history.length; xIdx++) {
        const frameBands = history[xIdx];
        const xPos = (totalColumns - history.length + xIdx) * colWidth;

        for (let b = 0; b < numBands; b++) {
          // Band 0 is lowest freq (bottom), band 39 is highest freq (top)
          const val = frameBands[b] || 0;
          const [r, g, bColor] = getHeatmapColor(val);
          const yPos = height - (b + 1) * bandHeight;

          ctx.fillStyle = `rgb(${r}, ${g}, ${bColor})`;
          ctx.fillRect(Math.floor(xPos), Math.floor(yPos), Math.ceil(colWidth), Math.ceil(bandHeight));
        }
      }

      // Draw Vocoder Artifact Sub-band Highlight (2800Hz - 3400Hz corresponds roughly to Mel bands 28-34)
      const vocoderTopY = height - 34 * bandHeight;
      const vocoderHeight = 6 * bandHeight;
      ctx.fillStyle = 'rgba(244, 63, 94, 0.08)';
      ctx.fillRect(0, vocoderTopY, width, vocoderHeight);

      ctx.strokeStyle = 'rgba(244, 63, 94, 0.35)';
      ctx.setLineDash([3, 3]);
      ctx.beginPath();
      ctx.moveTo(0, vocoderTopY);
      ctx.lineTo(width, vocoderTopY);
      ctx.moveTo(0, vocoderTopY + vocoderHeight);
      ctx.lineTo(width, vocoderTopY + vocoderHeight);
      ctx.stroke();
      ctx.setLineDash([]);

      // Vocoder boundary text label
      ctx.fillStyle = 'rgba(244, 63, 94, 0.8)';
      ctx.font = '9px monospace';
      ctx.fillText('VOCODER ARTIFACT ZONE [2.8 - 3.4 kHz]', 8, vocoderTopY + 12);

      animId = requestAnimationFrame(render);
    };

    animId = requestAnimationFrame(render);
    return () => cancelAnimationFrame(animId);
  }, []);

  return (
    <div className="p-4 rounded-xl bg-[#0f111a] border border-slate-800/90 flex flex-col justify-between">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <Waves className="w-4 h-4 text-cyan-400" />
          <span className="text-xs font-mono font-semibold tracking-wider text-slate-300 uppercase">
            Mel Filterbank Waterfall Spectrogram (40 Sub-Bands)
          </span>
        </div>
        <div className="flex items-center gap-3 text-[10px] font-mono text-slate-400">
          <span>0Hz (BASE) &rarr; 4000Hz (NYQUIST)</span>
          <span className="flex items-center gap-1 text-cyan-300">
            <span className="w-2 h-2 rounded-full bg-cyan-400 inline-block" /> 60 FPS
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

        {/* Frequency scale overlay on left */}
        <div className="absolute left-1.5 inset-y-1 flex flex-col justify-between text-[9px] font-mono text-slate-400/80 pointer-events-none drop-shadow">
          <span>4.0 kHz</span>
          <span className="text-rose-400 font-semibold">3.0 kHz</span>
          <span>2.0 kHz</span>
          <span>1.0 kHz</span>
          <span>0.1 kHz</span>
        </div>
      </div>
    </div>
  );
};
