import React, { useEffect, useState } from 'react';
import { Archive, X, AlertOctagon, Download, RefreshCw, Volume2 } from 'lucide-react';
import { IncidentRecord } from '../types/telemetry';

interface IncidentsDrawerProps {
  isOpen: boolean;
  onClose: () => void;
}

export const IncidentsDrawer: React.FC<IncidentsDrawerProps> = ({ isOpen, onClose }) => {
  const [incidents, setIncidents] = useState<IncidentRecord[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchIncidents = async () => {
    setLoading(true);
    try {
      const resp = await fetch('/api/v1/incidents');
      if (resp.ok) {
        const data = await resp.json();
        setIncidents(data || []);
      }
    } catch (err) {
      console.error('Failed to load incidents', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (isOpen) {
      fetchIncidents();
    }
  }, [isOpen]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/60 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="w-full max-w-xl h-full bg-[#0f111a] border-l border-slate-800 shadow-2xl flex flex-col">
        {/* Drawer Header */}
        <div className="p-4 border-b border-slate-800 flex items-center justify-between bg-slate-950/90">
          <div className="flex items-center gap-2">
            <Archive className="w-5 h-5 text-rose-400" />
            <h2 className="text-sm font-mono font-bold text-white tracking-wide">
              FORENSIC INCIDENT RECORDER ARCHIVE
            </h2>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={fetchIncidents}
              disabled={loading}
              className="p-1 rounded text-slate-400 hover:text-white hover:bg-slate-800 transition cursor-pointer"
              title="Refresh"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={onClose}
              className="p-1 rounded text-slate-400 hover:text-white hover:bg-slate-800 transition cursor-pointer"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Drawer Body */}
        <div className="flex-1 overflow-y-auto p-4 space-y-3">
          {incidents.length === 0 ? (
            <div className="py-12 text-center font-mono text-slate-500 text-xs">
              No recorded fraud incidents found. Incidents are automatically archived when sustained deepfake attacks exceed the 80% critical threshold.
            </div>
          ) : (
            incidents.map((inc, i) => (
              <div
                key={i}
                className="p-4 rounded-xl bg-slate-950 border border-slate-800/90 hover:border-slate-700 transition"
              >
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <AlertOctagon className="w-4 h-4 text-rose-400" />
                    <span className="font-mono text-xs font-bold text-white">
                      {inc.call_id}
                    </span>
                  </div>
                  <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-rose-950 text-rose-300 border border-rose-800">
                    PEAK: {(inc.peak_score * 100).toFixed(1)}%
                  </span>
                </div>

                <p className="text-xs text-slate-400 font-mono mb-2">
                  {inc.summary}
                </p>

                {inc.indicators && inc.indicators.length > 0 && (
                  <div className="mb-3 space-y-1">
                    {inc.indicators.map((ind, idx) => (
                      <div key={idx} className="text-[11px] font-mono text-rose-400/90 flex items-center gap-1.5">
                        <span className="text-rose-500">&bull;</span>
                        <span>{ind}</span>
                      </div>
                    ))}
                  </div>
                )}

                {/* Audio Player */}
                {inc.wav_file && (
                  <div className="mt-3 pt-3 border-t border-slate-900 flex flex-col gap-2">
                    <div className="flex items-center justify-between text-[11px] font-mono text-slate-400">
                      <span className="flex items-center gap-1">
                        <Volume2 className="w-3.5 h-3.5 text-cyan-400" /> Forensic Audio Capture
                      </span>
                      <a
                        href={`/api/v1/incidents/${inc.wav_file}`}
                        download
                        className="text-cyan-400 hover:underline flex items-center gap-1"
                      >
                        <Download className="w-3 h-3" /> Download WAV
                      </a>
                    </div>
                    <audio
                      src={`/api/v1/incidents/${inc.wav_file}`}
                      controls
                      className="w-full h-8 rounded bg-slate-900 border border-slate-800"
                    />
                  </div>
                )}
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
};
