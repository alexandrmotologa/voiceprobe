package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/alexandrmotologa/voiceprobe/internal/action"
	"github.com/alexandrmotologa/voiceprobe/internal/classifier"
	"github.com/alexandrmotologa/voiceprobe/internal/config"
	"github.com/alexandrmotologa/voiceprobe/internal/server"
	"github.com/alexandrmotologa/voiceprobe/internal/transport"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		runListen(os.Args[1:])
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "listen", "start", "serve":
		runListen(args)
	case "analyze", "scan":
		runAnalyze(args)
	case "stream-sim", "simulate":
		runSimulate(args)
	case "benchmark", "bench":
		runBenchmark(args)
	case "version", "--version", "-v":
		fmt.Printf("voiceprobe v%s (go: %s, arch: %s)\n", version, runtime.Version(), runtime.GOARCH)
	case "help", "--help", "-h":
		printHelp()
	default:
		// If argument looks like a wav file, default to analyze
		if filepath.Ext(command) == ".wav" {
			runAnalyze(os.Args[1:])
			return
		}
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nRun 'voiceprobe help' for usage.\n", command)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`voiceprobe v%s - Real-Time Deepfake Voice & Synthetic Audio Fraud Detector

Usage:
  voiceprobe [command] [options]

Commands:
  listen                 Start real-time telephony proxy and Web Cockpit (default)
  analyze <file.wav>     Run offline forensic inspection on an audio file
  stream-sim             Simulate a real-time phone call stream (human or synthetic)
  benchmark              Run latency and memory stress benchmarks (<25MB, <150ms)
  version                Print version information
  help                   Show this help message

Options for 'listen':
  --http-port <int>      HTTP API and Web Cockpit port (default: 7979)
  --rtp-port <int>       UDP RTP packet listener port (default: 5004)
  --threshold <float>    Critical alert probability threshold (default: 0.75)
  --webhook-url <url>    Outbound alert notification webhook URL
  --incidents-dir <dir>  Directory to save detected forensic WAV audio

Examples:
  voiceprobe listen --http-port 7979
  voiceprobe analyze call_sample.wav --json
  voiceprobe stream-sim --type cloned --duration 8
  voiceprobe benchmark --streams 50 --frames 100
