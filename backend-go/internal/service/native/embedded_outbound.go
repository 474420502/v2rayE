package native

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"v2raye/backend-go/internal/domain"
)

func buildEmbeddedDialer(profile *domain.ProfileItem) (outboundConnDialer, error) {
	if profile == nil {
		return func(targetAddr string) (net.Conn, error) {
			return net.DialTimeout("tcp", targetAddr, 15*time.Second)
		}, nil
	}

	switch profile.Protocol {
	case domain.ProtocolShadowsocks:
		return buildShadowsocksDialer(profile)
	case domain.ProtocolTrojan:
		return buildTrojanDialer(profile)
	case domain.ProtocolVLESS:
		return buildVLESSDialer(profile)
	case domain.ProtocolVMess:
		return nil, fmt.Errorf("embedded engine does not support vmess yet")
	default:
		return nil, fmt.Errorf("embedded engine does not support protocol %s", profile.Protocol)
	}
}

func buildShadowsocksDialer(profile *domain.ProfileItem) (outboundConnDialer, error) {
	if profile.Shadowsocks == nil {
		return nil, fmt.Errorf("missing shadowsocks config")
	}

	method := strings.ToLower(strings.TrimSpace(profile.Shadowsocks.Method))
	keyLen := 0
	switch method {
	case "aes-128-gcm":
		keyLen = 16
	case "aes-256-gcm":
		keyLen = 32
	default:
		return nil, fmt.Errorf("unsupported shadowsocks method in embedded engine: %s", profile.Shadowsocks.Method)
	}

	masterKey := evpBytesToKeyMD5([]byte(profile.Shadowsocks.Password), keyLen)
	serverAddr := net.JoinHostPort(profile.Address, strconv.Itoa(profile.Port))

	return func(targetAddr string) (net.Conn, error) {
		rawConn, err := net.DialTimeout("tcp", serverAddr, 15*time.Second)
		if err != nil {
			return nil, err
		}
		ssConn, err := newShadowsocksConn(rawConn, method, masterKey)
		if err != nil {
			_ = rawConn.Close()
			return nil, err
		}
		if _, err := ssConn.Write(serializeSocksAddr(targetAddr)); err != nil {
			_ = ssConn.Close()
			return nil, err
		}
		return ssConn, nil
	}, nil
}

func buildTrojanDialer(profile *domain.ProfileItem) (outboundConnDialer, error) {
	if profile.Trojan == nil {
		return nil, fmt.Errorf("missing trojan config")
	}

	serverAddr := net.JoinHostPort(profile.Address, strconv.Itoa(profile.Port))
	serverName := profile.Address
	insecure := false
	if profile.Transport != nil {
		if profile.Transport.SNI != "" {
			serverName = profile.Transport.SNI
		}
		insecure = profile.Transport.SkipCertVerify
		if profile.Transport.Network != "" && profile.Transport.Network != "tcp" {
			return nil, fmt.Errorf("embedded trojan supports tcp transport only")
		}
	}
	passwordHash := sha256.Sum224([]byte(profile.Trojan.Password))
	auth := strings.ToLower(hex.EncodeToString(passwordHash[:]))

	return func(targetAddr string) (net.Conn, error) {
		d := &net.Dialer{Timeout: 15 * time.Second}
		conn, err := tls.DialWithDialer(d, "tcp", serverAddr, &tls.Config{
			ServerName:         serverName,
			InsecureSkipVerify: insecure,
			MinVersion:         tls.VersionTLS12,
		})
		if err != nil {
			return nil, err
		}

		req := append([]byte(auth+"\r\n"), 0x01)
		req = append(req, serializeSocksAddr(targetAddr)...)
		req = append(req, '\r', '\n')
		if _, err := conn.Write(req); err != nil {
			_ = conn.Close()
			return nil, err
		}
		return conn, nil
	}, nil
}

