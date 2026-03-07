package native

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

type webSocketStreamConn struct {
	net.Conn
	readBuf bytes.Buffer
}

func dialWebSocketStream(serverAddr string, useTLS bool, serverName string, insecure bool, path string, hostHeader string) (net.Conn, error) {
	if strings.TrimSpace(path) == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	d := &net.Dialer{Timeout: 15 * time.Second}
	var conn net.Conn
	var err error
	if useTLS {
		conn, err = tls.DialWithDialer(d, "tcp", serverAddr, &tls.Config{
			ServerName:         serverName,
			InsecureSkipVerify: insecure,
			MinVersion:         tls.VersionTLS12,
		})
	} else {
		conn, err = d.Dial("tcp", serverAddr)
	}
	if err != nil {
		return nil, err
	}

	keyBytes := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, keyBytes); err != nil {
		_ = conn.Close()
		return nil, err
	}
	secKey := base64.StdEncoding.EncodeToString(keyBytes)
	if hostHeader == "" {
		hostHeader = serverAddr
	}

	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n", path, hostHeader, secKey)
	if _, err := conn.Write([]byte(req)); err != nil {
		_ = conn.Close()
		return nil, err
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		_ = conn.Close()
		return nil, fmt.Errorf("websocket upgrade failed: %s", resp.Status)
	}
	if !strings.EqualFold(strings.TrimSpace(resp.Header.Get("Upgrade")), "websocket") {
		_ = conn.Close()
		return nil, fmt.Errorf("websocket upgrade failed: invalid Upgrade header")
	}
	expectAccept := wsAcceptFromKey(secKey)
	if strings.TrimSpace(resp.Header.Get("Sec-WebSocket-Accept")) != expectAccept {
		_ = conn.Close()
		return nil, fmt.Errorf("websocket upgrade failed: invalid Sec-WebSocket-Accept")
	}

	wsConn := &webSocketStreamConn{Conn: conn}
	if br.Buffered() > 0 {
		buf := make([]byte, br.Buffered())
		n, _ := io.ReadFull(br, buf)
		if n > 0 {
			wsConn.readBuf.Write(buf[:n])
		}
	}
	return wsConn, nil
}

func wsAcceptFromKey(key string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, key)
	_, _ = io.WriteString(h, wsGUID)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (c *webSocketStreamConn) Read(p []byte) (int, error) {
	if c.readBuf.Len() == 0 {
		if err := c.readFrame(); err != nil {
			return 0, err
		}
	}
	return c.readBuf.Read(p)
}

func (c *webSocketStreamConn) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	const maxFramePayload = 16 * 1024
	written := 0
	for len(p) > 0 {
		n := len(p)
		if n > maxFramePayload {
			n = maxFramePayload
		}
		chunk := p[:n]
		p = p[n:]
		frame, err := buildClientBinaryFrame(chunk)
		if err != nil {
			return written, err
		}
		if _, err := c.Conn.Write(frame); err != nil {
			return written, err
		}
		written += n
	}
	return written, nil
}

func (c *webSocketStreamConn) readFrame() error {
	head := make([]byte, 2)
	if _, err := io.ReadFull(c.Conn, head); err != nil {
		return err
	}
	fin := head[0]&0x80 != 0
	opcode := head[0] & 0x0f
	masked := head[1]&0x80 != 0
	payloadLen := int(head[1] & 0x7f)

	switch payloadLen {
	case 126:
		ext := make([]byte, 2)
		if _, err := io.ReadFull(c.Conn, ext); err != nil {
			return err
		}
		payloadLen = int(uint16(ext[0])<<8 | uint16(ext[1]))
	case 127:
		ext := make([]byte, 8)
		if _, err := io.ReadFull(c.Conn, ext); err != nil {
			return err
		}
		for _, b := range ext {
			payloadLen = (payloadLen << 8) | int(b)
		}
	}

	var maskKey []byte
	if masked {
		maskKey = make([]byte, 4)
		if _, err := io.ReadFull(c.Conn, maskKey); err != nil {
			return err
		}
	}

	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(c.Conn, payload); err != nil {
		return err
	}
	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}

	switch opcode {
	case 0x0, 0x2:
		c.readBuf.Write(payload)
	case 0x8:
		return io.EOF
	case 0x9:
		pong, err := buildControlFrame(0xA, payload)
		if err == nil {
			_, _ = c.Conn.Write(pong)
		}
		if !fin {
			return c.readFrame()
		}
		return c.readFrame()
	case 0xA:
		return c.readFrame()
	default:
		return fmt.Errorf("unsupported websocket opcode %d", opcode)
	}
	return nil
}

func buildClientBinaryFrame(payload []byte) ([]byte, error) {
	return buildClientFrame(0x2, payload)
}

func buildControlFrame(opcode byte, payload []byte) ([]byte, error) {
	return buildClientFrame(opcode, payload)
}

func buildClientFrame(opcode byte, payload []byte) ([]byte, error) {
	if len(payload) > 125 && (opcode == 0x8 || opcode == 0x9 || opcode == 0xA) {
		return nil, fmt.Errorf("control frame payload too large")
	}

	maskKey := make([]byte, 4)
	if _, err := io.ReadFull(rand.Reader, maskKey); err != nil {
		return nil, err
	}

	frame := make([]byte, 0, 14+len(payload))
	frame = append(frame, 0x80|(opcode&0x0f))
	if len(payload) < 126 {
		frame = append(frame, 0x80|byte(len(payload)))
	} else if len(payload) <= 65535 {
		frame = append(frame, 0x80|126)
		frame = append(frame, byte(len(payload)>>8), byte(len(payload)))
	} else {
		frame = append(frame, 0x80|127)
		l := uint64(len(payload))
		for i := 7; i >= 0; i-- {
			frame = append(frame, byte(l>>(uint(i)*8)))
		}
	}
	frame = append(frame, maskKey...)
	for i, b := range payload {
		frame = append(frame, b^maskKey[i%4])
	}
	return frame, nil
}
