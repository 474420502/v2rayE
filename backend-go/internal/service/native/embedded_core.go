package native

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"v2raye/backend-go/internal/domain"
)

type outboundConnDialer func(targetAddr string) (net.Conn, error)

type embeddedCore struct {
	listenHost string
	httpPort   int
	socksPort  int
	onLog      func(domain.LogLine)
	profile    *domain.ProfileItem
	dialer     outboundConnDialer

	httpLn  net.Listener
	socksLn net.Listener

	wg sync.WaitGroup

	upBytes   atomic.Int64
	downBytes atomic.Int64
}

func newEmbeddedCore(listenHost string, httpPort, socksPort int, profile *domain.ProfileItem, onLog func(domain.LogLine)) *embeddedCore {
	if listenHost == "" {
		listenHost = "127.0.0.1"
	}
	return &embeddedCore{
		listenHost: listenHost,
		httpPort:   httpPort,
		socksPort:  socksPort,
		profile:    profile,
		onLog:      onLog,
	}
}

func (c *embeddedCore) Start() error {
	dialer, err := buildEmbeddedDialer(c.profile)
	if err != nil {
		return err
	}
	c.dialer = dialer

	httpAddr := net.JoinHostPort(c.listenHost, fmt.Sprintf("%d", c.httpPort))
	httpLn, err := net.Listen("tcp", httpAddr)
	if err != nil {
		return fmt.Errorf("listen http proxy %s: %w", httpAddr, err)
	}

	socksAddr := net.JoinHostPort(c.listenHost, fmt.Sprintf("%d", c.socksPort))
	socksLn, err := net.Listen("tcp", socksAddr)
	if err != nil {
		_ = httpLn.Close()
		return fmt.Errorf("listen socks proxy %s: %w", socksAddr, err)
	}

	c.httpLn = httpLn
	c.socksLn = socksLn
	c.logf("[embedded] started http=%s socks=%s", httpAddr, socksAddr)

	c.wg.Add(2)
	go c.acceptHTTP()
	go c.acceptSOCKS()
	return nil
}

func (c *embeddedCore) Stop() {
	if c.httpLn != nil {
		_ = c.httpLn.Close()
	}
	if c.socksLn != nil {
		_ = c.socksLn.Close()
	}
	c.wg.Wait()
	c.logf("[embedded] stopped")
}

func (c *embeddedCore) SnapshotStats() domain.StatsResult {
	return domain.StatsResult{
		UpBytes:   c.upBytes.Load(),
		DownBytes: c.downBytes.Load(),
	}
}

func (c *embeddedCore) acceptHTTP() {
	defer c.wg.Done()
	for {
		conn, err := c.httpLn.Accept()
		if err != nil {
			if isListenerClosedErr(err) {
				return
			}
			c.logf("[embedded] http accept error: %v", err)
			continue
		}
		go c.handleHTTPConn(conn)
	}
}

func (c *embeddedCore) acceptSOCKS() {
	defer c.wg.Done()
	for {
		conn, err := c.socksLn.Accept()
		if err != nil {
			if isListenerClosedErr(err) {
				return
			}
			c.logf("[embedded] socks accept error: %v", err)
			continue
		}
		go c.handleSOCKSConn(conn)
	}
}

func (c *embeddedCore) handleHTTPConn(conn net.Conn) {
	defer conn.Close()
	br := bufio.NewReader(conn)

	req, err := http.ReadRequest(br)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			c.logf("[embedded] http parse request failed: %v", err)
		}
		return
	}
	defer req.Body.Close()

	if strings.EqualFold(req.Method, http.MethodConnect) {
		targetAddr := req.Host
		if !strings.Contains(targetAddr, ":") {
			targetAddr += ":443"
		}
		target, err := c.dialTarget(targetAddr)
		if err != nil {
			_, _ = io.WriteString(conn, "HTTP/1.1 502 Bad Gateway\r\n\r\n")
			c.logf("[embedded] connect dial %s failed: %v", targetAddr, err)
			return
		}
		defer target.Close()

		_, _ = io.WriteString(conn, "HTTP/1.1 200 Connection Established\r\n\r\n")
		if br.Buffered() > 0 {
			buf := make([]byte, br.Buffered())
			n, _ := io.ReadFull(br, buf)
			if n > 0 {
				wn, _ := target.Write(buf[:n])
				c.upBytes.Add(int64(wn))
			}
		}
		c.relay(conn, target)
		return
	}

	reqURL := req.URL
	if reqURL == nil {
		_, _ = io.WriteString(conn, "HTTP/1.1 400 Bad Request\r\n\r\n")
		return
	}
	if reqURL.Scheme == "" {
		reqURL = &url.URL{Scheme: "http", Host: req.Host, Path: req.URL.Path, RawQuery: req.URL.RawQuery}
	}
	targetAddr := reqURL.Host
	if !strings.Contains(targetAddr, ":") {
		targetAddr += ":80"
	}
	target, err := c.dialTarget(targetAddr)
	if err != nil {
		_, _ = io.WriteString(conn, "HTTP/1.1 502 Bad Gateway\r\n\r\n")
		c.logf("[embedded] http dial %s failed: %v", targetAddr, err)
		return
	}
	defer target.Close()

	outReq := req.Clone(req.Context())
	outReq.URL = reqURL
	outReq.RequestURI = ""
	outReq.Header.Del("Proxy-Connection")
	if outReq.Host == "" {
		outReq.Host = reqURL.Host
	}
	if err := outReq.Write(target); err != nil {
		_, _ = io.WriteString(conn, "HTTP/1.1 502 Bad Gateway\r\n\r\n")
		c.logf("[embedded] write outbound request failed: %v", err)
		return
	}

	resp, err := http.ReadResponse(bufio.NewReader(target), outReq)
	if err != nil {
		_, _ = io.WriteString(conn, "HTTP/1.1 502 Bad Gateway\r\n\r\n")
		c.logf("[embedded] read outbound response failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if err := resp.Write(conn); err != nil {
		c.logf("[embedded] write response failed: %v", err)
	}
}

