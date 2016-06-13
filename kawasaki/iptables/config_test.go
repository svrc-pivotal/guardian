package iptables_test

import (
	. "github.com/cloudfoundry-incubator/guardian/kawasaki/iptables"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	var config IPTablesConfig

	BeforeEach(func() {
		chainPrefix := "prefix-"
		config = NewConfig(chainPrefix)
	})

	It("assigns the properties values", func() {
		Expect(config.InstanceChainPrefix).To(Equal("prefix-instance-"))
		Expect(config.PreroutingChain).To(Equal("prefix-prerouting"))
		Expect(config.PostroutingChain).To(Equal("prefix-postrouting"))
		Expect(config.InputChain).To(Equal("prefix-input"))
		Expect(config.ForwardChain).To(Equal("prefix-forward"))
		Expect(config.DefaultChain).To(Equal("prefix-default"))
	})

	It("returns the right instance chain", func() {
		chain := InstanceChain(config, "banana")
		Expect(chain).To(Equal("prefix-instance-banana"))
	})
})
