package kawasaki

import (
	"fmt"

	"github.com/cloudfoundry-incubator/guardian/kawasaki/dns"
)

type ResolvFactory struct {
	idMapReader dns.RootIdMapReader
}

func (r *ResolvFactory) CreateDNSResolvConfigurer(pid int, config NetworkConfig) DnsResolvConfigurer {

	configurer := &dns.ResolvConfigurer{
		HostsFileCompiler: &dns.HostsFileCompiler{
			Handle: config.ContainerHandle,
			IP:     config.ContainerIP,
		},
		ResolvFileCompiler: &dns.ResolvFileCompiler{
			HostResolvConfPath: "/etc/resolv.conf",
			HostIP:             config.BridgeIP,
			OverrideServers:    config.DNSServers,
		},
		FileWriter: &dns.RootfsWriter{
			RootfsPath: fmt.Sprintf("/proc/%d/root", pid),
			RootUid:    rootUid,
			RootGid:    rootGid,
		},
	}

	return configurer
}

func extractRootIds(pid int) (int, int) {
	return 0, 0
}
