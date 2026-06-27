// Package trafficmonitor provides real-time network traffic monitoring.
//
// Uses direct Win32 API (iphlpapi.dll / GetIfTable2) to sample total
// bytes received/sent every second and computes speeds as deltas.
// No external dependencies — only syscall, unsafe, and the standard library.
//
// Layout of MIB_IF_TABLE2 (fixed offset from pointer):
//
//	Offset  Size  Field
//	0       8     NumEntries (uint64)
//	8       1328  MIB_IF_ROW2[0]  — InOctets at +800, OutOctets at +832
//	...     1328  MIB_IF_ROW2[N]
package trafficmonitor

import (
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	iphlpapi     = syscall.NewLazyDLL("iphlpapi.dll")
	getIfTable2  = iphlpapi.NewProc("GetIfTable2")
	freeMibTable = iphlpapi.NewProc("FreeMibTable")
)

const (
	mibIfRow2Size = 1328
	inOctetsOff   = 800
	outOctetsOff  = 832
)

// TrafficMonitor samples network I/O counters every tick and computes
// download (recv) / upload (sent) speeds in bytes per second.
type TrafficMonitor struct {
	mu       sync.Mutex
	ticker   *time.Ticker
	done     chan struct{}
	prevRecv uint64
	prevSent uint64
	prevTime time.Time
	onUpdate func(downloadBps, uploadBps float64)
	running  bool
}

// NewTrafficMonitor creates a monitor that calls onUpdate(downloadBps, uploadBps)
// every second with the computed speeds.
func NewTrafficMonitor(onUpdate func(downloadBps, uploadBps float64)) *TrafficMonitor {
	return &TrafficMonitor{
		onUpdate: onUpdate,
	}
}

// Start begins polling network I/O counters every second.
func (tm *TrafficMonitor) Start() {
	tm.mu.Lock()
	if tm.running {
		tm.mu.Unlock()
		return
	}
	tm.running = true
	tm.done = make(chan struct{})
	tm.ticker = time.NewTicker(1 * time.Second)
	tm.mu.Unlock()

	// Seed with initial counters — first delta will effectively be zero.
	recv, sent := readTotalBytes()
	tm.mu.Lock()
	tm.prevRecv = recv
	tm.prevSent = sent
	tm.prevTime = time.Now()
	tm.mu.Unlock()

	go tm.pollLoop()
}

// Stop stops the ticker and the polling goroutine.
func (tm *TrafficMonitor) Stop() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if !tm.running {
		return
	}
	tm.running = false
	if tm.ticker != nil {
		tm.ticker.Stop()
	}
	if tm.done != nil {
		close(tm.done)
	}
}

func (tm *TrafficMonitor) pollLoop() {
	for {
		select {
		case <-tm.ticker.C:
			tm.sample()
		case <-tm.done:
			return
		}
	}
}

func (tm *TrafficMonitor) sample() {
	curRecv, curSent := readTotalBytes()

	tm.mu.Lock()
	now := time.Now()
	elapsed := now.Sub(tm.prevTime).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}

	dlBps := float64(curRecv-tm.prevRecv) / elapsed
	ulBps := float64(curSent-tm.prevSent) / elapsed

	tm.prevRecv = curRecv
	tm.prevSent = curSent
	tm.prevTime = now
	onUpdate := tm.onUpdate
	tm.mu.Unlock()

	if onUpdate != nil {
		onUpdate(dlBps, ulBps)
	}
}

// readTotalBytes calls GetIfTable2 and sums InOctets/OutOctets across all
// interfaces. Returns (recvBytes, sentBytes).
func readTotalBytes() (uint64, uint64) {
	var tablePtr unsafe.Pointer
	ret, _, _ := getIfTable2.Call(uintptr(unsafe.Pointer(&tablePtr)))
	if ret != 0 { // NO_ERROR
		return 0, 0
	}
	if tablePtr == nil {
		return 0, 0
	}
	defer freeMibTable.Call(uintptr(tablePtr))

	// First 8 bytes = NumEntries
	numEntries := *(*uint64)(tablePtr)

	var totalRecv, totalSent uint64
	base := uintptr(tablePtr) + 8 // skip NumEntries

	for i := uint64(0); i < numEntries; i++ {
		entryBase := base + uintptr(i)*mibIfRow2Size
		inOctets := *(*uint64)(unsafe.Pointer(entryBase + inOctetsOff))
		outOctets := *(*uint64)(unsafe.Pointer(entryBase + outOctetsOff))
		totalRecv += inOctets
		totalSent += outOctets
	}

	return totalRecv, totalSent
}

// FormatSpeed converts bytes-per-second to a human-readable string (B/s, KB/s, MB/s).
func FormatSpeed(bps float64) string {
	switch {
	case bps >= 1_000_000:
		return fmt.Sprintf("%.1f MB/s", bps/1_000_000)
	case bps >= 1_000:
		return fmt.Sprintf("%.0f KB/s", bps/1_000)
	default:
		return fmt.Sprintf("%.0f B/s", bps)
	}
}