func buildVLESSDialer(profile *domain.ProfileItem) (outboundConnDialer, error) {
	if profile.VLESS == nil {
		return nil, fmt.Errorf("missing vless config")
	}

	serverAddr := net.JoinHostPort(profile.Address, strconv.Itoa(profile.Port))
	uuidBytes, err := parseUUIDBytes(profile.VLESS.UUID)
	if err != nil {
		return nil, fmt.Errorf("invalid vless uuid: %w", err)
	}

	serverName := profile.Address
	useTLS := false
	insecure := false
	network := "tcp"
	wsPath := "/"
	wsHostHeader := ""
	if profile.Transport != nil {
		if profile.Transport.Network != "" {
			network = strings.ToLower(strings.TrimSpace(profile.Transport.Network))
		}
		if network != "tcp" && network != "ws" {
			return nil, fmt.Errorf("embedded vless supports tcp/ws transport only")
		}
		useTLS = profile.Transport.TLS
		insecure = profile.Transport.SkipCertVerify
		if profile.Transport.SNI != "" {
			serverName = profile.Transport.SNI
		}
		if network == "ws" {
			if strings.TrimSpace(profile.Transport.WSPath) != "" {
				wsPath = strings.TrimSpace(profile.Transport.WSPath)
			}
			if !strings.HasPrefix(wsPath, "/") {
				wsPath = "/" + wsPath
			}
			if profile.Transport.WSHeaders != nil {
				wsHostHeader = strings.TrimSpace(profile.Transport.WSHeaders["Host"])
			}
		}
	}

	return func(targetAddr string) (net.Conn, error) {
		var (
			conn net.Conn
			err  error
		)
		if network == "ws" {
			conn, err = dialWebSocketStream(serverAddr, useTLS, serverName, insecure, wsPath, wsHostHeader)
		} else {
			conn, err = dialTCPOrTLS(serverAddr, useTLS, serverName, insecure)
		}
		if err != nil {
			return nil, err
		}

		head, err := buildVLESSRequestHeader(uuidBytes, targetAddr)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		if _, err := conn.Write(head); err != nil {
			_ = conn.Close()
			return nil, err
		}

		respHead := make([]byte, 2)
		if _, err := io.ReadFull(conn, respHead); err != nil {
			_ = conn.Close()
			return nil, err
		}
		if addonLen := int(respHead[1]); addonLen > 0 {
			addon := make([]byte, addonLen)
			if _, err := io.ReadFull(conn, addon); err != nil {
				_ = conn.Close()
				return nil, err
			}
		}
		return conn, nil
	}, nil
}

func dialTCPOrTLS(serverAddr string, useTLS bool, serverName string, insecure bool) (net.Conn, error) {
	d := &net.Dialer{Timeout: 15 * time.Second}
	if useTLS {
		return tls.DialWithDialer(d, "tcp", serverAddr, &tls.Config{
			ServerName:         serverName,
			InsecureSkipVerify: insecure,
			MinVersion:         tls.VersionTLS12,
		})
	}
	return d.Dial("tcp", serverAddr)
}

