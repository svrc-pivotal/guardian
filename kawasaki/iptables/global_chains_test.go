package iptables_test

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"

	"github.com/cloudfoundry-incubator/guardian/kawasaki/iptables"
	"github.com/cloudfoundry-incubator/guardian/kawasaki/iptables/iptablesfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Start", func() {
	var (
		fakeIPTablesRunner *iptablesfakes.FakeIPTablesRunner
		fakeIPTablesDriver *iptablesfakes.FakeIPTablesDriver
		denyNetworks       []string
		starter            *iptables.Starter
		ipTablesConfig     iptables.IPTablesConfig
	)

	BeforeEach(func() {
		fakeIPTablesRunner = new(iptablesfakes.FakeIPTablesRunner)
		fakeIPTablesDriver = new(iptablesfakes.FakeIPTablesDriver)
		ipTablesConfig = iptables.NewConfig("prefix-")
	})

	JustBeforeEach(func() {
		starter = iptables.NewStarter(
			"my-iptables",
			ipTablesConfig,
			fakeIPTablesDriver,
			fakeIPTablesRunner,
			true,
			"the-nic-prefix",
			denyNetworks,
		)
	})

	It("should setup the chains", func() {
		Expect(starter.Start()).To(Succeed())

		Expect(fakeIPTablesRunner.RunCallCount()).To(Equal(1))
		_, cmd := fakeIPTablesRunner.RunArgsForCall(0)
		Expect(cmd).To(Equal(&exec.Cmd{
			Path: "/bin/bash",
			Args: []string{"bash", "-c", iptables.SetupScript},
			Env: []string{
				fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
				"ACTION=setup",

				"GARDEN_IPTABLES_BIN=my-iptables",
				fmt.Sprintf("GARDEN_IPTABLES_FILTER_INPUT_CHAIN=%s", ipTablesConfig.InputChain),
				fmt.Sprintf("GARDEN_IPTABLES_FILTER_FORWARD_CHAIN=%s", ipTablesConfig.ForwardChain),
				fmt.Sprintf("GARDEN_IPTABLES_FILTER_DEFAULT_CHAIN=%s", ipTablesConfig.DefaultChain),
				fmt.Sprintf("GARDEN_IPTABLES_FILTER_INSTANCE_PREFIX=%s", ipTablesConfig.InstanceChainPrefix),
				fmt.Sprintf("GARDEN_IPTABLES_NAT_PREROUTING_CHAIN=%s", ipTablesConfig.PreroutingChain),
				fmt.Sprintf("GARDEN_IPTABLES_NAT_POSTROUTING_CHAIN=%s", ipTablesConfig.PostroutingChain),
				fmt.Sprintf("GARDEN_IPTABLES_NAT_INSTANCE_PREFIX=%s", ipTablesConfig.InstanceChainPrefix),
				"GARDEN_NETWORK_INTERFACE_PREFIX=the-nic-prefix",
				"GARDEN_IPTABLES_ALLOW_HOST_ACCESS=true",
			},
		}))
	})

	Context("when the chain already exists", func() {
		BeforeEach(func() {
			fakeIPTablesDriver.ChainExistsStub = func(table, chain string) bool {
				Expect(table).To(Equal("filter"))
				Expect(chain).To(Equal(ipTablesConfig.InputChain))

				return true
			}
		})

		It("should not setup the chains", func() {
			Expect(starter.Start()).To(Succeed())

			Expect(fakeIPTablesRunner.RunCallCount()).To(Equal(0))
		})
	})

	Context("when run errors", func() {
		BeforeEach(func() {
			fakeIPTablesRunner.RunReturns(errors.New("run-exploded"))
		})

		It("should return an appropriate error", func() {
			Expect(starter.Start()).To(MatchError("setting up global chains: run-exploded"))
		})
	})

	It("should reset the deny networks", func() {
		Expect(starter.Start()).To(Succeed())

		Expect(fakeIPTablesDriver.ResetChainCallCount()).To(Equal(1))
		table, chain, rules := fakeIPTablesDriver.ResetChainArgsForCall(0)
		Expect(table).To(Equal("filter"))
		Expect(chain).To(Equal(ipTablesConfig.DefaultChain))
		Expect(rules).To(Equal([]iptables.Rule{iptables.IPTablesFlags{
			"-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "--jump", "ACCEPT",
		}}))
	})

	Context("when reseting the default chain errors", func() {
		BeforeEach(func() {
			fakeIPTablesDriver.ResetChainReturns(errors.New("reset-chain-exploded"))
		})

		It("returns an appropriate error", func() {
			Expect(starter.Start()).To(MatchError("resetting default chain: reset-chain-exploded"))
		})
	})

	Context("when there are deny networks entries", func() {
		BeforeEach(func() {
			denyNetworks = []string{"10.0.0.0/30", "10.0.1.0/30"}
		})

		It("should apply deny networks rules to the default chain", func() {
			Expect(starter.Start()).To(Succeed())

			_, firstNet, err := net.ParseCIDR("10.0.0.0/30")
			Expect(err).NotTo(HaveOccurred())

			_, secondNet, err := net.ParseCIDR("10.0.1.0/30")
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeIPTablesDriver.ResetChainCallCount()).To(Equal(1))
			table, chain, rules := fakeIPTablesDriver.ResetChainArgsForCall(0)
			Expect(table).To(Equal("filter"))
			Expect(chain).To(Equal(ipTablesConfig.DefaultChain))
			Expect(rules).To(Equal([]iptables.Rule{
				iptables.IPTablesFlags{
					"-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "--jump", "ACCEPT",
				},
				iptables.RejectRule{
					DestinationIPNet: firstNet,
				},
				iptables.RejectRule{
					DestinationIPNet: secondNet,
				},
			}))
		})

		Context("when a deny network is invalid", func() {
			BeforeEach(func() {
				denyNetworks = []string{"10.0.0.0/30", "banana"}
			})

			It("should return an error", func() {
				Expect(starter.Start()).To(MatchError(ContainSubstring("resetting default chain")))
			})
		})
	})
})
