package traffic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/pion/stun"
)

const reconnectDelay = 3 * time.Second
const stunDefaultServer = "stun.l.google.com:19302"

type connectionsResponse struct {
	DownloadTotal int64 `json:"downloadTotal"`
	UploadTotal   int64 `json:"uploadTotal"`
}

// Monitor polls the Clash API /connections endpoint every second,
// computes download/upload speeds, resolves external IP via STUN
// (through the VPN tunnel) and measures TCP ping delay to that IP.
type Monitor struct {
	mu      sync.Mutex
	cfg     ClashConfig
	client  *http.Client
	ticker  *time.Ticker
	stopCh  chan struct{}
	statsCh chan TrafficStats
	running bool

	prevDownload int64
	prevUpload   int64
	prevTime     time.Time
	seeded       bool

	stunServer  string
	serverIP    string // VPN server IP from STUN
	localIP     string // non-VPN interface IP for binding
	lastDelayMs int64
	pingInfo    string // diagnostic text for the UI
}

func NewMonitor(cfg ClashConfig) *Monitor {
	m := &Monitor{
		cfg:         cfg,
		client:      &http.Client{Timeout: 4 * time.Second},
		statsCh:     make(chan TrafficStats, 8),
		stopCh:      make(chan struct{}),
		lastDelayMs: -1,
		stunServer:  stunDefaultServer,
	}

	// Try cfg.LocalAddr, then auto-detect
	if cfg.LocalAddr != "" {
		if ip := net.ParseIP(cfg.LocalAddr); ip != nil {
			m.localIP = cfg.LocalAddr
			m.pingInfo = "Bind: " + cfg.LocalAddr
		}
	}
	if m.localIP == "" {
		if ip := resolveNonVPNIP(); ip != "" {
			m.localIP = ip
			m.pingInfo = "Bind: " + ip
		} else {
			m.pingInfo = "No bind IP (direct route)"
		}
	}

	return m
}

func resolveNonVPNIP() string {
	names := []string{"Ethernet", "eth0", "en0", "enp0s3"}
	for _, name := range names {
		iface, err := net.InterfaceByName(name)
		if err != nil {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok || ipnet.IP.IsLoopback() {
				continue
			}
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return ""
}

func (m *Monitor) Stats() <-chan TrafficStats {
	return m.statsCh
}

func (m *Monitor) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.stopCh = make(chan struct{})
	m.ticker = time.NewTicker(1 * time.Second)
	m.mu.Unlock()
	go m.pollLoop()
}

func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running {
		return
	}
	m.running = false
	if m.ticker != nil {
		m.ticker.Stop()
	}
	close(m.stopCh)
}

// --- internal ---

func (m *Monitor) pollLoop() {
	defer func() {
		m.mu.Lock()
		close(m.statsCh)
		m.mu.Unlock()
	}()

	m.resolveServerIP()
	m.measurePing()

	tickCount := 0
	for {
		select {
		case <-m.ticker.C:
			tickCount++
			m.measurePing()

			if tickCount%15 == 0 {
				m.resolveServerIP()
			}
			m.sample()

		case <-m.stopCh:
			return
		}
	}
}

// resolveServerIP does STUN through VPN tunnel to get the server's external IP.
func (m *Monitor) resolveServerIP() {
	m.mu.Lock()
	oldIP := m.serverIP
	m.mu.Unlock()

	conn, err := net.Dial("udp", m.stunServer)
	if err != nil {
		m.mu.Lock()
		m.pingInfo = "STUN dial err: " + err.Error()
		m.mu.Unlock()
		return
	}
	defer conn.Close()

	c, err := stun.NewClient(conn)
	if err != nil {
		m.mu.Lock()
		m.pingInfo = "STUN client err: " + err.Error()
		m.mu.Unlock()
		return
	}
	defer c.Close()

	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	var xorAddr stun.XORMappedAddress
	var errResult error

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		err = c.Do(message, func(res stun.Event) {
			if res.Error != nil {
				errResult = res.Error
				return
			}
			if err := xorAddr.GetFrom(res.Message); err != nil {
				errResult = err
				return
			}
		})
		if err != nil {
			errResult = err
		}
		close(done)
	}()

	select {
	case <-done:
		if errResult != nil {
			m.mu.Lock()
			m.pingInfo = "STUN err: " + errResult.Error()
			m.mu.Unlock()
			return
		}
		ip := xorAddr.IP.String()
		if ip == "" {
			m.mu.Lock()
			m.pingInfo = "STUN: empty IP response"
			m.mu.Unlock()
			return
		}
		m.mu.Lock()
		m.serverIP = ip
		if oldIP != ip {
			m.pingInfo = fmt.Sprintf("STUN OK: %s (new)", ip)
		} else {
			m.pingInfo = fmt.Sprintf("STUN OK: %s", ip)
		}
		m.mu.Unlock()

	case <-ctx.Done():
		m.mu.Lock()
		m.pingInfo = "STUN: timeout (5s)"
		m.mu.Unlock()
	}
}

