import React, { useState, useRef, useEffect } from 'react';
import { Mic, MicOff } from 'lucide-react';

interface MicrophoneProbeProps {
  onStreamActive: (active: boolean) => void;
}

export const MicrophoneProbe: React.FC<MicrophoneProbeProps> = ({ onStreamActive }) => {
  const [isRecording, setIsRecording] = useState(false);
  const [rmsLevel, setRmsLevel] = useState(0);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const audioCtxRef = useRef<AudioContext | null>(null);
  const mediaStreamRef = useRef<MediaStream | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const processorRef = useRef<ScriptProcessorNode | null>(null);

  const startMicrophone = async () => {
    setErrorMsg(null);
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          channelCount: 1,
          echoCancellation: true,
          noiseSuppression: false, // We want raw acoustic dynamics for micro-tremor detection!
          autoGainControl: false,
        },
      });
      mediaStreamRef.current = stream;

      const audioCtx = new (window.AudioContext || (window as any).webkitAudioContext)();
      audioCtxRef.current = audioCtx;

      const source = audioCtx.createMediaStreamSource(stream);
      // ScriptProcessor with 2048 buffer
      const processor = audioCtx.createScriptProcessor(2048, 1, 1);
      processorRef.current = processor;

      // Connect WebSocket
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = `${protocol}//${window.location.host}/stream/browser-mic?call_id=mic-live`;
      const ws = new WebSocket(wsUrl);
      ws.binaryType = 'arraybuffer';
      wsRef.current = ws;

      const inputSampleRate = audioCtx.sampleRate;
      const targetSampleRate = 8000;
      const downsampleRatio = inputSampleRate / targetSampleRate;

      processor.onaudioprocess = (e) => {
        if (ws.readyState !== WebSocket.OPEN) return;

        const inputData = e.inputBuffer.getChannelData(0);

        // Calculate visual RMS
        let sumSq = 0;
        for (let i = 0; i < inputData.length; i++) {
          sumSq += inputData[i] * inputData[i];
        }
        const rms = Math.sqrt(sumSq / inputData.length);
        setRmsLevel(Math.min(rms * 10, 1.0));

        // Downsample to 8000 Hz
        const outputLength = Math.floor(inputData.length / downsampleRatio);
        const pcm16 = new Int16Array(outputLength);

        for (let i = 0; i < outputLength; i++) {
          const srcIdx = Math.floor(i * downsampleRatio);
          const sample = Math.max(-1.0, Math.min(1.0, inputData[srcIdx]));
          // Convert to 16-bit PCM integer
          pcm16[i] = sample < 0 ? sample * 0x8000 : sample * 0x7FFF;
        }

        ws.send(pcm16.buffer);
      };

      source.connect(processor);
      processor.connect(audioCtx.destination);

      setIsRecording(true);
      onStreamActive(true);
    } catch (err: any) {
      console.error('Microphone access failed:', err);
      setErrorMsg(err.message || 'Microphone access denied');
      setIsRecording(false);
      onStreamActive(false);
    }
  };

  const stopMicrophone = () => {
    if (processorRef.current) {
      processorRef.current.disconnect();
      processorRef.current = null;
    }
    if (mediaStreamRef.current) {
      mediaStreamRef.current.getTracks().forEach((track) => track.stop());
      mediaStreamRef.current = null;
    }
    if (audioCtxRef.current) {
      audioCtxRef.current.close();
      audioCtxRef.current = null;
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    setIsRecording(false);
    setRmsLevel(0);
    onStreamActive(false);
  };

  useEffect(() => {
    return () => {
      stopMicrophone();
    };
  }, []);

  return (
    <div className="flex items-center gap-2">
      {isRecording ? (
        <button
          onClick={stopMicrophone}
          className="flex items-center gap-2 px-3 py-1.5 rounded-md bg-rose-600 hover:bg-rose-500 text-white font-mono text-xs font-bold transition shadow-[0_0_15px_rgba(244,63,94,0.4)] cursor-pointer"
        >
          <MicOff className="w-3.5 h-3.5 animate-pulse" />
          <span>STOP MIC STREAM</span>
          {/* Audio level meter */}
          <div className="w-12 h-2 rounded bg-black/50 overflow-hidden flex items-center">
            <div
              className="h-full bg-cyan-400 transition-all duration-75"
              style={{ width: `${rmsLevel * 100}%` }}
            />
          </div>
        </button>
      ) : (
        <button
          onClick={startMicrophone}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-cyan-950/80 hover:bg-cyan-900 text-cyan-300 border border-cyan-800/80 font-mono text-xs font-semibold transition hover:shadow-[0_0_12px_rgba(0,242,254,0.3)] cursor-pointer"
          title="Stream your real microphone audio directly into the DSP engine"
        >
          <Mic className="w-3.5 h-3.5 text-cyan-400" />
          <span>Test My Voice (Live Mic)</span>
        </button>
      )}

      {errorMsg && (
        <span className="text-[11px] font-mono text-rose-400 px-2 py-0.5 rounded bg-rose-950/80 border border-rose-800">
          {errorMsg}
        </span>
      )}
    </div>
  );
};