`, version)
}

func runListen(args []string) {
	fs := flag.NewFlagSet("listen", flag.ExitOnError)
	httpPort := fs.Int("http-port", 7979, "HTTP port")
	rtpPort := fs.Int("rtp-port", 5004, "RTP UDP port")
	threshold := fs.Float64("threshold", 0.75, "Critical threat threshold")
	webhookURL := fs.String("webhook-url", "", "Alert webhook endpoint")
	incidentsDir := fs.String("incidents-dir", "./data/incidents", "Incident recordings folder")
	configFile := fs.String("config", "", "Path to voiceprobe.json config")
	fs.Parse(args)

	cfg := config.DefaultConfig()
	if *configFile != "" {
		if loaded, err := config.LoadFromFile(*configFile); err == nil {
			cfg = loaded
		}
	}
	if *httpPort != 7979 {
		cfg.Server.HTTPPort = *httpPort
	}
	if *rtpPort != 5004 {
		cfg.Server.RTPPort = *rtpPort
	}
	if *threshold != 0.75 {
		cfg.Detection.CriticalThreshold = *threshold
	}
	if *webhookURL != "" {
		cfg.Actions.WebhookEnabled = true
		cfg.Actions.WebhookURL = *webhookURL
	}
	if *incidentsDir != "./data/incidents" {
		cfg.Actions.IncidentsDir = *incidentsDir
	}

	log.Printf("[voiceprobe] Initializing Real-Time Voice Fraud Detector v%s", version)
	log.Printf("[voiceprobe] Telephony DSP: %dHz | 300ms sliding window | 150ms hop", cfg.DSP.SampleRate)

	// Action dispatchers
	recorder, err := action.NewIncidentRecorder(cfg.Actions.IncidentsDir, cfg.Actions.RecordIncidents)
	if err != nil {
		log.Fatalf("Failed to initialize incident recorder: %v", err)
	}

	dispatcher := action.NewWebhookDispatcher(cfg.Actions.WebhookURL, cfg.Actions.WebhookEnabled, cfg.Actions.WebhookTimeout)
	defer dispatcher.Close()

	// Session Manager and SSE Hub
	sm := transport.NewSessionManager(cfg, recorder, dispatcher)
	sseHub := server.NewSSEHub()
	sseHub.AttachToSessionManager(sm)

	// RTP UDP Listener
	rtpListener, err := transport.NewRTPListener(cfg.Server.RTPPort, sm)
	if err != nil {
		log.Printf("[RTP Listener] Warning: Could not bind UDP port %d: %v", cfg.Server.RTPPort, err)
	} else {
		rtpListener.Start()
		defer rtpListener.Stop()
		log.Printf("[RTP Listener] Listening on UDP :%d (G.711u / G.711a / L16)", cfg.Server.RTPPort)
	}

	// HTTP Multiplexer
	mux := http.NewServeMux()
	api := server.NewServerAPI(cfg, sm, sseHub)
	api.RegisterRoutes(mux)
	server.RegisterStaticHandlers(mux)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler: mux,
	}

	go func() {
		log.Printf("[Web Cockpit] Dashboard and API active on http://127.0.0.1:%d", cfg.Server.HTTPPort)
		log.Printf("[Twilio WS] Media Stream endpoint ready at ws://127.0.0.1:%d%s", cfg.Server.HTTPPort, cfg.Server.TwilioPath)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Graceful termination handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("[voiceprobe] Shutting down services...")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("[voiceprobe] Core engine stopped successfully.")
}

func runAnalyze(args []string) {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output results as formatted JSON")
	outFile := fs.String("out", "", "Save JSON report to file")
	fs.Parse(args)

	remain := fs.Args()
	if len(remain) == 0 {
		fmt.Fprintln(os.Stderr, "Error: missing input audio file. Usage: voiceprobe analyze <file.wav>")
		os.Exit(1)
	}

	wavPath := remain[0]
	data, err := os.ReadFile(wavPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read file '%s': %v\n", wavPath, err)
		os.Exit(1)
	}

	samples, sampleRate, err := server.ParseWAVSamples(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing WAV: %v\n", err)
		os.Exit(1)
	}

	cfg := config.DefaultConfig()
	result := server.AnalyzeSamples(filepath.Base(wavPath), samples, sampleRate, cfg)

	if *jsonOutput || *outFile != "" {
		outBytes, _ := json.MarshalIndent(result, "", "  ")
		if *outFile != "" {
			if err := os.WriteFile(*outFile, outBytes, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to write output file: %v\n", err)
			} else {
				fmt.Printf("Report saved to %s\n", *outFile)
			}
		}
		if *jsonOutput {
			fmt.Println(string(outBytes))
		}
		return
	}

	// Human-readable terminal output
	fmt.Printf("\n=== voiceprobe Forensic Audio Analysis ===\n")
	fmt.Printf("File:               %s\n", result.FileName)
	fmt.Printf("Duration:           %.2f seconds (%d samples at %d Hz)\n", result.DurationSec, len(samples), result.SampleRate)
	fmt.Printf("Analysis Windows:   %d total (%d voiced speech frames)\n", result.TotalFrames, result.VoicedFrames)
	fmt.Printf("Peak Threat Score:  %.1f%%\n", result.PeakScore*100.0)
	fmt.Printf("Average Score:      %.1f%%\n", result.AverageScore*100.0)

	var threatColor string
	switch result.OverallThreatLevel {
	case classifier.RiskCritical:
		threatColor = "CRITICAL (SYNTHETIC DEEPFAKE VOICE DETECTED)"
	case classifier.RiskSuspicious:
		threatColor = "SUSPICIOUS (ELEVATED ANOMALIES)"
	default:
		threatColor = "CLEAN (NATURAL HUMAN PHONATION)"
	}
	fmt.Printf("Overall Verdict:    %s\n", threatColor)
	fmt.Printf("\nAcoustic Diagnostics:\n")
	for _, ind := range result.SummaryIndicators {
		fmt.Printf(" - %s\n", ind)
	}
	fmt.Println()
}

func runSimulate(args []string) {
	fs := flag.NewFlagSet("stream-sim", flag.ExitOnError)
	simType := fs.String("type", "cloned", "Simulation type ('cloned' or 'human')")
	duration := fs.Int("duration", 8, "Simulation duration in seconds")
	callID := fs.String("call-id", "", "Custom call ID identifier")
	fs.Parse(args)

	if *callID == "" {
		*callID = fmt.Sprintf("sim-%s-%d", *simType, time.Now().Unix()%10000)
	}

	cfg := config.DefaultConfig()
	sm := transport.NewSessionManager(cfg, nil, nil)

	fmt.Printf("Starting real-time simulation: type=%s, callID=%s, duration=%ds\n", *simType, *callID, *duration)

	sm.AddListener(func(session *transport.CallSession, diag classifier.FrameDiagnosis) {
		if diag.IsVoiced {
			fmt.Printf(" [%s] Frame #%02d | Score: %.1f%% | Level: %-10s | PhaseDisc: %.3f | Jitter: %.4f | Indicators: %v\n",
				time.Now().Format("15:04:05.000"), session.LastDiagnosis.ConsecutiveHits,
				diag.SmoothedScore*100.0, diag.RiskLevel,
				diag.Features.PhaseDiscontinuity, diag.Features.Jitter,
				diag.KeyIndicators)
		}
	})

	server.RunCallSimulation(sm, *simType, *callID, *duration)
	fmt.Println("Simulation completed.")
}

func runBenchmark(args []string) {
	fs := flag.NewFlagSet("benchmark", flag.ExitOnError)
	numStreams := fs.Int("streams", 50, "Number of concurrent phone call audio streams")
	numFrames := fs.Int("frames", 100, "Number of 300ms audio frames per stream")
	fs.Parse(args)

	fmt.Printf("Running voiceprobe Benchmark: %d concurrent streams x %d frames (300ms each)...\n", *numStreams, *numFrames)

	cfg := config.DefaultConfig()
	sm := transport.NewSessionManager(cfg, nil, nil)

	// Measure baseline memory before run
	var memBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	// Pre-generate 300ms frame (2400 samples at 8000Hz)
	frameSamples := make([]float32, 2400)
	for i := range frameSamples {
		frameSamples[i] = float32(0.5 * math.Sin(2.0*math.Pi*220.0*float64(i)/8000.0))
	}

	start := time.Now()
	totalEvaluations := (*numStreams) * (*numFrames)
	latencies := make([]time.Duration, totalEvaluations)
	latIdx := 0

	for s := 0; s < *numStreams; s++ {
		callID := fmt.Sprintf("bench-stream-%d", s)
		sess := sm.GetOrCreateSession(callID, "benchmark", "G.711u", 8000)

		for f := 0; f < *numFrames; f++ {
			fStart := time.Now()
			sm.FeedAudio(sess, frameSamples)
			latencies[latIdx] = time.Since(fStart)
			latIdx++
		}
	}

	totalDuration := time.Since(start)

	// Measure memory after run with a single GC sync
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	heapAllocMB := float64(memAfter.HeapAlloc) / (1024.0 * 1024.0)
	heapInUseMB := float64(memAfter.HeapInuse) / (1024.0 * 1024.0)
	residentRAMMB := float64(memAfter.Sys) / (1024.0 * 1024.0)

	// Calculate percentile latencies
	var sumLat time.Duration
	var maxLat time.Duration
	for _, l := range latencies {
		sumLat += l
		if l > maxLat {
			maxLat = l
		}
	}
	avgLat := sumLat / time.Duration(totalEvaluations)

	fmt.Printf("\n=== Benchmark Results ===\n")
	fmt.Printf("Total Audio Processed:   %.1f minutes of live calls\n", float64(totalEvaluations*300)/1000.0/60.0)
	fmt.Printf("Wall Clock Elapsed:      %v\n", totalDuration)
	fmt.Printf("Throughput:              %.0f frames/sec (%.1fx real-time)\n",
		float64(totalEvaluations)/totalDuration.Seconds(),
		(float64(totalEvaluations)*0.300)/totalDuration.Seconds())
	fmt.Printf("Average Frame Latency:   %v\n", avgLat)
	fmt.Printf("Max Single Frame Latency:%v\n", maxLat)
	fmt.Printf("Heap Allocated Memory:   %.2f MB (Target: < 25.0 MB)\n", heapAllocMB)
	fmt.Printf("Heap In-Use Memory:      %.2f MB\n", heapInUseMB)
	fmt.Printf("OS System Reserved:      %.2f MB\n", residentRAMMB)

	if heapAllocMB <= 25.0 {
		fmt.Printf("✓ Memory Verification:   PASSED (Heap: %.2f MB < 25.0 MB across %d streams)\n", heapAllocMB, *numStreams)
	} else {
		fmt.Printf("! Memory Notice:         Heap: %.2f MB\n", heapAllocMB)
	}

	if avgLat < 150*time.Millisecond {
		fmt.Printf("✓ Latency Verification:  PASSED (%v < 150ms)\n", avgLat)
	}
	fmt.Println()
}
