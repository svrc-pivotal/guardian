package iptables_test

import (
	"errors"
	"fmt"
	"net"
	"os/exec"

	"github.com/cloudfoundry-incubator/guardian/kawasaki/iptables"
	"github.com/cloudfoundry-incubator/guardian/kawasaki/iptables/iptablesfakes"
	"github.com/cloudfoundry-incubator/guardian/logging"
	"github.com/cloudfoundry/gunk/command_runner/fake_command_runner"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/cloudfoundry/gunk/command_runner/fake_command_runner/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Create", func() {
	var (
		fakeRunner         *fake_command_runner.FakeCommandRunner
		fakeIPTablesDriver *iptablesfakes.FakeIPTablesDriver
		iptablesConfig     iptables.IPTablesConfig
		creator            *iptables.InstanceChainCreator
		bridgeName         string
		ip                 net.IP
		network            *net.IPNet
		logger             lager.Logger
	)

	BeforeEach(func() {
		var err error

		fakeRunner = fake_command_runner.New()
		fakeIPTablesDriver = new(iptablesfakes.FakeIPTablesDriver)
		logger = lagertest.NewTestLogger("test")

		bridgeName = "some-bridge"
		ip, network, err = net.ParseCIDR("1.2.3.4/28")
		Expect(err).NotTo(HaveOccurred())

		iptablesConfig = iptables.NewConfig("prefix-")

		creator = iptables.NewInstanceChainCreator(
			"my-iptables",
			iptablesConfig,
			fakeIPTablesDriver,
			&iptables.IPTablesLoggingRunner{
				Runner: logging.Runner{
					CommandRunner: fakeRunner,
					Logger:        lagertest.NewTestLogger("iptables-runner"),
				},
			},
		)
	})

	Describe("Create", func() {
		Describe("The NAT instance chain", func() {
			It("creates the nat instance chain", func() {
				Expect(creator.Create(logger, "some-id", bridgeName, ip, network)).To(Succeed())

				Expect(fakeIPTablesDriver.CreateChainCallCount()).NotTo(BeZero())
				table, chain := fakeIPTablesDriver.CreateChainArgsForCall(0)
				Expect(table).To(Equal("nat"))
				Expect(chain).To(Equal(iptables.InstanceChain(iptablesConfig, "some-id")))
			})

			Context("when creating the nat instance chain errors", func() {
				BeforeEach(func() {
					fakeIPTablesDriver.CreateChainReturns(errors.New("creating-nat-chain-failed"))
				})

				It("returns the underlying error", func() {
					Expect(creator.Create(logger, "some-id", bridgeName, ip, network)).To(MatchError("creating-nat-chain-failed"))
				})
			})

			It("configures the nat instance chain", func() {
				instanceChain := iptables.InstanceChain(iptablesConfig, "some-id")
				specs := []fake_command_runner.CommandSpec{
					{
						Path: "my-iptables",
						Args: []string{"--wait", "--table", "nat", "-A", iptablesConfig.PreroutingChain,
							"--jump", instanceChain},
					},
					{
						Path: "sh",
						Args: []string{"-c", fmt.Sprintf(
							`(my-iptables --wait --table nat -S %s | grep "\-j MASQUERADE\b" | grep -q -F -- "-s %s") || my-iptables --wait --table nat -A %s --source %s ! --destination %s --jump MASQUERADE`,
							iptablesConfig.PostroutingChain, network.String(), iptablesConfig.PostroutingChain,
							network.String(), network.String(),
						)},
					},
				}

				Expect(creator.Create(logger, "some-id", bridgeName, ip, network)).To(Succeed())
				Expect(fakeRunner).To(HaveExecutedSerially(specs...))
			})

			Context("when configuring the nat instance chain errors", func() {
				BeforeEach(func() {
					instanceChain := iptables.InstanceChain(iptablesConfig, "some-id")
					cmd := fake_command_runner.CommandSpec{
						Path: "my-iptables",
						Args: []string{"--wait", "--table", "nat", "-A", iptablesConfig.PreroutingChain, "--jump", instanceChain},
					}

					fakeRunner.WhenRunning(cmd, func(cmd *exec.Cmd) error {
						cmd.Stderr.Write([]byte("configuring-nat-chain-failed"))
						return errors.New("configuring-nat-chain-failed")
					})
				})

				It("returns the underlying error", func() {
					Expect(creator.Create(logger, "some-id", bridgeName, ip, network)).To(MatchError("iptables create-instance-chains: configuring-nat-chain-failed"))
				})
			})
		})

		Describe("The Filter instance chain", func() {
			It("creates the filter instance chain", func() {
				Expect(creator.Create(logger, "some-id", bridgeName, ip, network)).To(Succeed())

				Expect(fakeIPTablesDriver.CreateChainCallCount()).NotTo(BeZero())
				table, chain := fakeIPTablesDriver.CreateChainArgsForCall(1)
				Expect(table).To(Equal("filter"))
				Expect(chain).To(Equal(iptables.InstanceChain(iptablesConfig, "some-id")))
			})

			Context("when creating the filter instance chain errors", func() {
				BeforeEach(func() {
					fakeIPTablesDriver.CreateChainStub = func(table, chain string) error {
						instanceChain := iptables.InstanceChain(iptablesConfig, "some-id")
						if table == "filter" && chain == instanceChain {
							return errors.New("creating-filter-chain-failed")
						}

						return nil
					}
				})

				It("returns the underlying error", func() {
					Expect(creator.Create(logger, "some-id", bridgeName, ip, network)).To(MatchError("creating-filter-chain-failed"))
				})
			})

			It("configures the filter instance chain", func() {
				instanceChain := iptables.InstanceChain(iptablesConfig, "some-id")
				specs := []fake_command_runner.CommandSpec{
					{
						Path: "my-iptables",
						Args: []string{"--wait", "-A", instanceChain,
							"-s", network.String(), "-d", network.String(), "-j", "ACCEPT"},
					},
					{
						Path: "my-iptables",
						Args: []string{"--wait", "-A", instanceChain,
							"--goto", iptablesConfig.DefaultChain},
					},
					{
						Path: "my-iptables",
						Args: []string{"--wait", "-I", iptablesConfig.ForwardChain, "2", "--in-interface", bridgeName,
							"--source", ip.String(), "--goto", instanceChain},
					},
				}

				Expect(creator.Create(logger, "some-id", bridgeName, ip, network)).To(Succeed())
				Expect(fakeRunner).To(HaveExecutedSerially(specs...))
			})

			Context("when configuring the filter instance chain errors", func() {
				BeforeEach(func() {
					instanceChain := iptables.InstanceChain(iptablesConfig, "some-id")
					cmd := fake_command_runner.CommandSpec{
						Path: "my-iptables",
						Args: []string{"--wait", "-A", instanceChain,
							"-s", network.String(), "-d", network.String(), "-j", "ACCEPT"},
					}

					fakeRunner.WhenRunning(cmd, func(cmd *exec.Cmd) error {
						cmd.Stderr.Write([]byte("configuring-filter-chain-failed"))
						return errors.New("configuring-filter-chain-failed")
					})
				})

				It("returns the underlying error", func() {
					Expect(creator.Create(logger, "some-id", bridgeName, ip, network)).To(MatchError("iptables create-instance-chains: configuring-filter-chain-failed"))
				})
			})
		})

		Describe("The Logging instance chain", func() {
			It("creates the logging instance chain", func() {
				Expect(creator.Create(logger, "some-id", bridgeName, ip, network)).To(Succeed())

				Expect(fakeIPTablesDriver.CreateChainCallCount()).NotTo(BeZero())
				table, chain := fakeIPTablesDriver.CreateChainArgsForCall(2)
				Expect(table).To(Equal("filter"))
				Expect(chain).To(Equal(iptables.InstanceChain(iptablesConfig, "some-id") + "-log"))
			})

			Context("when creating the logging instance chain errors", func() {
				BeforeEach(func() {
					fakeIPTablesDriver.CreateChainStub = func(table, chain string) error {
						instanceChain := iptables.InstanceChain(iptablesConfig, "some-id") + "-log"
						if table == "filter" && chain == instanceChain {
							return errors.New("creating-filter-chain-failed")
						}

						return nil
					}
				})

				It("returns the underlying error", func() {
					Expect(creator.Create(logger, "some-id", bridgeName, ip, network)).To(MatchError("creating-filter-chain-failed"))
				})
			})

			It("configures the logging instance chain", func() {
				instanceChain := iptables.InstanceChain(iptablesConfig, "some-id") + "-log"
				specs := []fake_command_runner.CommandSpec{
					{
						Path: "my-iptables",
						Args: []string{"--wait", "-A", instanceChain, "-m", "conntrack", "--ctstate", "NEW,UNTRACKED,INVALID",
							"--protocol", "tcp", "--jump", "LOG", "--log-prefix", "some-id"},
					},
					{
						Path: "my-iptables",
						Args: []string{"--wait", "-A", instanceChain, "--jump", "RETURN"},
					},
				}

				Expect(creator.Create(logger, "some-id", bridgeName, ip, network)).To(Succeed())
				Expect(fakeRunner).To(HaveExecutedSerially(specs...))
			})

			Context("when configuring the logging instance chain errors", func() {
				BeforeEach(func() {
					instanceChain := iptables.InstanceChain(iptablesConfig, "some-id") + "-log"
					cmd := fake_command_runner.CommandSpec{
						Path: "my-iptables",
						Args: []string{"--wait", "-A", instanceChain, "--jump", "RETURN"},
					}

					fakeRunner.WhenRunning(cmd, func(cmd *exec.Cmd) error {
						cmd.Stderr.Write([]byte("configuring-log-chain-failed"))
						return errors.New("configuring-log-chain-failed")
					})
				})

				It("returns the underlying error", func() {
					Expect(creator.Create(logger, "some-id", bridgeName, ip, network)).To(MatchError("iptables create-instance-chains: configuring-log-chain-failed"))
				})
			})
		})
	})

	Describe("Destroy", func() {
		It("flushes the nat instance chain", func() {
			Expect(creator.Destroy(logger, "some-id")).To(Succeed())

			Expect(fakeIPTablesDriver.FlushChainCallCount()).NotTo(BeZero())
			table, chain := fakeIPTablesDriver.FlushChainArgsForCall(0)
			Expect(table).To(Equal("nat"))
			instanceChain := iptables.InstanceChain(iptablesConfig, "some-id")
			Expect(chain).To(Equal(instanceChain))
		})

		It("removes the references to the nat instance chain", func() {
			Expect(creator.Destroy(logger, "some-id")).To(Succeed())

			Expect(fakeIPTablesDriver.DeleteChainReferencesCallCount()).NotTo(BeZero())
			table, targetChain, referencedChain := fakeIPTablesDriver.DeleteChainReferencesArgsForCall(0)
			Expect(table).To(Equal("nat"))
			instanceChain := iptables.InstanceChain(iptablesConfig, "some-id")
			Expect(targetChain).To(Equal(iptablesConfig.PreroutingChain))
			Expect(referencedChain).To(Equal(instanceChain))
		})

		It("deletes the nat instance chain", func() {
			Expect(creator.Destroy(logger, "some-id")).To(Succeed())

			Expect(fakeIPTablesDriver.DeleteChainCallCount()).NotTo(BeZero())
			table, chain := fakeIPTablesDriver.DeleteChainArgsForCall(0)
			Expect(table).To(Equal("nat"))
			instanceChain := iptables.InstanceChain(iptablesConfig, "some-id")
			Expect(chain).To(Equal(instanceChain))
		})

		It("flushes the filter instance chain", func() {
			Expect(creator.Destroy(logger, "some-id")).To(Succeed())

			Expect(fakeIPTablesDriver.FlushChainCallCount()).NotTo(BeZero())
			table, chain := fakeIPTablesDriver.FlushChainArgsForCall(1)
			Expect(table).To(Equal("filter"))
			instanceChain := iptables.InstanceChain(iptablesConfig, "some-id")
			Expect(chain).To(Equal(instanceChain))
		})

		It("removes the references to the filter instance chain", func() {
			Expect(creator.Destroy(logger, "some-id")).To(Succeed())

			Expect(fakeIPTablesDriver.DeleteChainReferencesCallCount()).NotTo(BeZero())
			table, targetChain, referencedChain := fakeIPTablesDriver.DeleteChainReferencesArgsForCall(1)
			Expect(table).To(Equal("filter"))
			instanceChain := iptables.InstanceChain(iptablesConfig, "some-id")
			Expect(targetChain).To(Equal(iptablesConfig.ForwardChain))
			Expect(referencedChain).To(Equal(instanceChain))
		})

		It("deletes the filter instance chain", func() {
			Expect(creator.Destroy(logger, "some-id")).To(Succeed())

			Expect(fakeIPTablesDriver.DeleteChainCallCount()).NotTo(BeZero())
			table, chain := fakeIPTablesDriver.DeleteChainArgsForCall(1)
			Expect(table).To(Equal("filter"))
			instanceChain := iptables.InstanceChain(iptablesConfig, "some-id")
			Expect(chain).To(Equal(instanceChain))
		})

		It("flushes the logging instance chain", func() {
			Expect(creator.Destroy(logger, "some-id")).To(Succeed())

			Expect(fakeIPTablesDriver.FlushChainCallCount()).NotTo(BeZero())
			table, chain := fakeIPTablesDriver.FlushChainArgsForCall(2)
			Expect(table).To(Equal("filter"))
			instanceChain := iptables.InstanceChain(iptablesConfig, "some-id") + "-log"
			Expect(chain).To(Equal(instanceChain))

		})

		It("deletes the loggin instance chain", func() {
			Expect(creator.Destroy(logger, "some-id")).To(Succeed())

			Expect(fakeIPTablesDriver.DeleteChainCallCount()).NotTo(BeZero())
			table, chain := fakeIPTablesDriver.DeleteChainArgsForCall(2)
			Expect(table).To(Equal("filter"))
			instanceChain := iptables.InstanceChain(iptablesConfig, "some-id") + "-log"
			Expect(chain).To(Equal(instanceChain))
		})
	})
})
