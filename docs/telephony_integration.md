# Telephony Integration Guide for voiceprobe

This document provides deployment patterns for integrating voiceprobe into corporate PBX systems, cloud telephony providers, and session border controllers (SBC).

## 1. Twilio Media Streams (WebSocket)

Twilio Media Streams forks live call audio over a bidirectional WebSocket. voiceprobe accepts Twilio stream payloads without external proxies.

### TwiML Configuration
Add a `<Stream>` instruction within your inbound TwiML voice response:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Start>
        <Stream url="wss://voiceprobe.internal.bank.com:8080/stream/twilio">
            <Parameter name="accountNumber" value="ACCT-8921" />
            <Parameter name="customerTier" value="VIP_WEALTH" />
        </Stream>
    </Start>
    <Dial>
        <Number>+14155552671</Number>
    </Dial>
</Response>
```

### Event Handling
voiceprobe processes the following Twilio events:
- `connected`: Handshake established.
- `start`: Call SID, stream SID, and custom parameters captured.
- `media`: Base64-encoded G.711 mu-law payload chunks extracted, decoded to PCM16, and fed to the circular DSP buffer.
- `stop`: Session closed and forensic summary written to disk.

## 2. SIP and RTP Proxy Integration

voiceprobe can run inline between an external Session Border Controller (SBC) and an internal PBX (such as Asterisk or FreeSWITCH).

### RTP Forking (Passive Tap)
Configure the SBC or PBX to fork an RTP audio copy to voiceprobe UDP ports:

```
[ Inbound Caller ] ──── SIP / RTP ────> [ Acme Packet / Kamailio SBC ]
                                                  │             │
                                              Forked RTP     Standard
                                                  │         Call Flow
                                                  ▼             │
                                           [ voiceprobe ]       ▼
                                                  │       [ Bank Agent ]
                                           HTTP Webhook         ▲
                                                  │             │
                                                  ▼             │
                                           [ CRM Console ] ─────┘
                                           (Screen Pop Alert)
```

### SIP Header Injection (Active B2BUA Mode)
When running as an active SIP proxy, voiceprobe inspects the audio stream during the initial 2 to 5 seconds of the call. If the synthetic probability score exceeds the threshold ($\theta \ge 0.75$), voiceprobe mutates the SIP signaling:

```
SIP/2.0 200 OK
Via: SIP/2.0/UDP sbc01.bank.net:5060;branch=z9hG4bK776asdh
From: <sip:+12025550198@sbc01.bank.net>;tag=as4b9c1d
To: <sip:finance-desk@bank.net>;tag=ba3c7e12
Call-ID: c84920489-382910@sbc01.bank.net
CSeq: 102 INVITE
Contact: <sip:voiceprobe@10.0.1.50:5060>
X-Deepfake-Alert: HIGH; score=0.94; confidence=0.98; model=voiceprobe-int8
X-Deepfake-Recommendation: REQUIRE_MFA_STEPUP
Content-Type: application/sdp
```

The receiving PBX or call center software reads `X-Deepfake-Alert` and automatically disables wire transfer authorization buttons on the agent desktop.

## 3. Asterisk Dialplan Integration

In Asterisk environments, you can use the `EAGI` (Extended Asterisk Gateway Interface) or `ExternalMedia` application in Asterisk 16+:

```asterisk
[inbound-banking-desk]
exten => 7000,1,NoOp(Inbound Call to Wire Transfer Desk)
 same => n,Set(CALL_START=${EPOCH})
 same => n,Originate(Local/vp-tap@voiceprobe-bridge,app,Playback,silence/1)
 same => n,Dial(SIP/agent-terminal-44,30,M(check_deepfake_status))

[voiceprobe-bridge]
exten => vp-tap,1,RTPStreaming(127.0.0.1,5004)
```

## 4. Webhook Alerts

When synthetic voice artifacts are detected, voiceprobe delivers an immediate HTTP POST webhook to configured endpoints:

```json
{
  "event": "deepfake.detected",
  "call_id": "CA38290184209180921",
  "timestamp": "2026-11-19T15:04:05.120Z",
  "threat_level": "CRITICAL",
  "synthetic_probability": 0.942,
  "confidence": 0.981,
  "metrics": {
    "glottal_phase_discontinuity": 0.082,
    "spectral_flux_variance": 0.061,
    "pitch_jitter_local": 0.0018,
    "shimmer_local": 0.0124
  },
  "action_taken": "SIP_HEADER_INJECTED",
  "forensic_wav_path": "/var/log/voiceprobe/incidents/CA38290184209180921.wav"
}
```
