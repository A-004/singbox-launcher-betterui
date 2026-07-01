// Package traffic provides real-time traffic monitoring via Clash API
// and a ready-to-use Fyne widget.
package traffic

// ClashConfig holds Clash API connection parameters.
type ClashConfig struct {
	APIAddress string // e.g. "http://127.0.0.1:9090"
	Secret     string // optional Bearer token

	// LocalAddr is the IP address of the non-VPN network interface
	// (e.g. Ethernet 192.168.1.11) used as the source for TCP ping.
	// When set, tcping binds to this address, which forces the SYN
	// to go through the physical interface and bypass the tunnel.
	// Empty = no bind (SYN may be intercepted by the tunnel).
	LocalAddr string
}

// DefaultClashConfig returns the default local Clash API config.
func DefaultClashConfig() ClashConfig {
	return ClashConfig{
		APIAddress: "http://127.0.0.1:9090",
	}
}

// Addr returns the base API address.
func (c ClashConfig) Addr() string {
	if c.APIAddress == "" {
		return "http://127.0.0.1:9090"
	}
	return c.APIAddress
}
