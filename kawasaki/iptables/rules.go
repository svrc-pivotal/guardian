package iptables

import (
	"fmt"
	"net"

	"github.com/cloudfoundry-incubator/garden"
)

type IPTablesFlags []string

func (flags IPTablesFlags) Flags(chain string) []string {
	return flags
}

type ForwardRule struct {
	DestinationIP   net.IP
	DestinationPort uint32
	ContainerIP     net.IP
	ContainerPort   uint32
}

func (r ForwardRule) Flags(chain string) []string {
	return IPTablesFlags([]string{
		"--protocol", "tcp",
		"--destination", r.DestinationIP.String(),
		"--destination-port", fmt.Sprintf("%d", r.DestinationPort),
		"--jump", "DNAT",
		"--to-destination", fmt.Sprintf("%s:%d", r.ContainerIP.String(), r.ContainerPort),
	})
}

type RejectRule struct {
	DestinationIPNet *net.IPNet
}

func (r RejectRule) Flags(chain string) []string {
	return []string{
		"--destination", r.DestinationIPNet.String(),
		"--jump", "REJECT",
	}
}

type SingleFilterRule struct {
	Protocol garden.Protocol
	Networks *garden.IPRange
	Ports    *garden.PortRange
	ICMPs    *garden.ICMPControl
	Log      bool
}

func (r SingleFilterRule) Flags(chain string) (params []string) {
	params = append(params, "--protocol", protocols[r.Protocol])

	network := r.Networks
	if network != nil {
		if network.Start != nil && network.End != nil {
			params = append(params, "-m", "iprange", "--dst-range", network.Start.String()+"-"+network.End.String())
		} else if network.Start != nil {
			params = append(params, "--destination", network.Start.String())
		} else if network.End != nil {
			params = append(params, "--destination", network.End.String())
		}
	}

	ports := r.Ports
	if ports != nil {
		if ports.End != ports.Start {
			params = append(params, "--destination-port", fmt.Sprintf("%d:%d", ports.Start, ports.End))
		} else {
			params = append(params, "--destination-port", fmt.Sprintf("%d", ports.Start))
		}
	}

	if r.ICMPs != nil {
		icmpType := fmt.Sprintf("%d", r.ICMPs.Type)
		if r.ICMPs.Code != nil {
			icmpType = fmt.Sprintf("%d/%d", r.ICMPs.Type, *r.ICMPs.Code)
		}

		params = append(params, "--icmp-type", icmpType)
	}

	if r.Log {
		params = append(params, "--goto", chain+"-log")
	} else {
		params = append(params, "--jump", "RETURN")
	}

	return params
}
