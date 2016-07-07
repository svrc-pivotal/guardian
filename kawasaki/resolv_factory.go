package kawasaki

import (
	"fmt"

	"github.com/cloudfoundry-incubator/guardian/kawasaki/dns"
)

// type IdMapReader struct{}

// func (i *IdMapReadeer) readId(id) (string, string)

// type ResolvFactory struct {
// 	idMapReader IdMapReader
// }

func (r *ResolvFactory) CreateDNSResolvConfigurer(pid int, config NetworkConfig) DnsResolvConfigurer {
	rootUid, rootGid := r.idMapReader.extractRootIds(pid)

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
