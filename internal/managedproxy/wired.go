package managedproxy

import (
	"context"
	"errors"
	"net/netip"
	"runtime"
	"strings"

	"github.com/sylvinus/agent-vm/internal/exitpolicy"
	"github.com/sylvinus/agent-vm/internal/netguard"
)

// WiredNodeID is reserved for the administrator's local setting. Imported
// node IDs are hexadecimal hashes and can never select this outbound.
const WiredNodeID = "cage-wired"

type WiredRoute struct {
	Interface string `json:"interface"`
	IP        string `json:"ip"`
}

func (w WiredRoute) Policy() exitpolicy.Policy {
	return exitpolicy.Policy{Mode: "static", Scope: "country", IP: w.IP}.Normalized()
}

func (w WiredRoute) validate() error {
	if runtime.GOOS != "linux" {
		return errors.New("指定网卡出网仅支持 Linux")
	}
	if w.Interface == "" || len(w.Interface) > 15 || w.Interface == "." || w.Interface == ".." || strings.ContainsAny(w.Interface, "/:\\ \t\r\n\x00") {
		return errors.New("请输入有效的网卡名称")
	}
	ip, err := netip.ParseAddr(w.IP)
	public := netguard.Policy{}
	if err != nil || !ip.Is4() || !public.Allows("tcp", netip.AddrPortFrom(ip, 443)) {
		return errors.New("固定出口应填写公网 IPv4 地址")
	}
	return nil
}

func (s *Store) SelectWired(ctx context.Context, name, ip string) error {
	w := WiredRoute{Interface: strings.TrimSpace(name), IP: strings.TrimSpace(ip)}
	if err := w.validate(); err != nil {
		return err
	}
	if _, err := linuxPhysicalInterface(ctx, w.Interface); err != nil {
		return err
	}
	s.Wired, s.Selected, s.Chain = &w, WiredNodeID, nil
	return nil
}

func (s Store) WiredActive() bool { return s.Selected == WiredNodeID }

// PinnedInterface returns an empty name only for ordinary node routes.
func (s Store) PinnedInterface(ctx context.Context) (string, error) {
	if !s.WiredActive() {
		return "", nil
	}
	if s.Wired == nil {
		return "", errors.New("指定网卡配置未保存")
	}
	if err := s.Wired.validate(); err != nil {
		return "", err
	}
	return linuxPhysicalInterface(ctx, s.Wired.Interface)
}

func (s Store) EgressInterface(ctx context.Context) (string, error) {
	if name, err := s.PinnedInterface(ctx); name != "" || err != nil {
		return name, err
	}
	return PhysicalInterface(ctx)
}

func (s Store) CheckWiredPolicy(policy exitpolicy.Policy) error {
	if s.WiredActive() && (s.Wired == nil || policy.Normalized() != s.Wired.Policy()) {
		return errors.New("请在指定网卡出网设置中修改允许的公网 IPv4")
	}
	return nil
}
