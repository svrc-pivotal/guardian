package gqt_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/guardian/gqt/runner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Surviving Restarts", func() {
	var (
		args   []string
		client *runner.RunningGarden
	)

	BeforeEach(func() {
		args = []string{}
	})

	JustBeforeEach(func() {
		client = startGarden(args...)
	})

	AfterEach(func() {
		Expect(client.DestroyAndStop()).To(Succeed())
	})

	const (
		subnetName string = "177-100-10-0"
	)

	Context("when a container is created and then garden is restarted", func() {
		var (
			container        garden.Container
			hostNetInPort    uint32
			externalIP       string
			interfacePrefix  string
			propertiesDir    string
			existingProc     garden.Process
			containerSpec    garden.ContainerSpec
			restartArgs      []string
			gracefulShutdown bool
		)

		BeforeEach(func() {
			var err error
			propertiesDir, err = ioutil.TempDir("", "props")
			Expect(err).NotTo(HaveOccurred())
			args = append(args, "--properties-path", path.Join(propertiesDir, "props.json"))

			containerSpec = garden.ContainerSpec{
				Network: "177.100.10.30/30",
			}

			restartArgs = []string{}
			gracefulShutdown = true
		})

		JustBeforeEach(func() {
			var err error
			container, err = client.Create(containerSpec)
			Expect(err).NotTo(HaveOccurred())

			hostNetInPort, _, err = container.NetIn(hostNetInPort, 8080)
			Expect(err).NotTo(HaveOccurred())

			container.NetOut(garden.NetOutRule{
				Networks: []garden.IPRange{
					garden.IPRangeFromIP(net.ParseIP("8.8.8.8")),
				},
			})

			info, err := container.Info()
			Expect(err).NotTo(HaveOccurred())
			externalIP = info.ExternalIP
			interfacePrefix = info.Properties["kawasaki.iptable-prefix"]

			out := gbytes.NewBuffer()
			existingProc, err = container.Run(
				garden.ProcessSpec{
					Path: "/bin/sh",
					Args: []string{"-c", "while true; do echo hello; sleep 1; done;"},
				},
				garden.ProcessIO{
					Stdout: io.MultiWriter(GinkgoWriter, out),
					Stderr: io.MultiWriter(GinkgoWriter, out),
				})
			Expect(err).NotTo(HaveOccurred())

			if gracefulShutdown {
				Expect(client.Stop()).To(Succeed())
			} else {
				Expect(client.Kill()).To(MatchError("exit status 137"))
			}

			if len(restartArgs) == 0 {
				restartArgs = args
			}
			client = startGarden(restartArgs...)
		})

		AfterEach(func() {
			Expect(os.RemoveAll(propertiesDir)).To(Succeed())
		})

		Context("when the destroy-containers-on-startup flag is passed", func() {
			BeforeEach(func() {
				args = append(args, "--destroy-containers-on-startup")
			})

			It("destroys the remaining containers in the depotDir", func() {
				Expect(ioutil.ReadDir(client.DepotDir)).To(BeEmpty())
			})

			It("destroys the remaining containers' iptables", func() {
				out, err := exec.Command("iptables", "-w", "-S", "-t", "filter").CombinedOutput()
				Expect(err).NotTo(HaveOccurred())
				Expect(string(out)).NotTo(MatchRegexp(fmt.Sprintf("%sinstance.*", interfacePrefix)))
			})

			It("destroys the remaining containers' bridges", func() {
				out, err := exec.Command("ifconfig").CombinedOutput()
				Expect(err).NotTo(HaveOccurred())

				pattern := fmt.Sprintf(".*w%d%s.*", GinkgoParallelNode(), subnetName)
				Expect(string(out)).NotTo(MatchRegexp(pattern))
			})

			It("kills the container processes", func() {
				processes, err := exec.Command("ps", "aux").CombinedOutput()
				Expect(err).NotTo(HaveOccurred())

				Expect(string(processes)).NotTo(ContainSubstring(fmt.Sprintf("run runc /tmp/test-garden-%d/containers/%s", GinkgoParallelNode(), container.Handle())))
			})

			Context("when the garden server does not shut down gracefully", func() {
				BeforeEach(func() {
					gracefulShutdown = false
				})

				It("destroys orphaned containers' iptables filter rules", func() {
					out, err := exec.Command("iptables", "-w", "-S", "-t", "filter").CombinedOutput()
					Expect(err).NotTo(HaveOccurred())
					Expect(string(out)).NotTo(MatchRegexp(fmt.Sprintf("%sinstance.*", interfacePrefix)))
				})

				It("destroys orphaned containers' iptables nat rules", func() {
					out, err := exec.Command("iptables", "-w", "-S", "-t", "nat").CombinedOutput()
					Expect(err).NotTo(HaveOccurred())
					Expect(string(out)).NotTo(MatchRegexp(fmt.Sprintf("%sinstance.*", interfacePrefix)))
				})
			})

			Context("when a container is created after restart", func() {
				It("can be created with the same network reservation", func() {
					_, err := client.Create(containerSpec)
					Expect(err).NotTo(HaveOccurred())
				})
			})

		})

		Context("when the destroy-containers-on-startup flag is not passed", func() {
			FContext("when there are a lot of iptables rules", func() {
				var chainName string

				BeforeEach(func() {
					chainName = "w--instance-test-chain"

					cmd := exec.Command("iptables", "-w", "-N", chainName)
					sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Eventually(sess).Should(gexec.Exit(0))

					basePortNum := 5000
					for j := 0; j < 4000; j++ {
						portNumStr := strconv.FormatInt(int64(basePortNum+j), 10)

						cmd := exec.Command(
							"iptables", "-w", "-A", chainName, "-p", "tcp",
							"--dport", portNumStr, "-m", "iprange",
							"--dst-range", "10.10.38.8-10.10.38.8", "-m", "tcp",
							"-j", "RETURN",
						)
						sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						Eventually(sess).Should(gexec.Exit(0))
					}

					time.Sleep(time.Hour)
				})

				AfterEach(func() {
					// Expect(exec.Command("iptables", "-F", chainName).Run()).To(Succeed())
					// Expect(exec.Command("iptables", "-X", chainName).Run()).To(Succeed())
				})

				It("should still restart cleanly and handle API operations", func() {
					_, err := client.Create(containerSpec)
					Expect(err).NotTo(HaveOccurred())
				})
			})
			Describe("on th pre-existing VM", func() {
				It("does not destroy the depot", func() {
					Expect(filepath.Join(client.DepotDir, container.Handle())).To(BeADirectory())
				})

				It("can still run processes", func() {
					out := gbytes.NewBuffer()
					proc, err := container.Run(
						garden.ProcessSpec{
							Path: "/bin/sh",
							Args: []string{"-c", "echo hello; exit 12"},
						},
						garden.ProcessIO{
							Stdout: io.MultiWriter(GinkgoWriter, out),
							Stderr: io.MultiWriter(GinkgoWriter, out),
						})
					Expect(err).NotTo(HaveOccurred())
					exitCode, err := proc.Wait()
					Expect(err).NotTo(HaveOccurred())

					Expect(exitCode).To(Equal(12))
					Expect(out).To(gbytes.Say("hello"))
				})

				It("can reattach to processes that are still running", func() {
					out := gbytes.NewBuffer()
					procId := existingProc.ID()
					process, err := container.Attach(procId, garden.ProcessIO{
						Stdout: io.MultiWriter(GinkgoWriter, out),
						Stderr: io.MultiWriter(GinkgoWriter, out),
					})
					Expect(err).NotTo(HaveOccurred())
					Eventually(out).Should(gbytes.Say("hello"))

					Expect(process.Signal(garden.SignalKill)).To(Succeed())

					exited := make(chan struct{})
					go func() {
						process.Wait()
						close(exited)
					}()

					Eventually(exited).Should(BeClosed())
				})

				It("can still destroy the container", func() {
					Expect(client.Destroy(container.Handle())).To(Succeed())
				})

				It("can still be able to access the internet", func() {
					Expect(checkConnection(container, "8.8.8.8", 53)).To(Succeed())
				})

				It("can still be accessible from the outside", func() {
					Expect(listenInContainer(container, 8080)).To(Succeed())

					info, err := container.Info()
					Expect(err).NotTo(HaveOccurred())
					externalIP := info.ExternalIP

					// retry because listener process inside other container
					// may not start immediately
					Eventually(func() int {
						session := sendRequest(externalIP, hostNetInPort)
						return session.Wait().ExitCode()
					}).Should(Equal(0))
				})

				Context("when the server denies all the networks", func() {
					BeforeEach(func() {
						args = append(args, "--deny-network", "0.0.0.0/0")
					})

					It("still can't access disallowed IPs", func() {
						Expect(checkConnection(container, "8.8.4.4", 53)).NotTo(Succeed())
					})

					It("can still be able to access the allowed IPs", func() {
						Expect(checkConnection(container, "8.8.8.8", 53)).To(Succeed())
					})
				})

				Context("when the server is restarted without deny networks applied", func() {
					BeforeEach(func() {
						restartArgs = args[:]
						args = append(args, "--deny-network", "0.0.0.0/0")
					})

					It("is able to access the internet", func() {
						Expect(checkConnection(container, "8.8.8.8", 53)).To(Succeed())
						Expect(checkConnection(container, "8.8.4.4", 53)).To(Succeed())
					})
				})
			})

			Context("when creating a container after restart", func() {
				It("should not allocate ports used before restart", func() {
					secondContainer, err := client.Create(garden.ContainerSpec{})
					secondContainerHostPort, _, err := secondContainer.NetIn(0, 8080)
					Expect(err).NotTo(HaveOccurred())
					Expect(hostNetInPort).NotTo(Equal(secondContainerHostPort))
				})

				Context("with a subnet used before restart", func() {
					It("will not allocate an IP", func() {
						_, err := client.Create(containerSpec)
						Expect(err).To(MatchError("the requested IP is already allocated"))
					})
				})

				Context("with an IP used before restart", func() {
					BeforeEach(func() {
						containerSpec = garden.ContainerSpec{
							// Specifying a CIDR of < 30 will make garden give us exactly 177.100.10.5
							Network: "177.100.10.5/29",
						}
					})

					It("should not allocate the IP", func() {
						_, err := client.Create(containerSpec)
						Expect(err).To(MatchError("the requested IP is already allocated"))
					})
				})

				Context("with no network specified", func() {
					BeforeEach(func() {
						containerSpec = garden.ContainerSpec{}
					})

					It("successfully creates another container with no network specified", func() {
						_, err := client.Create(containerSpec)
						Expect(err).NotTo(HaveOccurred())
					})
				})
			})
		})
	})
})