func parseUUIDBytes(raw string) ([]byte, error) {
	s := strings.ReplaceAll(strings.TrimSpace(raw), "-", "")
	if len(s) != 32 {
		return nil, fmt.Errorf("uuid length mismatch")
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if len(b) != 16 {
		return nil, fmt.Errorf("uuid bytes mismatch")
	}
	return b, nil
}

func buildVLESSRequestHeader(uuidBytes []byte, targetAddr string) ([]byte, error) {
	if len(uuidBytes) != 16 {
		return nil, fmt.Errorf("invalid uuid bytes")
	}
	addr := serializeSocksAddr(targetAddr)
	port := binary.BigEndian.Uint16(addr[len(addr)-2:])
	atypAndAddr := addr[:len(addr)-2]

	head := make([]byte, 0, 1+16+1+1+2+len(atypAndAddr))
	head = append(head, 0x00)
	head = append(head, uuidBytes...)
	head = append(head, 0x00)
	head = append(head, 0x01)
	head = binary.BigEndian.AppendUint16(head, port)
	head = append(head, atypAndAddr...)
	return head, nil
}

func serializeSocksAddr(targetAddr string) []byte {
	host, portStr, err := net.SplitHostPort(targetAddr)
	if err != nil {
		host = targetAddr
		portStr = "80"
	}
	port, _ := strconv.Atoi(portStr)
	if port < 0 || port > 65535 {
		port = 80
	}

	if ip := net.ParseIP(host); ip != nil {
		if ipv4 := ip.To4(); ipv4 != nil {
			buf := make([]byte, 1+4+2)
			buf[0] = 0x01
			copy(buf[1:5], ipv4)
			binary.BigEndian.PutUint16(buf[5:], uint16(port))
			return buf
		}
		ipv6 := ip.To16()
		buf := make([]byte, 1+16+2)
		buf[0] = 0x04
		copy(buf[1:17], ipv6)
		binary.BigEndian.PutUint16(buf[17:], uint16(port))
		return buf
	}

	host = strings.Trim(host, "[]")
	if len(host) > 255 {
		host = host[:255]
	}
	buf := make([]byte, 1+1+len(host)+2)
	buf[0] = 0x03
	buf[1] = byte(len(host))
	copy(buf[2:2+len(host)], []byte(host))
	binary.BigEndian.PutUint16(buf[2+len(host):], uint16(port))
	return buf
}

func evpBytesToKeyMD5(password []byte, keyLen int) []byte {
	key := make([]byte, 0, keyLen)
	prev := []byte{}
	for len(key) < keyLen {
		h := md5.New()
		_, _ = h.Write(prev)
		_, _ = h.Write(password)
		prev = h.Sum(nil)
		key = append(key, prev...)
	}
	return key[:keyLen]
}

type shadowsocksConn struct {
	net.Conn
	method    string
	masterKey []byte
	keyLen    int

	encAEAD   cipher.AEAD
	decAEAD   cipher.AEAD
	sendSalt  []byte
	sendNonce []byte
	recvNonce []byte
	sentSalt  bool
	readBuf   bytes.Buffer
}

func newShadowsocksConn(raw net.Conn, method string, masterKey []byte) (*shadowsocksConn, error) {
	keyLen := len(masterKey)
	if keyLen != 16 && keyLen != 32 {
		return nil, fmt.Errorf("invalid shadowsocks key length")
	}

	salt := make([]byte, keyLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	encAEAD, err := buildShadowsocksAEAD(method, masterKey, salt)
	if err != nil {
		return nil, err
	}

	return &shadowsocksConn{
		Conn:      raw,
		method:    method,
		masterKey: append([]byte(nil), masterKey...),
		keyLen:    keyLen,
		encAEAD:   encAEAD,
		sendSalt:  salt,
		sendNonce: make([]byte, encAEAD.NonceSize()),
	}, nil
}

func buildShadowsocksAEAD(method string, masterKey, salt []byte) (cipher.AEAD, error) {
	subkey, err := deriveShadowsocksSubkey(masterKey, salt)
	if err != nil {
		return nil, err
	}
	switch method {
	case "aes-128-gcm":
		block, err := aes.NewCipher(subkey[:16])
		if err != nil {
			return nil, err
		}
		return cipher.NewGCM(block)
	case "aes-256-gcm":
		block, err := aes.NewCipher(subkey[:32])
		if err != nil {
			return nil, err
		}
		return cipher.NewGCM(block)
	default:
		return nil, fmt.Errorf("unsupported shadowsocks method: %s", method)
	}
}

func deriveShadowsocksSubkey(masterKey, salt []byte) ([]byte, error) {
	return hkdf.Key(sha1.New, masterKey, salt, "ss-subkey", len(masterKey))
}

func (c *shadowsocksConn) initDecoderIfNeeded() error {
	if c.decAEAD != nil {
		return nil
	}
	recvSalt := make([]byte, c.keyLen)
	if _, err := io.ReadFull(c.Conn, recvSalt); err != nil {
		return err
	}
	decAEAD, err := buildShadowsocksAEAD(c.method, c.masterKey, recvSalt)
	if err != nil {
		return err
	}
	c.decAEAD = decAEAD
	c.recvNonce = make([]byte, decAEAD.NonceSize())
	return nil
}

func (c *shadowsocksConn) Read(p []byte) (int, error) {
	if c.readBuf.Len() == 0 {
		if err := c.readChunk(); err != nil {
			return 0, err
		}
	}
	return c.readBuf.Read(p)
}

func (c *shadowsocksConn) readChunk() error {
	if err := c.initDecoderIfNeeded(); err != nil {
		return err
	}

	tagSize := c.decAEAD.Overhead()
	encryptedLen := make([]byte, 2+tagSize)
	if _, err := io.ReadFull(c.Conn, encryptedLen); err != nil {
		return err
	}
	plainLen, err := c.decAEAD.Open(nil, c.recvNonce, encryptedLen, nil)
	if err != nil {
		return err
	}
	incNonce(c.recvNonce)
	payloadLen := int(binary.BigEndian.Uint16(plainLen))
	if payloadLen == 0 {
		return io.EOF
	}
	encryptedPayload := make([]byte, payloadLen+tagSize)
	if _, err := io.ReadFull(c.Conn, encryptedPayload); err != nil {
		return err
	}
	plain, err := c.decAEAD.Open(nil, c.recvNonce, encryptedPayload, nil)
	if err != nil {
		return err
	}
	incNonce(c.recvNonce)
	c.readBuf.Write(plain)
	return nil
}

func (c *shadowsocksConn) Write(p []byte) (int, error) {
	if !c.sentSalt {
		if _, err := c.Conn.Write(c.sendSalt); err != nil {
			return 0, err
		}
		c.sentSalt = true
	}

	written := 0
	for len(p) > 0 {
		n := len(p)
		if n > 0x3fff {
			n = 0x3fff
		}
		chunk := p[:n]
		p = p[n:]

		lenBuf := []byte{byte(n >> 8), byte(n)}
		encLen := c.encAEAD.Seal(nil, c.sendNonce, lenBuf, nil)
		incNonce(c.sendNonce)
		encPayload := c.encAEAD.Seal(nil, c.sendNonce, chunk, nil)
		incNonce(c.sendNonce)

		if _, err := c.Conn.Write(encLen); err != nil {
			return written, err
		}
		if _, err := c.Conn.Write(encPayload); err != nil {
			return written, err
		}
		written += n
	}
	return written, nil
}

func incNonce(nonce []byte) {
	for i := 0; i < len(nonce); i++ {
		nonce[i]++
		if nonce[i] != 0 {
			break
		}
	}
}
