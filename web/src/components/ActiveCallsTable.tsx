import React from 'react';
import { Phone, PhoneOff, Radio, ShieldAlert, CheckCircle, Eye } from 'lucide-react';
import { CallSession, RiskLevel } from '../types/telemetry';

interface ActiveCallsTableProps {
  calls: CallSession[];
  selectedCallId?: string;
  onSelectCall: (callId: string) => void;
  onDropCall: (callId: string) => void;
  onTriggerSimulation: (type: 'cloned' | 'human') => void;
}

export const ActiveCallsTable: React.FC<ActiveCallsTableProps> = ({
  calls,
  selectedCallId,
  onSelectCall,
  onDropCall,
  onTriggerSimulation,
}) => {
  const getBadge = (risk: RiskLevel, peakScore: number) => {
    if (risk === 'CRITICAL' || peakScore >= 0.8) {
      return (
        <span className="px-2 py-0.5 rounded text-[11px] font-mono font-bold bg-rose-950/80 text-rose-300 border border-rose-800 flex items-center gap-1 w-fit">
          <ShieldAlert className="w-3 h-3 text-rose-400" /> CRITICAL
        </span>
      );
    } else if (risk === 'SUSPICIOUS' || peakScore >= 0.35) {
      return (
        <span className="px-2 py-0.5 rounded text-[11px] font-mono font-bold bg-amber-950/80 text-amber-300 border border-amber-800 flex items-center gap-1 w-fit">
          SUSPICIOUS
        </span>
      );
    }
    return (
      <span className="px-2 py-0.5 rounded text-[11px] font-mono font-bold bg-emerald-950/80 text-emerald-300 border border-emerald-800 flex items-center gap-1 w-fit">
        <CheckCircle className="w-3 h-3 text-emerald-400" /> CLEAN
      </span>
    );
  };

  return (
    <div className="p-5 rounded-xl bg-[#0f111a] border border-slate-800/90">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2 mb-3">
        <div className="flex items-center gap-2">
          <Phone className="w-4 h-4 text-cyan-400" />
          <h2 className="text-xs font-mono font-semibold tracking-wider text-slate-300 uppercase">
            Active Telephony Ingestion Streams ({calls.length})
          </h2>
        </div>
        <div className="text-[11px] font-mono text-slate-400 flex items-center gap-2 flex-wrap">
          <span>SIP RFC 3261</span>
          <span>&bull;</span>
          <span>UDP RTP RFC 3550</span>
          <span>&bull;</span>
          <span>Twilio WebSocket</span>
        </div>
      </div>

      {calls.length === 0 ? (
        <div className="py-8 px-4 rounded-lg border border-dashed border-slate-800 bg-slate-950/40 text-center flex flex-col items-center justify-center">
          <Radio className="w-8 h-8 text-slate-600 mb-2 animate-pulse" />
          <p className="text-sm font-mono text-slate-400 mb-1">
            No active telephony streams currently ingested.
          </p>
          <p className="text-xs text-slate-500 max-w-md mb-4">
            Connect your PBX / SIP Trunk, configure Twilio Media Stream to <code className="text-cyan-400">/stream/twilio</code>, or start an instant acoustic simulation.
          </p>
          <div className="flex items-center gap-3">
            <button
              onClick={() => onTriggerSimulation('cloned')}
              className="px-3 py-1.5 rounded bg-rose-950/60 hover:bg-rose-900 text-rose-300 border border-rose-800 text-xs font-mono font-semibold cursor-pointer transition"
            >
              Simulate Cloned Vishing Call
            </button>
            <button
              onClick={() => onTriggerSimulation('human')}
              className="px-3 py-1.5 rounded bg-emerald-950/60 hover:bg-emerald-900 text-emerald-300 border border-emerald-800 text-xs font-mono font-semibold cursor-pointer transition"
            >
              Simulate Human Phonation
            </button>
          </div>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs font-mono">
            <thead>
              <tr className="border-b border-slate-800 text-slate-400 bg-slate-950/60">
                <th className="py-2.5 px-3">CALL ID</th>
                <th className="py-2.5 px-3">TRANSPORT</th>
                <th className="py-2.5 px-3">CODEC</th>
                <th className="py-2.5 px-3">SAMPLE RATE</th>
                <th className="py-2.5 px-3">PEAK SCORE</th>
                <th className="py-2.5 px-3">VERDICT</th>
                <th className="py-2.5 px-3">SIP INJECTION</th>
                <th className="py-2.5 px-3 text-right">ACTIONS</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60">
              {calls.map((call) => {
                const isSelected = call.call_id === selectedCallId;
                const peakPct = (call.peak_score * 100);

                return (
                  <tr
                    key={call.call_id}
                    className={`transition-colors ${
                      isSelected
                        ? 'bg-cyan-950/20 border-l-2 border-cyan-400'
                        : 'hover:bg-slate-900/50'
                    }`}
                  >
                    <td className="py-2.5 px-3 font-semibold text-white flex items-center gap-2">
                      <span className="w-2 h-2 rounded-full bg-cyan-400 animate-ping" />
                      <span>{call.call_id}</span>
                    </td>
                    <td className="py-2.5 px-3 text-slate-300 uppercase">
                      {call.transport}
                    </td>
                    <td className="py-2.5 px-3 text-slate-400">
                      {call.codec}
                    </td>
                    <td className="py-2.5 px-3 text-slate-400">
                      {call.sample_rate} Hz
                    </td>
                    <td className="py-2.5 px-3">
                      <div className="flex items-center gap-2">
                        <div className="w-16 h-1.5 rounded-full bg-slate-800 overflow-hidden">
                          <div
                            className={`h-full ${
                              peakPct >= 80
                                ? 'bg-rose-500'
                                : peakPct >= 35
                                ? 'bg-amber-500'
                                : 'bg-emerald-500'
                            }`}
                            style={{ width: `${Math.min(peakPct, 100)}%` }}
                          />
                        </div>
                        <span className="text-white font-bold">{peakPct.toFixed(1)}%</span>
                      </div>
                    </td>
                    <td className="py-2.5 px-3">
                      {getBadge(call.overall_risk, call.peak_score)}
                    </td>
                    <td className="py-2.5 px-3">
                      {call.alert_triggered ? (
                        <span className="text-[10px] px-2 py-0.5 rounded bg-rose-950 text-rose-300 border border-rose-800">
                          X-Deepfake-Alert: HIGH
                        </span>
                      ) : (
                        <span className="text-[10px] text-slate-500">
                          CLEAN / STANDBY
                        </span>
                      )}
                    </td>
                    <td className="py-2.5 px-3 text-right">
                      <div className="flex items-center justify-end gap-1.5">
                        <button
                          onClick={() => onSelectCall(call.call_id)}
                          className="p-1 rounded bg-slate-800 hover:bg-cyan-950 text-slate-300 hover:text-cyan-400 border border-slate-700 transition cursor-pointer"
                          title="Focus stream telemetry"
                        >
                          <Eye className="w-3.5 h-3.5" />
                        </button>
                        <button
                          onClick={() => onDropCall(call.call_id)}
                          className="p-1 rounded bg-rose-950/80 hover:bg-rose-900 text-rose-300 border border-rose-800 transition cursor-pointer"
                          title="Terminate call (Drop Call)"
                        >
                          <PhoneOff className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};
