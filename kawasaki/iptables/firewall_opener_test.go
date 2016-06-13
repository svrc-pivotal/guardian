package iptables_test

import (
	"errors"
	"net"

	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry-incubator/guardian/kawasaki/iptables"
	"github.com/cloudfoundry-incubator/guardian/kawasaki/iptables/iptablesfakes"
	"github.com/cloudfoundry/gunk/command_runner/fake_command_runner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("FirewallOpener", func() {
	var (
		logger             lager.Logger
		fakeRunner         *fake_command_runner.FakeCommandRunner
		fakeIPTablesDriver *iptablesfakes.FakeIPTablesDriver
		iptablesConfig     iptables.IPTablesConfig
		opener             *iptables.FirewallOpener
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		fakeRunner = fake_command_runner.New()

		fakeIPTablesDriver = new(iptablesfakes.FakeIPTablesDriver)
		iptablesConfig = iptables.NewConfig("prefix-")

		opener = iptables.NewFirewallOpener(
			iptablesConfig,
			fakeIPTablesDriver,
		)
	})

	Describe("Open", func() {
		It("uses the correct chain", func() {
			Expect(opener.Open(logger, "foo-bar-baz", garden.NetOutRule{})).To(Succeed())

			Expect(fakeIPTablesDriver.PrependRuleCallCount()).To(Equal(1))
			tableName, chainName, _ := fakeIPTablesDriver.PrependRuleArgsForCall(0)
			Expect(tableName).To(Equal("filter"))
			Expect(chainName).To(Equal(iptables.InstanceChain(iptablesConfig, "foo-bar-baz")))
		})

		It("applies the default rule", func() {
			Expect(opener.Open(logger, "foo-bar-baz", garden.NetOutRule{})).To(Succeed())

			Expect(fakeIPTablesDriver.PrependRuleCallCount()).To(Equal(1))
			_, _, rule := fakeIPTablesDriver.PrependRuleArgsForCall(0)
			Expect(rule).To(Equal(iptables.SingleFilterRule{}))
		})

		Context("when a portrange is specified for ProtocolALL", func() {
			It("returns a nice error message", func() {
				Expect(opener.Open(logger, "foo-bar-baz", garden.NetOutRule{
					Protocol: garden.ProtocolAll,
					Ports:    []garden.PortRange{{Start: 1, End: 5}},
				})).To(MatchError("Ports cannot be specified for Protocol ALL"))
			})
		})

		Context("when an invaild protocol is specified", func() {
			It("returns an error", func() {
				Expect(opener.Open(logger, "foo-bar-baz", garden.NetOutRule{
					Protocol: garden.Protocol(52),
				})).To(MatchError("invalid protocol: 52"))
			})
		})

		It("sets the protocol in the rule", func() {
			Expect(opener.Open(logger, "foo-bar-baz", garden.NetOutRule{
				Protocol: garden.ProtocolTCP,
			})).To(Succeed())

			Expect(fakeIPTablesDriver.PrependRuleCallCount()).To(Equal(1))
			_, _, rule := fakeIPTablesDriver.PrependRuleArgsForCall(0)
			Expect(rule).To(Equal(iptables.SingleFilterRule{
				Protocol: garden.ProtocolTCP,
			}))
		})

		It("sets the IMCP control in the rule", func() {
			icmpControl := &garden.ICMPControl{
				Type: garden.ICMPType(1),
			}

			Expect(opener.Open(logger, "foo-bar-baz", garden.NetOutRule{
				ICMPs: icmpControl,
			})).To(Succeed())

			Expect(fakeIPTablesDriver.PrependRuleCallCount()).To(Equal(1))
			_, _, rule := fakeIPTablesDriver.PrependRuleArgsForCall(0)
			Expect(rule).To(Equal(iptables.SingleFilterRule{
				ICMPs: icmpControl,
			}))
		})

		Describe("Log", func() {
			It("sets the log flag to the rule", func() {
				Expect(opener.Open(logger, "foo-bar-baz", garden.NetOutRule{
					Log: true,
				})).To(Succeed())

				Expect(fakeIPTablesDriver.PrependRuleCallCount()).To(Equal(1))
				_, _, rule := fakeIPTablesDriver.PrependRuleArgsForCall(0)
				Expect(rule).To(Equal(iptables.SingleFilterRule{
					Log: true,
				}))
			})
		})

		Context("when prepending the rule fails", func() {
			BeforeEach(func() {
				fakeIPTablesDriver.PrependRuleReturns(errors.New("i-lost-my-banana"))
			})

			It("returns the error", func() {
				Expect(opener.Open(logger, "foo-bar-baz", garden.NetOutRule{})).To(MatchError("i-lost-my-banana"))
			})
		})

		DescribeTable("networks and ports",
			func(netOut garden.NetOutRule, rules []iptables.SingleFilterRule) {
				Expect(opener.Open(logger, "foo-bar-baz", netOut)).To(Succeed())

				n := fakeIPTablesDriver.PrependRuleCallCount()
				Expect(n).To(Equal(len(rules)))

				for i := 0; i < n; i++ {
					_, _, appliedRule := fakeIPTablesDriver.PrependRuleArgsForCall(i)
					Expect(appliedRule).To(Equal(rules[i]))
				}
			},
			Entry("with a single destination IP specified",
				garden.NetOutRule{Networks: []garden.IPRange{{Start: net.ParseIP("1.2.3.4")}}},
				[]iptables.SingleFilterRule{
					{Networks: &garden.IPRange{Start: net.ParseIP("1.2.3.4")}},
				},
			),
			Entry("with multiple destination networks specified",
				garden.NetOutRule{Networks: []garden.IPRange{
					{Start: net.ParseIP("1.2.3.4")},
					{Start: net.ParseIP("2.2.3.4"), End: net.ParseIP("2.2.3.9")},
				}},
				[]iptables.SingleFilterRule{
					{Networks: &garden.IPRange{Start: net.ParseIP("1.2.3.4")}},
					{Networks: &garden.IPRange{Start: net.ParseIP("2.2.3.4"), End: net.ParseIP("2.2.3.9")}},
				},
			),
			Entry("with a single port specified",
				garden.NetOutRule{
					Protocol: garden.ProtocolTCP,
					Ports: []garden.PortRange{
						garden.PortRangeFromPort(22),
					},
				},
				[]iptables.SingleFilterRule{
					{Protocol: garden.ProtocolTCP, Ports: &garden.PortRange{Start: 22, End: 22}},
				},
			),
			Entry("with multiple ports specified",
				garden.NetOutRule{
					Protocol: garden.ProtocolTCP,
					Ports: []garden.PortRange{
						garden.PortRangeFromPort(22),
						garden.PortRange{Start: 1000, End: 10000},
					},
				},
				[]iptables.SingleFilterRule{
					{Protocol: garden.ProtocolTCP, Ports: &garden.PortRange{Start: 22, End: 22}},
					{Protocol: garden.ProtocolTCP, Ports: &garden.PortRange{Start: 1000, End: 10000}},
				},
			),
			Entry("with both networks and ports specified",
				garden.NetOutRule{
					Protocol: garden.ProtocolTCP,
					Networks: []garden.IPRange{
						{Start: net.ParseIP("1.2.3.4")},
						{Start: net.ParseIP("2.2.3.4"), End: net.ParseIP("2.2.3.9")},
					},
					Ports: []garden.PortRange{
						garden.PortRangeFromPort(22),
						garden.PortRange{Start: 1000, End: 10000},
					},
				},
				[]iptables.SingleFilterRule{
					{
						Protocol: garden.ProtocolTCP,
						Networks: &garden.IPRange{Start: net.ParseIP("1.2.3.4")},
						Ports:    &garden.PortRange{Start: 22, End: 22},
					},
					{
						Protocol: garden.ProtocolTCP,
						Networks: &garden.IPRange{Start: net.ParseIP("2.2.3.4"), End: net.ParseIP("2.2.3.9")},
						Ports:    &garden.PortRange{Start: 22, End: 22},
					},
					{
						Protocol: garden.ProtocolTCP,
						Networks: &garden.IPRange{Start: net.ParseIP("1.2.3.4")},
						Ports:    &garden.PortRange{Start: 1000, End: 10000},
					},
					{
						Protocol: garden.ProtocolTCP,
						Networks: &garden.IPRange{Start: net.ParseIP("2.2.3.4"), End: net.ParseIP("2.2.3.9")},
						Ports:    &garden.PortRange{Start: 1000, End: 10000},
					},
				},
			),
		)
	})
})
