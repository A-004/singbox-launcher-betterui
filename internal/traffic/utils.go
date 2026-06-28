package traffic

import (
	"fmt"
	"time"
)

// TrafficStats holds a single traffic snapshot with formatted strings.
type TrafficStats struct {
	Down      float64   // bytes/sec
	Up        float64   // bytes/sec
	DownStr   string    // formatted, e.g. "45.3 MB/s"
	UpStr     string    // formatted, e.g. "2.1 MB/s"
	IsActive  bool      // true when at least one direction > 0
	Timestamp time.Time
}

// NewTrafficStats creates a TrafficStats from raw bps values.
func NewTrafficStats(down, up float64) TrafficStats {
	return TrafficStats{
		Down:      down,
		Up:        up,
		DownStr:   FormatSpeed(down),
		UpStr:     FormatSpeed(up),
		IsActive:  down > 0 || up > 0,
		Timestamp: time.Now(),
	}
}

// FormatSpeed converts bytes-per-second to a human-readable string.
// Examples: "0 B/s", "1.2 KB/s", "45.8 MB/s", "1.34 GB/s".
func FormatSpeed(bps float64) string {
	switch {
	case bps >= 1_000_000_000:
		return fmt.Sprintf("%.2f GB/s", bps/1_000_000_000)
	case bps >= 1_000_000:
		return fmt.Sprintf("%.1f MB/s", bps/1_000_000)
	case bps >= 1_000:
		return fmt.Sprintf("%.0f KB/s", bps/1_000)
	default:
		return fmt.Sprintf("%.0f B/s", bps)
	}
}
