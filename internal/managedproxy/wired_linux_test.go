package managedproxy

import (
	"context"
	"strings"
	"testing"

	"github.com/sylvinus/agent-vm/internal/exitpolicy"
)

func TestWiredCoreKeepsOnlyTheDirectOutbound(t *testing.T) {
	profile := t.TempDir()
	p, err := Load(profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Import("other", "", []byte(`proxies: [{name: other, type: socks5, server: proxy.example.com, port: 1080}]`), "other", 0, ""); err != nil {
		t.Fatal(err)
	}
	p.Selected = WiredNodeID
	p.Wired = &WiredRoute{Interface: "eno1", IP: "9.9.9.9"}
	p.GuestDNS = []string{"https://8.8.8.8/dns-query"}
	data, err := p.Runtime(profile)
	if err != nil {
		t.Fatal(err)
	}
	data, err = BindInterface(data, p.Wired.Interface)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"interface-name: eno1", "type: direct", "IP-CIDR,192.168.0.0/16,REJECT", "IN-USER,cage,CAGE", "https://8.8.8.8/dns-query#CAGE", "MATCH,REJECT"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("wired core missing %q", want)
		}
	}
	if strings.Contains(string(data), "proxy.example.com") || strings.Contains(string(data), "node-") {
		t.Fatal("wired core still loads alternate routes or their credentials")
	}
	if _, err := p.NodeEndpoint(p.Sources[0].Nodes[0].ID); err == nil {
		t.Fatal("an account could select another exit")
	}
	if err := p.CheckWiredPolicy(p.Wired.Policy()); err != nil {
		t.Fatal(err)
	}
	if err := p.CheckWiredPolicy(exitpolicy.Policy{Mode: "static", IP: "8.8.8.8"}); err == nil {
		t.Fatal("accepted policy that does not match stored wired IP")
	}
}

func TestWiredFailsClosedOnUnavailableInterface(t *testing.T) {
	p := Store{Selected: WiredNodeID, Wired: &WiredRoute{Interface: "cage-not-found", IP: "9.9.9.9"}}
	if got, err := p.EgressInterface(context.Background()); got != "" || err == nil {
		t.Fatalf("missing pinned NIC fell back: %q, %v", got, err)
	}
	for _, address := range []string{"", "192.168.0.1", "127.0.0.1", "::1"} {
		if err := (WiredRoute{Interface: "eno1", IP: address}).validate(); err == nil {
			t.Errorf("accepted non-public IPv4 %q", address)
		}
	}
	p.Chain = []string{WiredNodeID}
	if _, err := p.Route(); err == nil {
		t.Fatal("wired outbound was accepted as a chain hop")
	}
}
