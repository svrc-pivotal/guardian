package iptables_test

import (
	"fmt"
	"net"

	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry-incubator/guardian/kawasaki/iptables"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Rules", func() {
	Describe("IPTablesFlags Flags", func() {
		It("returns the provided flags", func() {
			flags := []string{
				"this", "is", "an", "iptables", "rule",
			}

			rule := iptables.IPTablesFlags(flags)
			Expect(rule.Flags("banana-chain")).To(Equal(flags))
		})
	})

	Describe("RejectRule Flags", func() {
		It("assigns the destination IP", func() {
			_, net, err := net.ParseCIDR("1.2.3.4/30")
			Expect(err).NotTo(HaveOccurred())

			rule := iptables.RejectRule{
				DestinationIPNet: net,
			}

			Expect(rule.Flags("banana-chain")).To(Equal([]string{
				"--destination", "1.2.3.4/30",
				"--jump", "REJECT",
			}))
		})
	})

	Describe("SingleFilterRule Flags", func() {
		It("assigns the protocol", func() {
			rule := iptables.SingleFilterRule{
				Protocol: garden.ProtocolTCP,
			}

			Expect(rule.Flags("banana-chain")).To(Equal([]string{"--protocol", "tcp", "--jump", "RETURN"}))
		})

		DescribeTable("networks",
			func(rule iptables.SingleFilterRule, networkArgs []string) {
				expectedArgs := []string{"--protocol", "tcp"}
				expectedArgs = append(expectedArgs, networkArgs...)
				expectedArgs = append(expectedArgs, []string{"--jump", "RETURN"}...)

				Expect(rule.Flags("banana-chain")).To(Equal(expectedArgs))
			},

			Entry("with start and end specified", iptables.SingleFilterRule{
				Protocol: garden.ProtocolTCP,
				Networks: &garden.IPRange{
					Start: net.ParseIP("122.100.10.10"),
					End:   net.ParseIP("122.100.10.40"),
				},
			}, []string{"-m", "iprange", "--dst-range", "122.100.10.10-122.100.10.40"}),

			Entry("with start specified", iptables.SingleFilterRule{
				Protocol: garden.ProtocolTCP,
				Networks: &garden.IPRange{
					Start: net.ParseIP("122.100.10.20"),
				},
			}, []string{"--destination", "122.100.10.20"}),

			Entry("with end specified", iptables.SingleFilterRule{
				Protocol: garden.ProtocolTCP,
				Networks: &garden.IPRange{
					End: net.ParseIP("122.100.10.11"),
				},
			}, []string{"--destination", "122.100.10.11"}),
		)

		Describe("ports", func() {
			Context("when the start and end ports are the same", func() {
				It("assigns the port", func() {
					rule := iptables.SingleFilterRule{
						Protocol: garden.ProtocolTCP,
						Ports: &garden.PortRange{
							Start: 112,
							End:   112,
						},
					}

					Expect(rule.Flags("banana-chain")).To(Equal([]string{
						"--protocol", "tcp",
						"--destination-port", "112",
						"--jump", "RETURN",
					}))
				})
			})

			Context("when the start and end ports are not the same", func() {
				It("assigns the ports", func() {
					rule := iptables.SingleFilterRule{
						Protocol: garden.ProtocolTCP,
						Ports: &garden.PortRange{
							Start: 112,
							End:   1112,
						},
					}

					Expect(rule.Flags("banana-chain")).To(Equal([]string{
						"--protocol", "tcp",
						"--destination-port", "112:1112",
						"--jump", "RETURN",
					}))
				})
			})
		})

		Describe("ICMPs", func() {
			It("assigns the icmp type", func() {
				rule := iptables.SingleFilterRule{
					Protocol: garden.ProtocolTCP,
					ICMPs: &garden.ICMPControl{
						Type: 0,
					},
				}

				Expect(rule.Flags("banana-chain")).To(Equal([]string{
					"--protocol", "tcp",
					"--icmp-type", "0",
					"--jump", "RETURN",
				}))
			})

			Context("when code is specified", func() {
				It("assigns the icmp type and the code", func() {
					code := garden.ICMPCode(1)
					rule := iptables.SingleFilterRule{
						Protocol: garden.ProtocolTCP,
						ICMPs: &garden.ICMPControl{
							Type: 0,
							Code: &code,
						},
					}

					Expect(rule.Flags("banana-chain")).To(Equal([]string{
						"--protocol", "tcp",
						"--icmp-type", "0/1",
						"--jump", "RETURN",
					}))
				})
			})
		})

		It("goes to the log chain when logging is enabled", func() {
			rule := iptables.SingleFilterRule{
				Protocol: garden.ProtocolTCP,
				Log:      true,
			}
			chain := "banana-chain"

			Expect(rule.Flags(chain)).To(Equal([]string{
				"--protocol", "tcp",
				"--goto", fmt.Sprintf("%s-log", chain),
			}))
		})
	})

	Describe("ForwardRule Flags", func() {
		It("assigns the destination IP", func() {
			rule := iptables.ForwardRule{
				DestinationIP:   net.ParseIP("5.6.7.8"),
				ContainerIP:     net.ParseIP("1.2.3.4"),
				DestinationPort: 22,
				ContainerPort:   33,
			}

			Expect(rule.Flags("banana-chain")).To(Equal([]string{
				"--protocol", "tcp",
				"--destination", "5.6.7.8",
				"--destination-port", "22",
				"--jump", "DNAT",
				"--to-destination", "1.2.3.4:33",
			}))
		})
	})
})
