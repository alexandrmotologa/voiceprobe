import React, { useEffect, useState, useRef } from 'react';
import { Header } from './components/Header';
import { ThreatGauge } from './components/ThreatGauge';
import { SpectrogramWaterfall } from './components/SpectrogramWaterfall';
import { Oscilloscope } from './components/Oscilloscope';
import { AcousticMetrics } from './components/AcousticMetrics';
import { ActiveCallsTable } from './components/ActiveCallsTable';
import { FileAnalyzerModal } from './components/FileAnalyzerModal';
import { IncidentsDrawer } from './components/IncidentsDrawer';
import { TelemetryEvent, CallSession, ServerStatus } from './types/telemetry';

export const App: React.FC = () => {
  const [status, setStatus] = useState<ServerStatus | null>(null);
  const [sseConnected, setSseConnected] = useState<boolean>(false);
  const [activeCalls, setActiveCalls] = useState<CallSession[]>([]);
  const [selectedCallId, setSelectedCallId] = useState<string>('');
  const [latestTelemetry, setLatestTelemetry] = useState<TelemetryEvent | null>(null);

  const [isUploadOpen, setIsUploadOpen] = useState<boolean>(false);
  const [isIncidentsOpen, setIsIncidentsOpen] = useState<boolean>(false);

  const selectedCallIdRef = useRef<string>('');
  selectedCallIdRef.current = selectedCallId;

  // 1. Fetch server status and active calls
  const fetchStatusAndCalls = async () => {
    try {
      const [statusRes, callsRes] = await Promise.all([
        fetch('/api/v1/status'),
        fetch('/api/v1/calls')
      ]);

      if (statusRes.ok) {
        const sData = await statusRes.json();
        setStatus(sData);
      }
      if (callsRes.ok) {
        const cData = await callsRes.json();
        const callsList: CallSession[] = cData || [];
        setActiveCalls(callsList);

        // Auto-select first call if none selected
        if (!selectedCallIdRef.current && callsList.length > 0) {
          setSelectedCallId(callsList[0].call_id);
        }
      }
    } catch (err) {
      // Offline / reconnecting
    }
  };

  useEffect(() => {
    fetchStatusAndCalls();
    const interval = setInterval(fetchStatusAndCalls, 2000);
    return () => clearInterval(interval);
  }, []);

  // 2. Connect to SSE Telemetry stream
  useEffect(() => {
    let es: EventSource | null = null;
    let reconnectTimeout: any = null;

    const connectSSE = () => {
      es = new EventSource('/api/v1/events');

      es.onopen = () => {
        setSseConnected(true);
      };

      es.addEventListener('telemetry', (e: MessageEvent) => {
        try {
          const event: TelemetryEvent = JSON.parse(e.data);
          // If no call is currently selected, select this one
          if (!selectedCallIdRef.current) {
            setSelectedCallId(event.call_id);
          }

          // If event matches selected call (or we are in single active call mode)
          if (!selectedCallIdRef.current || selectedCallIdRef.current === event.call_id) {
            setLatestTelemetry(event);
          }
        } catch (err) {
          console.error('Failed to parse SSE event', err);
        }
      });

      es.onerror = () => {
        setSseConnected(false);
        if (es) {
          es.close();
        }
        reconnectTimeout = setTimeout(connectSSE, 3000);
      };
    };

    connectSSE();

    return () => {
      if (es) es.close();
      if (reconnectTimeout) clearTimeout(reconnectTimeout);
    };
  }, []);

  // 3. Actions
  const handleSimulate = async (type: 'cloned' | 'human') => {
    try {
      const resp = await fetch('/api/v1/simulate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type, duration_sec: 8 }),
      });
      if (resp.ok) {
        const res = await resp.json();
        setSelectedCallId(res.call_id);
        fetchStatusAndCalls();
      }
    } catch (err) {
      console.error('Failed to trigger simulation', err);
    }
  };

  const handleDropCall = async (callId: string) => {
    try {
      await fetch(`/api/v1/calls/${callId}/drop`, { method: 'POST' });
      if (selectedCallId === callId) {
        setSelectedCallId('');
        setLatestTelemetry(null);
      }
      fetchStatusAndCalls();
    } catch (err) {
      console.error('Failed to drop call', err);
    }
  };

  return (
    <div className="min-h-screen bg-[#08090d] text-slate-100 flex flex-col font-sans scanline-overlay selection:bg-cyan-500 selection:text-black">
      {/* Top Header */}
      <Header
        status={status}
        sseConnected={sseConnected}
        onSimulate={handleSimulate}
        onOpenUpload={() => setIsUploadOpen(true)}
        onOpenIncidents={() => setIsIncidentsOpen(true)}
        activeCount={activeCalls.length}
      />

      {/* Main Forensic Dashboard */}
      <main className="flex-1 max-w-7xl w-full mx-auto p-4 sm:p-6 space-y-6">
        {/* Top Grid: Threat Gauge & Real-Time Waveform / Spectrogram */}
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-stretch">
          {/* Column 1: Threat Gauge & Biometrics (5 cols) */}
          <div className="lg:col-span-5 flex flex-col gap-6">
            <ThreatGauge
              score={latestTelemetry?.instant_score ?? 0}
              smoothedScore={latestTelemetry?.smoothed_score ?? 0}
              riskLevel={latestTelemetry?.risk_level ?? 'CLEAN'}
              confidence={latestTelemetry?.confidence ?? 0.95}
              callId={selectedCallId || latestTelemetry?.call_id}
              consecutiveHits={latestTelemetry?.consecutive_hits ?? 0}
            />

            <AcousticMetrics
              telemetry={latestTelemetry}
            />
          </div>

          {/* Column 2: Visualizers - Spectrogram & Oscilloscope (7 cols) */}
          <div className="lg:col-span-7 flex flex-col gap-6">
            <SpectrogramWaterfall
              melEnergies={latestTelemetry?.mel_energies}
              isVoiced={latestTelemetry?.is_voiced ?? true}
            />

            <Oscilloscope
              f0Hz={latestTelemetry?.f0_hz ?? 0}
              vadEnergy={latestTelemetry?.vad_energy ?? 0.005}
              isVoiced={latestTelemetry?.is_voiced ?? true}
              jitter={latestTelemetry?.jitter ?? 0.015}
              shimmer={latestTelemetry?.shimmer ?? 0.035}
              riskLevel={latestTelemetry?.risk_level ?? 'CLEAN'}
            />
          </div>
        </div>

        {/* Bottom Section: Active Telephony Streams Table */}
        <ActiveCallsTable
          calls={activeCalls}
          selectedCallId={selectedCallId}
          onSelectCall={(id) => setSelectedCallId(id)}
          onDropCall={handleDropCall}
          onTriggerSimulation={handleSimulate}
        />
      </main>

      {/* Modals & Drawers */}
      <FileAnalyzerModal
        isOpen={isUploadOpen}
        onClose={() => setIsUploadOpen(false)}
      />

      <IncidentsDrawer
        isOpen={isIncidentsOpen}
        onClose={() => setIsIncidentsOpen(false)}
      />

      {/* Footer bar */}
      <footer className="border-t border-slate-900 bg-[#07080c] py-3 px-6 text-center text-xs font-mono text-slate-500">
        voiceprobe &bull; Real-time deepfake & voice cloning fraud sentinel &bull; SIP / RTP / Twilio Media Streams &bull; Zero-alloc DSP Engine
      </footer>
    </div>
  );
};

export default App;
