package managedproxy

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

// Local smoke test; use a copy of the bundled mihomo beside this test binary.
// It never uses the normal Cage profile or requires root or a guest VM.
func TestWiredLiveCore(t *testing.T) {
	name, expected := os.Getenv("CAGE_TEST_WIRED_NIC"), os.Getenv("CAGE_TEST_WIRED_IP")
	if name == "" || expected == "" {
		t.Skip("set CAGE_TEST_WIRED_NIC and CAGE_TEST_WIRED_IP to test a live pinned uplink")
	}
	profile := t.TempDir()
	p, err := Load(profile)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p.Port = listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	p.GuestDNS = []string{"https://8.8.8.8/dns-query"}
	p.BootstrapDNS = []string{"https://223.5.5.5/dns-query"}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := p.SelectWired(ctx, name, expected); err != nil {
		t.Fatal(err)
	}
	if err := Start(ctx, profile, p); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := Stop(profile); err != nil {
			t.Error(err)
		}
	})
	proxyURL, _ := url.Parse(p.Endpoint())
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 12 * time.Second}
	resp, err := client.Get("https://ipv4.geojs.io/v1/ip/geo.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var result struct {
		IP      string `json:"ip"`
		Country string `json:"country_code"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&result); err != nil || resp.StatusCode != 200 || result.IP != expected {
		t.Fatalf("exit check: IP=%s country=%s HTTP=%d error=%v", result.IP, result.Country, resp.StatusCode, err)
	}
	t.Logf("core exit: %s %s", result.IP, result.Country)
}
