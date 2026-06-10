package subscription

import (
	"net/url"
	"strings"
	"testing"
)

// SPEC 073.2: AWG 2.0 header randomization ranges (H1-H4 = "lo-hi") collapse to
// the range start; the sing-box-lx endpoint shape takes a single uint32.
func TestParseWireGuardURI_AWGHeaderRanges(t *testing.T) {
	e := url.Values{}
	e.Set("h1", "43613244-384550127")
	e.Set("h2", "300-200") // reversed — smaller bound wins
	e.Set("h3", "10-x")    // garbage — skipped, node survives
	e.Set("h4", "992706287")
	node, err := parseWireGuardURI(awgTestURI("wireguard", e), nil)
	if err != nil || node == nil {
		t.Fatalf("parse failed: err=%v", err)
	}
	if v, _ := node.Outbound["h1"].(int64); v != 43613244 {
		t.Errorf("h1 = %v, want range start 43613244", node.Outbound["h1"])
	}
	if v, _ := node.Outbound["h2"].(int64); v != 200 {
		t.Errorf("h2 = %v, want 200 (reversed range)", node.Outbound["h2"])
	}
	if _, ok := node.Outbound["h3"]; ok {
		t.Error("garbage h3 must be skipped, not stored")
	}
	if v, _ := node.Outbound["h4"].(int64); v != 992706287 {
		t.Errorf("h4 = %v, want plain 992706287", node.Outbound["h4"])
	}
}

// Real-world AmneziaWG 2.0 .conf shape (synthetic keys): H ranges, an i1 blob,
// empty I2-I5 lines — pasted as text it must yield a complete AWG endpoint.
func TestConvertWGConfText_AWG2RangesAndEmptyI(t *testing.T) {
	conf := `[Interface]
Address = 10.8.1.25/32
DNS = 172.29.172.254, 1.0.0.1
PrivateKey = UFJJVkFURUtFWTAwMDAwMDAwMDAwMDAwMDAwMA=
Jc = 5
Jmin = 10
Jmax = 50
S1 = 28
S2 = 121
S3 = 25
S4 = 9
H1 = 43613244-384550127
H2 = 826869626-2105069164
H3 = 2124774725-2141151992
H4 = 2144594503-2146278491
I1 = <b 0x084481800001000300000000077469636b657473>
I2 =
I3 =
I4 =
I5 =

[Peer]
PublicKey = QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVo=
PresharedKey = UFNLMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDA=
AllowedIPs = 0.0.0.0/0, ::/0
Endpoint = 203.0.113.7:44733
PersistentKeepalive = 25
`
	uri, err := ConvertWGConfText(conf)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	node, err := ParseNode(uri, nil)
	if err != nil || node == nil {
		t.Fatalf("parse: %v", err)
	}
	want := map[string]int64{
		"jc": 5, "jmin": 10, "jmax": 50, "s1": 28, "s2": 121, "s3": 25, "s4": 9,
		"h1": 43613244, "h2": 826869626, "h3": 2124774725, "h4": 2144594503,
	}
	for k, w := range want {
		if v, _ := node.Outbound[k].(int64); v != w {
			t.Errorf("%s = %v, want %d", k, node.Outbound[k], w)
		}
	}
	if s, _ := node.Outbound["i1"].(string); !strings.HasPrefix(s, "<b 0x0844") {
		t.Errorf("i1 lost or mangled: %q", s)
	}
	for _, k := range []string{"i2", "i3", "i4", "i5"} {
		if _, ok := node.Outbound[k]; ok {
			t.Errorf("empty %s line must not produce a key", k)
		}
	}
	if v, _ := node.Outbound["mtu"].(int); v != 1280 {
		t.Errorf("mtu = %v, want AWG default 1280", node.Outbound["mtu"])
	}
}
