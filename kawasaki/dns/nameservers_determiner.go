package dns

import (
	"net"

	"code.cloudfoundry.org/lager"
)

type NameserversDeterminer struct{}

func (n *NameserversDeterminer) Determine(log lager.Logger, resolvConfPath string, containerIp net.IP, overrideNameservers, additionalNameservers []net.IP) ([]net.IP, error) {
	return overrideNameservers, nil
}