// measurePing does TCP connect. Measures RTT from ANY response (SYN-ACK or RST).
func (m *Monitor) measurePing() {
	m.mu.Lock()
	ip := m.serverIP
	bindIP := m.localIP
	m.mu.Unlock()

	if ip == "" {
		m.mu.Lock()
		m.pingInfo = "No server IP (STUN pending)"
		m.mu.Unlock()
		return
	}

	d, info := tcpingDiagnostic(ip, bindIP, 2*time.Second)

	m.mu.Lock()
	m.lastDelayMs = d
	m.pingInfo = info
	m.mu.Unlock()
}

// MeasureServerDelay returns RTT in ms, -1 on failure.
func (m *Monitor) MeasureServerDelay() int64 {
	m.mu.Lock()
	ip := m.serverIP
	bindIP := m.localIP
	m.mu.Unlock()
	if ip == "" {
		return -1
	}
	d, _ := tcpingDiagnostic(ip, bindIP, 2*time.Second)
	return d
}

func (m *Monitor) sample() {
	m.mu.Lock()
	serverIP := m.serverIP
	lastDelay := m.lastDelayMs
	localAddr := m.localIP
	diag := m.pingInfo
	m.mu.Unlock()

	curDl, curUl, err := m.fetchTotals()
	if err != nil {
		return
	}

	m.mu.Lock()

	if !m.seeded {
		m.seeded = true
		m.prevDownload = curDl
		m.prevUpload = curUl
		m.prevTime = time.Now()
		m.mu.Unlock()
		return
	}

	if curDl < m.prevDownload || curUl < m.prevUpload {
		m.prevDownload = curDl
		m.prevUpload = curUl
		m.prevTime = time.Now()
		m.seeded = false
		m.mu.Unlock()
		return
	}

	now := time.Now()
	elapsed := now.Sub(m.prevTime).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}

	dlBps := float64(curDl-m.prevDownload) / elapsed
	ulBps := float64(curUl-m.prevUpload) / elapsed
	if dlBps < 0 {
		dlBps = 0
	}
	if ulBps < 0 {
		ulBps = 0
	}

	m.prevDownload = curDl
	m.prevUpload = curUl
	m.prevTime = now

	pingOk := lastDelay > 0
	proxyAddr := serverIP
	if proxyAddr == "" {
		proxyAddr = "N/A"
	}

	bindLabel := localAddr
	if bindLabel == "" {
		bindLabel = "direct"
	}

	m.statsCh <- NewTrafficStats(dlBps, ulBps, lastDelay, bindLabel, proxyAddr, pingOk, diag)
	m.mu.Unlock()
}

func (m *Monitor) fetchTotals() (int64, int64, error) {
	req, err := http.NewRequest("GET", m.cfg.Addr()+"/connections", nil)
	if err != nil {
		return 0, 0, err
	}
	if m.cfg.Secret != "" {
		req.Header.Set("Authorization", "Bearer "+m.cfg.Secret)
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, err
	}

	var cr connectionsResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return 0, 0, err
	}
	return cr.DownloadTotal, cr.UploadTotal, nil
}

// tcpingDiagnostic does TCP connect on multiple ports and returns (RTT_ms, diag_string).
func tcpingDiagnostic(ip, bindIP string, timeout time.Duration) (int64, string) {
	ports := []int{443, 80, 22, 8080, 8443}
	bestMs := int64(-1)
	bestInfo := ""

	var dialer net.Dialer
	bindNote := ""
	if bindIP != "" {
		if parsed := net.ParseIP(bindIP); parsed != nil {
			dialer.LocalAddr = &net.TCPAddr{IP: parsed}
			bindNote = " (bind " + bindIP + ")"
		}
	}

	for _, port := range ports {
		addr := net.JoinHostPort(ip, fmt.Sprintf("%d", port))
		start := time.Now()
		conn, err := dialer.Dial("tcp", addr)
		elapsed := time.Since(start)
		ms := elapsed.Milliseconds()
		if ms < 1 && err == nil {
			ms = 1
		}

		if err == nil {
			conn.Close()
			info := fmt.Sprintf("TCP %d%s: SYN-ACK %dms", port, bindNote, ms)
			return ms, info
		}

		if ms > 0 && ms < int64(timeout.Milliseconds())/2 {
			// RST = port closed, but server responded
			info := fmt.Sprintf("TCP %d%s: RST %dms", port, bindNote, ms)
			if bestMs < 0 || ms < bestMs {
				bestMs = ms
				bestInfo = info
			}
			continue
		}

		// Full timeout
		continue
	}

	if bestMs > 0 {
		return bestMs, bestInfo
	}
	return -1, fmt.Sprintf("TCP %s%s: all ports timeout (2s)", ip, bindNote)
}
