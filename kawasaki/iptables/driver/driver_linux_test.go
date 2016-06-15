package driver_test

import (
	"fmt"
	"os/exec"

	"github.com/cloudfoundry-incubator/guardian/kawasaki/iptables"
	"github.com/cloudfoundry-incubator/guardian/kawasaki/iptables/driver"
	fakes "github.com/cloudfoundry-incubator/guardian/kawasaki/iptables/iptablesfakes"
	"github.com/cloudfoundry/gunk/command_runner/fake_command_runner"
	. "github.com/cloudfoundry/gunk/command_runner/fake_command_runner/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("IPTables controller", func() {
	var (
		netnsName          string
		prefix             string
		fakeRunner         *fake_command_runner.FakeCommandRunner
		iptablesController iptables.IPTablesDriver
	)

	BeforeEach(func() {
		netnsName = fmt.Sprintf("ginkgo-netns-%d", GinkgoParallelNode())
		makeNamespace(netnsName)

		fakeRunner = fake_command_runner.New()
		fakeRunner.WhenRunning(fake_command_runner.CommandSpec{},
			func(cmd *exec.Cmd) error {
				return wrapCmdInNs(netnsName, cmd).Run()
			},
		)

		prefix = fmt.Sprintf("g-%d", GinkgoParallelNode())
		iptablesController = driver.New("/sbin/iptables", fakeRunner, prefix)
	})

	AfterEach(func() {
		deleteNamespace(netnsName)
	})

	It("uses the iptables binary argument as the path to iptables", func() {
		iptablesController = driver.New("/silly/bin/path", fakeRunner, prefix)
		iptablesController.CreateChain("filter", "test-chain")

		cmd := fake_command_runner.CommandSpec{
			Path: "sh",
			Args: []string{"-c", "/silly/bin/path --wait --table filter -N test-chain"},
		}

		Expect(fakeRunner).To(HaveExecutedSerially(cmd))
	})

	Describe("CreateChain", func() {
		It("creates the chain", func() {
			Expect(iptablesController.CreateChain("filter", "test-chain")).To(Succeed())

			sess, err := gexec.Start(wrapCmdInNs(netnsName, exec.Command("iptables", "-L", "test-chain")), GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(gexec.Exit(0))
		})

		Context("when the table is nat", func() {
			It("creates the nat chain", func() {
				Expect(iptablesController.CreateChain("nat", "test-chain")).To(Succeed())

				sess, err := gexec.Start(wrapCmdInNs(netnsName, exec.Command("iptables", "-t", "nat", "-L", "test-chain")), GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(gexec.Exit(0))
			})
		})

		Context("when the chain exists", func() {
			BeforeEach(func() {
				Expect(iptablesController.CreateChain("nat", "test-chain")).To(Succeed())
			})

			It("returns an error", func() {
				Expect(iptablesController.CreateChain("nat", "test-chain")).NotTo(Succeed())
			})
		})
	})

	Describe("DeleteChain", func() {

		Context("when there is a chain to delete", func() {
			BeforeEach(func() {
				Expect(iptablesController.CreateChain("filter", "test-chain")).To(Succeed())
			})

			It("deletes the chain", func() {
				Expect(iptablesController.DeleteChain("filter", "test-chain")).To(Succeed())

				sess, err := gexec.Start(wrapCmdInNs(netnsName, exec.Command("iptables", "-L", "test-chain")), GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(gexec.Exit(1))
			})

			Context("when the table is nat", func() {
				BeforeEach(func() {
					Expect(iptablesController.CreateChain("nat", "test-chain")).To(Succeed())
				})

				It("deletes the nat chain", func() {
					Expect(iptablesController.DeleteChain("nat", "test-chain")).To(Succeed())

					sess, err := gexec.Start(wrapCmdInNs(netnsName, exec.Command("iptables", "-t", "nat", "-L", "test-chain")), GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Eventually(sess).Should(gexec.Exit(1))
				})
			})
		})

		Context("when the chain does not exist", func() {
			It("does not return an error", func() {
				Expect(iptablesController.DeleteChain("filter", "test-non-existing-chain")).To(Succeed())
			})
		})
	})

	itFlushesTheChain := func(table, chain string) {
		buff := gbytes.NewBuffer()
		sess, err := gexec.Start(wrapCmdInNs(netnsName, exec.Command("iptables", "-t", table, "-S", chain)), buff, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess).Should(gexec.Exit(0))
		Consistently(buff).ShouldNot(gbytes.Say(fmt.Sprintf("-A %s", chain)))
	}

	Describe("FlushChain", func() {
		var table string

		BeforeEach(func() {
			table = "filter"
		})

		Context("when there is a chain to flush", func() {
			JustBeforeEach(func() {
				Expect(iptablesController.CreateChain(table, "test-chain")).To(Succeed())

				sess, err := gexec.Start(wrapCmdInNs(netnsName, exec.Command("iptables", "-t", table, "-A", "test-chain", "-j", "ACCEPT")), GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(gexec.Exit(0))
			})

			It("flushes the chain", func() {
				Expect(iptablesController.FlushChain(table, "test-chain")).To(Succeed())

				itFlushesTheChain(table, "test-chain")
			})

			Context("when the table is nat", func() {
				BeforeEach(func() {
					table = "nat"
				})

				It("flushes the nat chain", func() {
					Expect(iptablesController.FlushChain(table, "test-chain")).To(Succeed())

					buff := gbytes.NewBuffer()
					sess, err := gexec.Start(wrapCmdInNs(netnsName, exec.Command("iptables", "-t", table, "-S", "test-chain")), buff, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Eventually(sess).Should(gexec.Exit(0))
					Consistently(buff).ShouldNot(gbytes.Say("-A test-chain"))
				})
			})
		})

		Context("when the chain does not exist", func() {
			It("does not return an error", func() {
				Expect(iptablesController.FlushChain("filter", "test-non-existing-chain")).To(Succeed())
			})
		})
	})

	Describe("ResetChain", func() {
		var table string

		BeforeEach(func() {
			table = "filter"
		})

		Context("when there is a chain to reset", func() {

			JustBeforeEach(func() {
				Expect(iptablesController.CreateChain(table, "test-chain")).To(Succeed())
				Expect(iptablesController.AppendRule(table, "test-chain", iptables.IPTablesFlags{"-j", "REJECT"})).To(Succeed())
			})

			It("flushes the chain", func() {
				Expect(iptablesController.ResetChain(table, "test-chain", []iptables.Rule{})).To(Succeed())

				itFlushesTheChain(table, "test-chain")
			})

			It("appends new rules provided", func() {
				Expect(iptablesController.ResetChain(table, "test-chain", []iptables.Rule{
					iptables.IPTablesFlags{
						"-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "ACCEPT",
					},
					iptables.IPTablesFlags{
						"-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "REJECT",
					},
				})).To(Succeed())

				buff := gbytes.NewBuffer()
				sess, err := gexec.Start(wrapCmdInNs(netnsName, exec.Command("iptables", "-t", table, "-S", "test-chain")), buff, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(gexec.Exit(0))
				Expect(buff.Contents()).To(ContainSubstring(`-m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
-A test-chain -m conntrack --ctstate RELATED,ESTABLISHED -j REJECT`))
			})
		})

		Context("when the chain does not exist", func() {
			It("returns an error", func() {
				Expect(iptablesController.ResetChain("filter", "test-non-existing-chain", []iptables.Rule{})).NotTo(Succeed())
			})
		})
	})

	Describe("ChainExists", func() {
		It("returns true when the chain exists", func() {
			Expect(iptablesController.CreateChain("nat", "test-chain")).To(Succeed())
			Expect(iptablesController.ChainExists("nat", "test-chain")).To(BeTrue())
		})

		Context("when the chain does not exist", func() {
			It("returns false", func() {
				Expect(iptablesController.ChainExists("nat", "test-chain")).To(BeFalse())
			})
		})
	})

	Describe("DeleteChainReferences", func() {
		var table string

		BeforeEach(func() {
			table = "filter"
		})

		Context("when there is a chain to references from", func() {
			JustBeforeEach(func() {
				Expect(iptablesController.CreateChain(table, "test-chain-1")).To(Succeed())
				Expect(iptablesController.CreateChain(table, "test-chain-2")).To(Succeed())

				sess, err := gexec.Start(wrapCmdInNs(netnsName, exec.Command("iptables", "-t", table, "-A", "test-chain-1", "-j", "test-chain-2")), GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(gexec.Exit(0))
			})

			It("deletes the references", func() {
				Expect(iptablesController.DeleteChainReferences(table, "test-chain-1", "test-chain-2")).To(Succeed())

				Eventually(func() string {
					buff := gbytes.NewBuffer()
					sess, err := gexec.Start(wrapCmdInNs(netnsName, exec.Command("iptables", "-t", table, "-S", "test-chain-1")), buff, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())

					Eventually(sess).Should(gexec.Exit(0))

					return string(buff.Contents())
				}).ShouldNot(ContainSubstring("test-chain-2"))
			})
		})
	})

	Describe("AppendRule", func() {

		Context("when a chain exists", func() {
			BeforeEach(func() {
				Expect(iptablesController.CreateChain("nat", "test-chain")).To(Succeed())
			})

			It("appends the rule to that chain", func() {
				fakeTCPRule := new(fakes.FakeRule)
				fakeTCPRule.FlagsReturns([]string{"--protocol", "tcp"})
				fakeUDPRule := new(fakes.FakeRule)
				fakeUDPRule.FlagsReturns([]string{"--protocol", "udp"})

				Expect(iptablesController.AppendRule("nat", "test-chain", fakeTCPRule)).To(Succeed())
				Expect(iptablesController.AppendRule("nat", "test-chain", fakeUDPRule)).To(Succeed())

				buff := gbytes.NewBuffer()
				sess, err := gexec.Start(wrapCmdInNs(netnsName, exec.Command("iptables", "-t", "nat", "-S", "test-chain")), buff, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(gexec.Exit(0))
				Expect(buff).To(gbytes.Say("-A test-chain -p tcp\n-A test-chain -p udp"))
			})
		})

		Context("when the chain does not exist", func() {
			It("returns an error", func() {
				fakeRule := new(fakes.FakeRule)
				fakeRule.FlagsReturns([]string{})

				Expect(iptablesController.AppendRule("filter", "test-chain", fakeRule)).NotTo(Succeed())
			})
		})
	})

	Describe("PrependRule", func() {
		Context("when a chain exists", func() {
			BeforeEach(func() {
				Expect(iptablesController.CreateChain("nat", "test-chain")).To(Succeed())
			})

			It("prepends the rule", func() {
				fakeTCPRule := new(fakes.FakeRule)
				fakeTCPRule.FlagsReturns([]string{"--protocol", "tcp"})
				fakeUDPRule := new(fakes.FakeRule)
				fakeUDPRule.FlagsReturns([]string{"--protocol", "udp"})

				Expect(iptablesController.PrependRule("nat", "test-chain", fakeTCPRule)).To(Succeed())
				Expect(iptablesController.PrependRule("nat", "test-chain", fakeUDPRule)).To(Succeed())

				buff := gbytes.NewBuffer()
				sess, err := gexec.Start(wrapCmdInNs(netnsName, exec.Command("iptables", "-t", "nat", "-S", "test-chain")), buff, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(gexec.Exit(0))
				Expect(buff).To(gbytes.Say("-A test-chain -p udp\n-A test-chain -p tcp"))
			})
		})

		Context("when no chain exists", func() {
			It("returns an error", func() {
				fakeRule := new(fakes.FakeRule)
				fakeRule.FlagsReturns([]string{})

				Expect(iptablesController.PrependRule("filter", "test-chain", fakeRule)).NotTo(Succeed())
			})
		})
	})
})

func makeNamespace(nsName string) {
	sess, err := gexec.Start(exec.Command("ip", "netns", "add", nsName), GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess).Should(gexec.Exit(0))
}

func deleteNamespace(nsName string) {
	sess, err := gexec.Start(exec.Command("ip", "netns", "delete", nsName), GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess).Should(gexec.Exit(0))
}

func wrapCmdInNs(nsName string, cmd *exec.Cmd) *exec.Cmd {
	wrappedCmd := exec.Command("ip", "netns", "exec", nsName)
	wrappedCmd.Args = append(wrappedCmd.Args, cmd.Args...)
	wrappedCmd.Stdout = cmd.Stdout
	wrappedCmd.Stderr = cmd.Stderr
	return wrappedCmd
}
