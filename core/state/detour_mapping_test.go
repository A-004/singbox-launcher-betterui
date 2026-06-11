package state

import "testing"

// SPEC 077: DetourTag must survive the Source ↔ ProxySource round-trip in both
// directions and for both source types.
func TestDetourTag_ToProxySourceV4(t *testing.T) {
	sub := Source{Type: SourceTypeSubscription, Enabled: true, URL: "https://x/sub", DetourTag: "hop-out"}
	if got := sub.ToProxySourceV4().DetourTag; got != "hop-out" {
		t.Errorf("subscription: DetourTag = %q, want hop-out", got)
	}
	srv := Source{Type: SourceTypeServer, Enabled: true, URI: "vless://u@h:443#a", DetourTag: "hop-out"}
	if got := srv.ToProxySourceV4().DetourTag; got != "hop-out" {
		t.Errorf("server: DetourTag = %q, want hop-out", got)
	}
}

// syncLegacyFromConnections (Source → ProxySource) then syncConnectionsFromLegacy
// (ProxySource → Source) must preserve DetourTag for both types.
func TestDetourTag_LegacyRoundTrip(t *testing.T) {
	s := &State{}
	s.Connections.Sources = []Source{
		{ID: "a", Type: SourceTypeSubscription, Enabled: true, URL: "https://x/sub", DetourTag: "hop-sub"},
		{ID: "b", Type: SourceTypeServer, Enabled: true, URI: "vless://u@h:443#srv", DetourTag: "hop-srv"},
	}

	syncLegacyFromConnections(s)
	if len(s.ParserConfig.ParserConfig.Proxies) != 2 {
		t.Fatalf("proxies = %d, want 2", len(s.ParserConfig.ParserConfig.Proxies))
	}
	if got := s.ParserConfig.ParserConfig.Proxies[0].DetourTag; got != "hop-sub" {
		t.Errorf("legacy subscription DetourTag = %q, want hop-sub", got)
	}
	if got := s.ParserConfig.ParserConfig.Proxies[1].DetourTag; got != "hop-srv" {
		t.Errorf("legacy server DetourTag = %q, want hop-srv", got)
	}

	syncConnectionsFromLegacy(s)
	byID := map[string]Source{}
	for _, src := range s.Connections.Sources {
		// match by URL/URI since round-trip may re-key IDs
		switch src.Type {
		case SourceTypeSubscription:
			byID["sub"] = src
		case SourceTypeServer:
			byID["srv"] = src
		}
	}
	if got := byID["sub"].DetourTag; got != "hop-sub" {
		t.Errorf("round-trip subscription DetourTag = %q, want hop-sub", got)
	}
	if got := byID["srv"].DetourTag; got != "hop-srv" {
		t.Errorf("round-trip server DetourTag = %q, want hop-srv", got)
	}
}
