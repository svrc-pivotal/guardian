package dns_test

import (
	. "code.cloudfoundry.org/guardian/kawasaki/dns"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
)

var _ = FDescribe("NameserversDeterminer", func() {
	Describe("Determine", func() {
		var (
			log                   lager.Logger
			containerIP           net.IP
			overrideNameservers   []net.IP
			additionalNameservers []net.IP
			ip1, ip2              net.IP
		)

		BeforeEach(func() {
			log = lagertest.NewTestLogger("NameserverDeterminer-test")
		})

		Context("when passed one or more overrideNameservers", func() {
			BeforeEach(func() {
				ip1 = net.ParseIP("10.0.0.1")
				ip2 = net.ParseIP("10.0.0.2")
				overrideNameservers = []net.IP{ip1, ip2}
			})

			It("returns the overrideNameservers only", func() {
				determiner := &NameserversDeterminer{}

				nameservers, err := determiner.Determine(log, "/etc/resolv.conf", containerIP, overrideNameservers, additionalNameservers)
				Expect(err).NotTo(HaveOccurred())

				Expect(nameservers).To(Equal([]net.IP{ip1, ip2}))
			})
		})
	})
})
