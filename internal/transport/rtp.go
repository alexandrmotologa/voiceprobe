package transport

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
)

// RTPHeader represents the fixed 12-byte RTP header (RFC 3550).
type RTPHeader struct {
	Version        uint8
	Padding        bool
	Extension      bool
	CSRCCount      uint8
	Marker         bool
	PayloadType    uint8
	SequenceNumber uint16
	Timestamp      uint32
	SSRC           uint32
}

// ParseRTP parses a raw UDP datagram into an RTPHeader and payload slice.
func ParseRTP(packet []byte) (*RTPHeader, []byte, error) {
	if len(packet) < 12 {
		return nil, nil, fmt.Errorf("packet too short for RTP header: %d bytes", len(packet))
	}

	b0 := packet[0]
	version := b0 >> 6
	if version != 2 {
		return nil, nil, fmt.Errorf("unsupported RTP version: %d", version)
	}

	padding := (b0 & 0x20) != 0
	extension := (b0 & 0x10) != 0
	csrcCount := b0 & 0x0F

	b1 := packet[1]
	marker := (b1 & 0x80) != 0
	payloadType := b1 & 0x7F

	seq := binary.BigEndian.Uint16(packet[2:4])
	ts := binary.BigEndian.Uint32(packet[4:8])
	ssrc := binary.BigEndian.Uint32(packet[8:12])

	offset := 12 + int(csrcCount)*4
	if len(packet) < offset {
		return nil, nil, fmt.Errorf("packet truncated before CSRC end")
	}

	// Extension header handling
	if extension {
		if len(packet) < offset+4 {
			return nil, nil, fmt.Errorf("packet truncated before extension header")
		}
		extLen := int(binary.BigEndian.Uint16(packet[offset+2:offset+4])) * 4
		offset += 4 + extLen
		if len(packet) < offset {
			return nil, nil, fmt.Errorf("packet truncated before extension end")
		}
	}

	payload := packet[offset:]

	// Strip padding if present
	if padding && len(payload) > 0 {
		padLen := int(payload[len(payload)-1])
		if padLen <= len(payload) {
			payload = payload[:len(payload)-padLen]
		}
	}

	return &RTPHeader{
		Version:        version,
		Padding:        padding,
		Extension:      extension,
		CSRCCount:      csrcCount,
		Marker:         marker,
		PayloadType:    payloadType,
		SequenceNumber: seq,
		Timestamp:      ts,
		SSRC:           ssrc,
	}, payload, nil
}

// RTPListener listens for RTP packets over UDP and directs them to the SessionManager.
type RTPListener struct {
	port       int
	sm         *SessionManager
	conn       *net.UDPConn
	stopChan   chan struct{}
	wg         sync.WaitGroup
	ssrcToCall map[uint32]string
	mu         sync.Mutex
}

// NewRTPListener binds the UDP socket.
func NewRTPListener(port int, sm *SessionManager) (*RTPListener, error) {
	addr := &net.UDPAddr{Port: port}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind UDP RTP socket on port %d: %w", port, err)
	}

	return &RTPListener{
		port:       port,
		sm:         sm,
		conn:       conn,
		stopChan:   make(chan struct{}),
		ssrcToCall: make(map[uint32]string),
	}, nil
}

// Start initiates the packet processing loop.
func (l *RTPListener) Start() {
	l.wg.Add(1)
	go l.listenLoop()
}

func (l *RTPListener) listenLoop() {
	defer l.wg.Done()
	buf := make([]byte, 2048)

	for {
		select {
		case <-l.stopChan:
			return
		default:
		}

		n, _, err := l.conn.ReadFrom(buf)
		if err != nil {
			select {
			case <-l.stopChan:
				return
			default:
				continue
			}
		}

		hdr, payload, err := ParseRTP(buf[:n])
		if err != nil || len(payload) == 0 {
			continue
		}

		// Map SSRC to a call session
		l.mu.Lock()
		callID, exists := l.ssrcToCall[hdr.SSRC]
		if !exists {
			callID = fmt.Sprintf("rtp-ssrc-%d", hdr.SSRC)
			l.ssrcToCall[hdr.SSRC] = callID
		}
		l.mu.Unlock()

		var codecName string
		var samples []float32
		sampleRate := 8000

		switch hdr.PayloadType {
		case 0: // PCMU (G.711 mu-law)
			codecName = "G.711u"
			samples = DecodeMuLaw(payload)
		case 8: // PCMA (G.711 A-law)
			codecName = "G.711a"
			samples = DecodeALaw(payload)
		case 10: // L16 mono 44.1kHz or custom 16kHz
			codecName = "L16"
			samples = DecodePCM16LittleEndian(payload)
			sampleRate = 16000
		default:
			// Default assume PCMU for standard telephony
			codecName = fmt.Sprintf("PT-%d", hdr.PayloadType)
			samples = DecodeMuLaw(payload)
		}

		session := l.sm.GetOrCreateSession(callID, "rtp", codecName, sampleRate)
		l.sm.FeedAudio(session, samples)
	}
}

// Stop closes the UDP socket and waits for the loop to terminate.
func (l *RTPListener) Stop() {
	close(l.stopChan)
	if l.conn != nil {
		l.conn.Close()
	}
	l.wg.Wait()
	log.Printf("[RTP Listener] Stopped on port %d", l.port)
}