func (c *embeddedCore) handleSOCKSConn(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	head := make([]byte, 2)
	if _, err := io.ReadFull(conn, head); err != nil {
		return
	}
	if head[0] != 0x05 {
		return
	}

	nMethods := int(head[1])
	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return
	}

	_, _ = conn.Write([]byte{0x05, 0x00})

	reqHead := make([]byte, 4)
	if _, err := io.ReadFull(conn, reqHead); err != nil {
		return
	}
	if reqHead[0] != 0x05 {
		return
	}
	if reqHead[1] != 0x01 {
		_, _ = conn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	targetAddr, err := readSOCKSAddr(conn, reqHead[3])
	if err != nil {
		_, _ = conn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	target, err := c.dialTarget(targetAddr)
	if err != nil {
		_, _ = conn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		c.logf("[embedded] socks dial %s failed: %v", targetAddr, err)
		return
	}
	defer target.Close()

	_, _ = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	_ = conn.SetDeadline(time.Time{})
	c.relay(conn, target)
}

func (c *embeddedCore) dialTarget(targetAddr string) (net.Conn, error) {
	if c.dialer == nil {
		return net.DialTimeout("tcp", targetAddr, 15*time.Second)
	}
	return c.dialer(targetAddr)
}

func readSOCKSAddr(r io.Reader, atyp byte) (string, error) {
	switch atyp {
	case 0x01:
		buf := make([]byte, 6)
		if _, err := io.ReadFull(r, buf); err != nil {
			return "", err
		}
		host := net.IP(buf[:4]).String()
		port := int(buf[4])<<8 | int(buf[5])
		return net.JoinHostPort(host, fmt.Sprintf("%d", port)), nil
	case 0x03:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(r, lenBuf); err != nil {
			return "", err
		}
		hostBuf := make([]byte, int(lenBuf[0])+2)
		if _, err := io.ReadFull(r, hostBuf); err != nil {
			return "", err
		}
		host := string(hostBuf[:len(hostBuf)-2])
		port := int(hostBuf[len(hostBuf)-2])<<8 | int(hostBuf[len(hostBuf)-1])
		return net.JoinHostPort(host, fmt.Sprintf("%d", port)), nil
	case 0x04:
		buf := make([]byte, 18)
		if _, err := io.ReadFull(r, buf); err != nil {
			return "", err
		}
		host := net.IP(buf[:16]).String()
		port := int(buf[16])<<8 | int(buf[17])
		return net.JoinHostPort(host, fmt.Sprintf("%d", port)), nil
	default:
		return "", fmt.Errorf("unsupported atyp %d", atyp)
	}
}

func (c *embeddedCore) relay(client, target net.Conn) {
	errCh := make(chan error, 2)
	go func() {
		n, err := io.Copy(target, client)
		if n > 0 {
			c.upBytes.Add(n)
		}
		errCh <- err
	}()
	go func() {
		n, err := io.Copy(client, target)
		if n > 0 {
			c.downBytes.Add(n)
		}
		errCh <- err
	}()

	<-errCh
	_ = client.SetDeadline(time.Now())
	_ = target.SetDeadline(time.Now())
	<-errCh
}

func (c *embeddedCore) logf(format string, args ...interface{}) {
	if c.onLog == nil {
		return
	}
	c.onLog(domain.LogLine{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     "info",
		Message:   fmt.Sprintf(format, args...),
	})
}

func isListenerClosedErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "use of closed network connection")
}
