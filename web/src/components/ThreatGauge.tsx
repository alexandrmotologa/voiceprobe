import React from 'react';
import { ShieldCheck, AlertTriangle, AlertOctagon, CheckCircle2 } from 'lucide-react';
import { RiskLevel } from '../types/telemetry';

interface ThreatGaugeProps {
  score: number; // 0.0 to 1.0
  smoothedScore: number;
  riskLevel: RiskLevel;
  confidence: number;
  callId?: string;
  consecutiveHits: number;
}

export const ThreatGauge: React.FC<ThreatGaugeProps> = ({
  score,
  smoothedScore,
  riskLevel,
  confidence,
  callId,
  consecutiveHits,
}) => {
  const percentage = Math.min(Math.max(smoothedScore * 100, 0), 100);
  const instantPercentage = Math.min(Math.max(score * 100, 0), 100);

  // Semicircle gauge calculation
  // Radius = 85, Circumference of semicircle = PI * R = ~267
  const radius = 80;
  const circumference = Math.PI * radius;
  const strokeDashoffset = circumference - (percentage / 100) * circumference;

  let colorClasses = {
    stroke: '#10b981', // emerald
    bgGlow: 'shadow-[0_0_30px_rgba(16,185,129,0.15)]',
    text: 'text-emerald-400',
    badge: 'bg-emerald-950/80 text-emerald-300 border-emerald-800/60',
    title: 'CLEAN // NATURAL HUMAN',
    recommendation: 'ALLOW FUNDS TRANSFER // LOW FRAUD RISK',
    icon: CheckCircle2,
  };

  if (riskLevel === 'CRITICAL' || percentage >= 80) {
    colorClasses = {
      stroke: '#f43f5e', // rose
      bgGlow: 'shadow-[0_0_35px_rgba(244,63,94,0.25)]',
      text: 'text-rose-400',
      badge: 'bg-rose-950/90 text-rose-300 border-rose-800 animate-pulse',
      title: 'CRITICAL // SYNTHETIC VOICE DETECTED',
      recommendation: 'INJECT SIP HEADER: REQUIRE BIOMETRIC MFA STEP-UP',
      icon: AlertOctagon,
    };
  } else if (riskLevel === 'SUSPICIOUS' || percentage >= 35) {
    colorClasses = {
      stroke: '#f59e0b', // amber
      bgGlow: 'shadow-[0_0_30px_rgba(245,158,11,0.2)]',
      text: 'text-amber-400',
      badge: 'bg-amber-950/80 text-amber-300 border-amber-800/60',
      title: 'SUSPICIOUS // ELEVATED ACOUSTIC ANOMALIES',
      recommendation: 'CONTINUE MONITORING // OPERATOR ALERT LOGGED',
      icon: AlertTriangle,
    };
  }

  const IconComponent = colorClasses.icon;

  return (
    <div className={`p-5 rounded-xl bg-[#0f111a] border border-slate-800/90 relative overflow-hidden flex flex-col items-center justify-between transition-all ${colorClasses.bgGlow}`}>
      {/* Top Header */}
      <div className="w-full flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <ShieldCheck className="w-4 h-4 text-slate-400" />
          <span className="text-xs font-mono font-semibold tracking-wider text-slate-300 uppercase">
            Synthetic Threat Gauge
          </span>
        </div>
        {callId && (
          <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-slate-900 border border-slate-800 text-slate-400">
            {callId}
          </span>
        )}
      </div>

      {/* SVG Semicircle Dial */}
      <div className="relative w-64 h-36 flex items-end justify-center my-1">
        <svg className="w-64 h-36 overflow-visible" viewBox="0 0 200 110">
          {/* Background Arc */}
          <path
            d="M 20 100 A 80 80 0 0 1 180 100"
            fill="none"
            stroke="#1c2030"
            strokeWidth="14"
            strokeLinecap="round"
          />

          {/* Tick Threshold Indicators */}
          {/* Suspicious 35% mark */}
          <line
            x1="58" y1="36" x2="65" y2="44"
            stroke="#f59e0b" strokeWidth="2" strokeDasharray="2,2"
          />
          {/* Critical 80% mark */}
          <line
            x1="135" y1="36" x2="142" y2="44"
            stroke="#f43f5e" strokeWidth="2" strokeDasharray="2,2"
          />

          {/* Dynamic Value Arc */}
          <path
            d="M 20 100 A 80 80 0 0 1 180 100"
            fill="none"
            stroke={colorClasses.stroke}
            strokeWidth="14"
            strokeDasharray={circumference}
            strokeDashoffset={strokeDashoffset}
            strokeLinecap="round"
            className="transition-all duration-300 ease-out"
          />
        </svg>

        {/* Numeric Readout Overlay */}
        <div className="absolute inset-0 flex flex-col items-center justify-end pb-1">
          <div className="flex items-baseline gap-0.5">
            <span className={`text-4xl font-extrabold font-mono tracking-tight ${colorClasses.text}`}>
              {percentage.toFixed(1)}
            </span>
            <span className="text-lg font-mono text-slate-400 font-bold">%</span>
          </div>
          <span className="text-[11px] text-slate-400 font-mono tracking-wide">
            INSTANT: {instantPercentage.toFixed(1)}% | CONF: {(confidence * 100).toFixed(0)}%
          </span>
        </div>
      </div>

      {/* Threat Status Badge */}
      <div className="w-full mt-3 flex flex-col items-center gap-2">
        <div className={`w-full py-1.5 px-3 rounded-lg border text-xs font-mono font-semibold flex items-center justify-center gap-2 ${colorClasses.badge}`}>
          <IconComponent className="w-4 h-4 flex-shrink-0" />
          <span className="tracking-wide text-center">{colorClasses.title}</span>
        </div>

        {/* Recommendation notice */}
        <div className="w-full p-2 rounded bg-slate-900/60 border border-slate-800/80 text-center">
          <div className="text-[10px] text-slate-400 font-mono uppercase tracking-wider mb-0.5">
            Telephony Action Dispatcher
          </div>
          <div className={`text-xs font-mono font-bold ${colorClasses.text}`}>
            {colorClasses.recommendation}
          </div>
        </div>

        {/* Consecutive Hits Telemetry */}
        {consecutiveHits > 0 && (
          <div className="text-[11px] font-mono text-rose-400 flex items-center gap-1.5">
            <span className="inline-block w-2 h-2 rounded-full bg-rose-500 animate-ping" />
            <span>Sustained Anomaly Frames: {consecutiveHits} / 3</span>
          </div>
        )}
      </div>
    </div>
  );
};
