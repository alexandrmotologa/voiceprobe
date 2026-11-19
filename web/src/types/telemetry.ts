export type RiskLevel = 'CLEAN' | 'SUSPICIOUS' | 'CRITICAL';

export interface TelemetryEvent {
  call_id: string;
  timestamp: string;
  instant_score: number;
  smoothed_score: number;
  risk_level: RiskLevel;
  confidence: number;
  phase_discontinuity: number;
  spectral_flux: number;
  spectral_centroid: number;
  jitter: number;
  shimmer: number;
  f0_hz: number;
  spectral_tilt?: number;
  formant_dispersion?: number;
  thd_percent?: number;
  loudspeaker_anomaly?: boolean;
  vad_energy: number;
  is_voiced: boolean;
  consecutive_hits: number;
  key_indicators?: string[];
  mel_energies?: number[];
}

export interface FrameDiagnosis {
  frame_index: number;
  timestamp_offset_ms: number;
  is_voiced: boolean;
  instant_score: number;
  smoothed_score: number;
  risk_level: RiskLevel;
  jitter: number;
  shimmer: number;
  f0_hz: number;
  phase_discontinuity: number;
  spectral_flux: number;
  consecutive_hits: number;
  key_indicators?: string[];
}

export interface CallSession {
  call_id: string;
  transport: string;
  codec: string;
  start_time: string;
  last_active: string;
  sample_rate: number;
  total_samples: number;
  peak_score: number;
  overall_risk: RiskLevel;
  alert_triggered: boolean;
  last_diagnosis?: FrameDiagnosis;
}

export interface ServerStatus {
  status: string;
  service: string;
  version: string;
  uptime_seconds: number;
  active_sessions: number;
  config: {
    detection: {
      suspicious_threshold: number;
      critical_threshold: number;
      smoothing_alpha: number;
      min_voiced_frames: number;
    };
    audio: {
      sample_rate: number;
      frame_duration_ms: number;
      hop_duration_ms: number;
    };
  };
}

export interface FileAnalysisResult {
  file_name: string;
  sample_rate: number;
  duration_sec: number;
  total_frames: number;
  voiced_frames: number;
  peak_score: number;
  average_score: number;
  overall_threat_level: RiskLevel;
  summary_indicators: string[];
  frames: FrameDiagnosis[];
}

export interface IncidentRecord {
  call_id: string;
  timestamp: string;
  peak_score: number;
  overall_risk: RiskLevel;
  summary: string;
  wav_file: string;
  indicators: string[];
}

export interface SecurityPolicy {
  name: 'STANDARD' | 'HIGH_VALUE_WIRE' | 'EXECUTIVE_PROTECTION';
  description: string;
  suspicious_threshold: number;
  critical_threshold: number;
  min_voiced_hits: number;
  action: string;
  max_wire_amount_eur?: number;
}

export interface ForensicComplianceReport {
  report_id: string;
  generated_at: string;
  call_id: string;
  audio_sha256: string;
  overall_threat_level: RiskLevel;
  peak_synthetic_score: number;
  confidence: number;
  acoustic_anomalies: string[];
  regulatory_framework: string;
  recommended_action: string;
  digital_signature: string;
}
