package cage

import "github.com/sylvinus/agent-vm/internal/managedproxy"

func (c Config) RequireManagedProxy(profile string) error {
	p, err := managedproxy.Applied(profile, c.ProxyURL, c.ProxyRevision)
	if err != nil {
		return err
	}
	return p.CheckWiredPolicy(c.ExitPolicy)
}
