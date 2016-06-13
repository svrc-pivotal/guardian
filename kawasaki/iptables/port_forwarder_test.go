package iptables_test

import (
	"net"

	"github.com/cloudfoundry-incubator/guardian/kawasaki"
	"github.com/cloudfoundry-incubator/guardian/kawasaki/iptables"
	"github.com/cloudfoundry-incubator/guardian/kawasaki/iptables/iptablesfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PortForwarder", func() {
	var (
		fakeIPTablesConfig iptables.IPTablesConfig
		fakeIPTablesDriver *iptablesfakes.FakeIPTablesDriver
		forwarder          *iptables.PortForwarder
	)

	BeforeEach(func() {
		fakeIPTablesConfig = iptables.NewConfig("prefix-")
		fakeIPTablesDriver = new(iptablesfakes.FakeIPTablesDriver)
		forwarder = iptables.NewPortForwarder(fakeIPTablesConfig, fakeIPTablesDriver)
	})

	It("adds a NAT rule to forward the port", func() {
		Expect(forwarder.Forward(kawasaki.PortForwarderSpec{
			InstanceID:  "some-instance",
			ExternalIP:  net.ParseIP("5.6.7.8"),
			ContainerIP: net.ParseIP("1.2.3.4"),
			FromPort:    22,
			ToPort:      33,
		})).To(Succeed())

		Expect(fakeIPTablesDriver.AppendRuleCallCount()).To(Equal(1))
		table, chain, rule := fakeIPTablesDriver.AppendRuleArgsForCall(0)
		Expect(table).To(Equal("nat"))
		Expect(chain).To(Equal(iptables.InstanceChain(fakeIPTablesConfig, "some-instance")))
		Expect(rule).To(Equal(iptables.ForwardRule{
			DestinationIP:   net.ParseIP("5.6.7.8"),
			ContainerIP:     net.ParseIP("1.2.3.4"),
			DestinationPort: 22,
			ContainerPort:   33,
		}))
	})
})
