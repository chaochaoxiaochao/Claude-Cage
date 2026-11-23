package workbench

import (
	"strings"
	"testing"

	"github.com/sylvinus/agent-vm/internal/cage"
	"github.com/sylvinus/agent-vm/internal/managedproxy"
)

func TestWiredUsesExplicitIPAndReportsMismatch(t *testing.T) {
	s := testServer(t)
	cfg, err := loadConfig(s.configPath)
	if err != nil {
		t.Fatal(err)
	}
	previous, err := managedproxy.Load(s.profile)
	if err != nil {
		t.Fatal(err)
	}
	p := previous
	p.Wired = &managedproxy.WiredRoute{Interface: "eno1", IP: "130.12.50.18"}
	p.Selected = managedproxy.WiredNodeID
	cfg.ExitPolicy = selectProxyExitPolicy(&p, previous, cage.ExitPolicy{Mode: "dynamic"}, true)
	if cfg.ExitPolicy.Mode != "static" || cfg.ExitPolicy.IP != p.Wired.IP {
		t.Fatal("did not pin the exact explicitly configured IP")
	}
	if err := p.Save(s.profile); err != nil {
		t.Fatal(err)
	}
	if err := writePrivate(s.configPath, cfg); err != nil {
		t.Fatal(err)
	}
	if err := s.finishLocationProbe(cfg, Location{IP: "8.8.8.8", CountryCode: "SG"}, nil); err == nil || !strings.Contains(err.Error(), p.Wired.IP) {
		t.Fatalf("same-country IP change did not report mismatch: %v", err)
	}
	if err := s.finishLocationProbe(cfg, Location{IP: p.Wired.IP, CountryCode: "SG"}, nil); err != nil {
		t.Fatal(err)
	}
	input := settingsView(cfg, Preferences{Keyboard: "native"})
	input.ExitPolicy = cage.ExitPolicy{Mode: "dynamic"}
	if _, _, err := proposedSettings(input, cfg, s.profile); err == nil {
		t.Fatal("generic settings weakened wired IP lock")
	}
}
