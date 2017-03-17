package gqt_test

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"unicode"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/guardian/gardener"
	"code.cloudfoundry.org/guardian/gqt/runner"
	"code.cloudfoundry.org/guardian/kawasaki/mtu"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-golang/localip"
)

var _ = Describe("Networking", func() {
	var (
		client    *runner.RunningGarden
		container garden.Container

		containerSpec    garden.ContainerSpec
		containerNetwork string
		args             []string

		exampleDotCom net.IP

		extraProperties garden.Properties
	)

	BeforeEach(func() {
		args = []string{}
		containerNetwork = fmt.Sprintf("192.168.%d.0/24", 12+GinkgoParallelNode())
		containerSpec = garden.ContainerSpec{}

		var ips []net.IP
		Eventually(func() error {
			var err error
			ips, err = net.LookupIP("www.example.com")
			return err
		}, "60s", "2s").Should(Succeed())

		exampleDotCom = ips[0]
	})

	JustBeforeEach(func() {
		var err error

		client = startGarden(args...)

		containerSpec.Network = containerNetwork
		containerSpec.Properties = extraProperties
		container, err = client.Create(containerSpec)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(client.DestroyAndStop()).To(Succeed())
	})

	It("should have a loopback interface", func() {
		buffer := gbytes.NewBuffer()
		proc, err := container.Run(
			garden.ProcessSpec{
				Path: "ifconfig",
				User: "root",
			}, garden.ProcessIO{Stdout: io.MultiWriter(GinkgoWriter, buffer), Stderr: GinkgoWriter},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(proc.Wait()).To(Equal(0))

		Expect(buffer).To(gbytes.Say("lo"))
	})

	It("should have a (dynamically assigned) IP address", func() {
		buffer := gbytes.NewBuffer()
		proc, err := container.Run(
			garden.ProcessSpec{
				Path: "ifconfig",
				User: "root",
			}, garden.ProcessIO{Stdout: io.MultiWriter(GinkgoWriter, buffer), Stderr: io.MultiWriter(GinkgoWriter, buffer)},
		)
		Expect(err).NotTo(HaveOccurred())

		exitCode, err := proc.Wait()
		Expect(err).NotTo(HaveOccurred())
		Expect(exitCode).To(Equal(0))

		Expect(buffer).To(gbytes.Say(ipAddress(containerNetwork, 2)))
	})

	Context("when default network pool is changed", func() {
		BeforeEach(func() {
			args = []string{"--network-pool", "10.253.0.0/29"}
			containerNetwork = ""
		})

		It("vends IPs from the given network pool", func() {
			Expect(containerIP(container)).To(ContainSubstring("10.253."))
		})
	})

	It("should be pingable", func() {
		out, err := exec.Command("/bin/ping", "-c 2", ipAddress(containerNetwork, 2)).Output()
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(ContainSubstring(" 0% packet loss"))
	})

	Context("when container handle is longer than 49 chars", func() {
		var (
			longHandle          string = "too-looooong-haaaaaaaaaaaaaannnnnndddle-1234456787889"
			longHandleContainer garden.Container
		)

		JustBeforeEach(func() {
			var err error
			longHandleContainer, err = client.Create(garden.ContainerSpec{Handle: longHandle})
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			client.Destroy(longHandle)
		})

		It("should lookup container ip using last 49 chars of handle as hostname", func() {
			buff := gbytes.NewBuffer()
			p, err := longHandleContainer.Run(garden.ProcessSpec{
				Path: "cat",
				Args: []string{"/etc/hosts"},
			}, garden.ProcessIO{
				Stdout: buff,
				Stderr: buff,
			})
			Expect(err).NotTo(HaveOccurred())

			code, err := p.Wait()
			Expect(err).NotTo(HaveOccurred())
			Expect(code).To(Equal(0))

			hostsFile := string(buff.Contents())
			Expect(hostsFile).NotTo(ContainSubstring(longHandle))
			Expect(hostsFile).To(ContainSubstring(longHandle[len(longHandle)-49:]))
		})
	})

	Describe("a second container", func() {
		var otherContainer garden.Container

		JustBeforeEach(func() {
			var err error
			otherContainer, err = client.Create(garden.ContainerSpec{
				Network: containerNetwork,
			})

			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			Expect(client.Destroy(otherContainer.Handle())).To(Succeed())
		})

		It("should have the next IP address", func() {
			buffer := gbytes.NewBuffer()
			proc, err := otherContainer.Run(
				garden.ProcessSpec{
					Path: "ifconfig",
					User: "root",
				}, garden.ProcessIO{Stdout: buffer},
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(proc.Wait()).To(Equal(0))

			Expect(buffer).To(gbytes.Say(ipAddress(containerNetwork, 3)))
		})

		It("should be pingable", func() {
			out, err := exec.Command("/bin/ping", "-c 2", ipAddress(containerNetwork, 3)).Output()
			Expect(out).To(ContainSubstring(" 0% packet loss"))
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("the first container", func() {
			It("should still be pingable", func() {
				out, err := exec.Command("/bin/ping", "-c 2", ipAddress(containerNetwork, 2)).Output()
				Expect(out).To(ContainSubstring(" 0% packet loss"))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		It("should access internet", func() {
			Expect(checkConnection(otherContainer, exampleDotCom.String(), 80)).To(Succeed())
		})
	})

	Context("when it is recreated", func() {
		var contIP string

		JustBeforeEach(func() {
			var err error

			contIP = containerIP(container)

			Expect(client.Destroy(container.Handle())).To(Succeed())

			container, err = client.Create(garden.ContainerSpec{
				Network: containerNetwork,
			})
			Expect(err).ToNot(HaveOccurred())
		})

		It("reuses IP addresses", func() {
			newIpAddress := containerIP(container)
			Expect(newIpAddress).To(Equal(contIP))
		})

		It("is accessible from the outside", func() {
			hostPort, containerPort, err := container.NetIn(0, 4321)
			Expect(err).ToNot(HaveOccurred())

			Expect(listenInContainer(container, containerPort)).To(Succeed())

			externalIP := externalIP(container)

			// retry because listener process inside other container
			// may not start immediately
			Eventually(func() int {
				session := sendRequest(externalIP, hostPort)
				return session.Wait().ExitCode()
			}).Should(Equal(0))
		})
	})

	Describe("NetIn", func() {
		It("maps the provided host port to the container port", func() {
			const (
				hostPort      uint32 = 9888
				containerPort uint32 = 9080
			)

			actualHostPort, actualContainerPort, err := container.NetIn(hostPort, containerPort)
			Expect(err).ToNot(HaveOccurred())

			Expect(actualHostPort).To(Equal(hostPort))
			Expect(actualContainerPort).To(Equal(containerPort))
			Expect(listenInContainer(container, containerPort)).To(Succeed())

			externalIP := externalIP(container)

			Eventually(func() *gexec.Session { return sendRequest(externalIP, hostPort).Wait() }).
				Should(gbytes.Say(fmt.Sprintf("%d", containerPort)))
		})

		It("maps the random host port to a container port", func() {
			actualHostPort, actualContainerPort, err := container.NetIn(0, 0)
			Expect(err).ToNot(HaveOccurred())

			Expect(actualHostPort).NotTo(Equal(0))
			Expect(actualContainerPort).NotTo(Equal(0))
			Expect(listenInContainer(container, actualContainerPort)).To(Succeed())

			externalIP := externalIP(container)

			Eventually(func() *gexec.Session { return sendRequest(externalIP, actualHostPort).Wait() }).
				Should(gbytes.Say(fmt.Sprintf("%d", actualContainerPort)))
		})
	})

	Describe("--deny-network flag", func() {
		BeforeEach(func() {
			args = append(args, "--deny-network", "8.8.8.0/24")
		})

		It("should deny outbound traffic to IPs in the range", func() {
			Expect(checkConnection(container, "8.8.8.8", 53)).To(MatchError("Request failed. Process exited with code 1"))
		})

		It("should allow outbound traffic to IPs outside of the range", func() {
			Expect(checkConnection(container, "8.8.4.4", 53)).To(Succeed())
		})

		Context("when multiple --deny-networks are passed", func() {
			BeforeEach(func() {
				args = append(args, "--deny-network", "8.8.4.0/24")
			})

			It("should deny IPs in either range", func() {
				Expect(checkConnection(container, "8.8.8.8", 53)).To(MatchError("Request failed. Process exited with code 1"))
				Expect(checkConnection(container, "8.8.4.4", 53)).To(MatchError("Request failed. Process exited with code 1"))
			})
		})
	})

	Describe("NetOut", func() {
		var (
			rule garden.NetOutRule
		)

		BeforeEach(func() {
			rule = garden.NetOutRule{
				Protocol: garden.ProtocolTCP,
				Networks: []garden.IPRange{garden.IPRangeFromIP(net.ParseIP("8.8.8.8"))},
				Ports:    []garden.PortRange{garden.PortRangeFromPort(53)},
			}
		})

		Context("when an IP within the denied network range is permitted", func() {
			BeforeEach(func() {
				args = append(args, "--deny-network", "0.0.0.0/0")
			})

			JustBeforeEach(func() {
				Expect(checkConnection(container, "8.8.8.8", 53)).To(MatchError("Request failed. Process exited with code 1"))
			})

			It("should access internet", func() {
				Expect(container.NetOut(rule)).To(Succeed())
				Expect(checkConnection(container, "8.8.8.8", 53)).To(Succeed())
			})

			Context("when the dropped packets should get logged", func() {
				BeforeEach(func() {
					rule.Log = true
				})

				It("should access internet", func() {
					Expect(container.NetOut(rule)).To(Succeed())
					Expect(checkConnection(container, "8.8.8.8", 53)).To(Succeed())
				})
			})
		})
	})

	Describe("BulkNetOut", func() {
		var (
			rule1 garden.NetOutRule
			rule2 garden.NetOutRule
		)

		BeforeEach(func() {
			rule1 = garden.NetOutRule{
				Protocol: garden.ProtocolTCP,
				Networks: []garden.IPRange{garden.IPRangeFromIP(net.ParseIP("8.8.8.8"))},
				Ports:    []garden.PortRange{garden.PortRangeFromPort(53)},
			}
			rule2 = garden.NetOutRule{
				Protocol: garden.ProtocolTCP,
				Networks: []garden.IPRange{garden.IPRangeFromIP(net.ParseIP("8.8.4.4"))},
				Ports:    []garden.PortRange{garden.PortRangeFromPort(53)},
			}
		})

		Context("when an IP within the denied network range is permitted", func() {
			BeforeEach(func() {
				args = append(args, "--deny-network", "0.0.0.0/0")
			})

			JustBeforeEach(func() {
				Expect(checkConnection(container, "8.8.8.8", 53)).To(MatchError("Request failed. Process exited with code 1"))
				Expect(checkConnection(container, "8.8.4.4", 53)).To(MatchError("Request failed. Process exited with code 1"))
			})

			It("should access internet", func() {
				Expect(container.BulkNetOut([]garden.NetOutRule{rule1, rule2})).To(Succeed())

				Expect(checkConnection(container, "8.8.8.8", 53)).To(Succeed())
				Expect(checkConnection(container, "8.8.4.4", 53)).To(Succeed())
			})

			Context("when the dropped packets should get logged", func() {
				BeforeEach(func() {
					rule1.Log = true
					rule2.Log = true
				})

				It("should access internet", func() {
					Expect(container.BulkNetOut([]garden.NetOutRule{rule1, rule2})).To(Succeed())
					Expect(checkConnection(container, "8.8.8.8", 53)).To(Succeed())
					Expect(checkConnection(container, "8.8.4.4", 53)).To(Succeed())
				})
			})

			Context("when the iptables-restore-bin returns non zero", func() {
				BeforeEach(func() {
					args = append(args, "--iptables-restore-bin", "/bin/false")
				})

				It("should fail on BulkNetOut", func() {
					Expect(container.BulkNetOut([]garden.NetOutRule{rule1, rule2})).To(MatchError("iptables: bulk-prepend-rules: "))
				})
			})
		})
	})

	Context("when a network plugin path is provided at startup", func() {
		var argsFile string
		var stdinFile string
		var pluginReturn string

		BeforeEach(func() {
			tmpDir, err := ioutil.TempDir("", "netplugtest")
			Expect(err).NotTo(HaveOccurred())

			argsFile = path.Join(tmpDir, "args.log")
			stdinFile = path.Join(tmpDir, "stdin.log")
			args = append(args, "--network-plugin-extra-arg", pluginReturn)

			args = []string{
				"--network-plugin", testNetPluginBin,
				"--network-plugin-extra-arg", argsFile,
				"--network-plugin-extra-arg", stdinFile,
			}
		})

		Context("and the plugin is essentially a noop", func() {
			BeforeEach(func() {
				args = []string{"--network-plugin", "/bin/true"}
			})

			It("successfully creates a container", func() {
				_, err := client.Create(garden.ContainerSpec{})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the network plugin returns properties and dns_servers", func() {
			BeforeEach(func() {
				pluginReturn = `{
					"properties":{
						"foo":"bar",
						"kawasaki.mtu":"1499",
						"garden.network.container-ip":"10.255.10.10",
						"garden.network.host-ip":"255.255.255.255"
					},
					"dns_servers": [
						"1.2.3.4",
						"1.2.3.5"
					]
			  }`
				args = append(args, "--network-plugin-extra-arg", pluginReturn)
				extraProperties = garden.Properties{
					"some-property-on-the-spec": "some-value",
					"network.some-key":          "some-value",
					"network.some-other-key":    "some-other-value",
					"some-other-key":            "do-not-propagate",
					"garden.whatever":           "do-not-propagate",
					"kawasaki.nope":             "do-not-propagate",
				}
			})

			Context("when the container spec has properties that start with 'network.'", func() {
				var expectedJSON string

				BeforeEach(func() {
					expectedJSON = `"some-key":"some-value","some-other-key":"some-other-value"}`
				})

				It("propagates those properties as JSON to the network plugin up action", func() {
					Eventually(getContent(stdinFile)).Should(ContainSubstring(expectedJSON))
				})
			})

			It("sets the nameserver entries in the container's /etc/resolv.conf to the values supplied by the network plugin", func() {
				Expect(getNameservers(container)).To(Equal([]string{"1.2.3.4", "1.2.3.5"}))
			})

			It("executes the network plugin during container destroy", func() {
				containerHandle := container.Handle()

				Expect(client.Destroy(containerHandle)).To(Succeed())
				Expect(argsFile).To(BeAnExistingFile())

				Eventually(getContent(argsFile)).Should(ContainSubstring(fmt.Sprintf("%s %s", argsFile, stdinFile)))
				Eventually(getContent(argsFile)).Should(ContainSubstring(fmt.Sprintf("--action down --handle %s", containerHandle)))
			})

			It("passes the container pid to plugin's stdin", func() {
				Eventually(getContent(stdinFile)).Should(
					MatchRegexp(`.*{"Pid":[0-9]+.*}.*`),
				)
			})

			It("executes the network plugin during container creation", func() {
				containerHandle := container.Handle()

				Eventually(getContent(argsFile)).Should(
					ContainSubstring(
						fmt.Sprintf("%s %s %s --action up --handle %s", argsFile, stdinFile, pluginReturn, containerHandle),
					),
				)
			})

			Context("and the containerSpec contains NetOutRules", func() {
				BeforeEach(func() {
					containerSpec.NetOut = []garden.NetOutRule{
						garden.NetOutRule{
							Protocol: garden.ProtocolTCP,
							Networks: []garden.IPRange{garden.IPRangeFromIP(net.ParseIP("8.8.8.8"))},
							Ports:    []garden.PortRange{garden.PortRangeFromPort(53)},
						},
						garden.NetOutRule{
							Protocol: garden.ProtocolTCP,
							Networks: []garden.IPRange{garden.IPRangeFromIP(net.ParseIP("8.8.4.4"))},
							Ports:    []garden.PortRange{garden.PortRangeFromPort(53)},
						},
					}
				})

				It("passes the NetOut rules to the plugin during container creation", func() {
					jsonBytes, err := json.Marshal(containerSpec.NetOut)
					Expect(err).NotTo(HaveOccurred())

					Eventually(getContent(stdinFile)).Should(
						ContainSubstring("\"netout_rules\":" + string(jsonBytes)),
					)
				})
			})

			Context("and the containerSpec contains NetIn", func() {
				BeforeEach(func() {
					containerSpec.NetIn = []garden.NetIn{
						garden.NetIn{
							HostPort:      9999,
							ContainerPort: 8080,
						},
						garden.NetIn{
							HostPort:      9989,
							ContainerPort: 8081,
						},
					}
				})

				It("passes the NetIn input to the plugin during container creation", func() {
					jsonBytes, err := json.Marshal(containerSpec.NetIn)
					Expect(err).NotTo(HaveOccurred())

					Eventually(getContent(stdinFile)).Should(
						ContainSubstring("\"netin\":" + string(jsonBytes)),
					)
				})
			})

			It("persists the returned properties to the container's properties", func() {
				info, err := container.Info()
				Expect(err).NotTo(HaveOccurred())

				containerProperties := info.Properties

				Expect(containerProperties["foo"]).To(Equal("bar"))
				Expect(containerProperties["garden.network.container-ip"]).To(Equal("10.255.10.10"))
				Expect(containerProperties["garden.network.host-ip"]).To(Equal("255.255.255.255"))
			})

			It("doesn't remove existing properties", func() {
				info, err := container.Info()
				Expect(err).NotTo(HaveOccurred())

				Expect(info.Properties).To(HaveKey("some-property-on-the-spec"))
			})

			It("sets the ExternalIP and ContainerIP fields on the container.Info()", func() {
				info, err := container.Info()
				Expect(err).NotTo(HaveOccurred())

				Expect(info.ExternalIP).NotTo(BeEmpty())
				Expect(info.ContainerIP).To(Equal("10.255.10.10"))
			})

			Context("when BulkNetOut is called", func() {
				It("passes down the bulk net out rules to the external networker", func() {
					rules := []garden.NetOutRule{
						garden.NetOutRule{
							Protocol: garden.ProtocolTCP,
						},
						garden.NetOutRule{
							Protocol: garden.ProtocolUDP,
						},
					}
					container.BulkNetOut(rules)

					Eventually(getContent(stdinFile)).Should(
						ContainSubstring(`{"container_ip":"10.255.10.10","netout_rules":[{"protocol":1},{"protocol":2}]}`),
					)
				})
			})

		})
	})

	Describe("MTU size", func() {
		AfterEach(func() {
			err := client.Destroy(container.Handle())
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when container mtu is specified by operator", func() {
			BeforeEach(func() {
				args = append(args, "--mtu", "6789")
			})

			Describe("container's network interface", func() {
				It("has the correct MTU size", func() {
					stdout := gbytes.NewBuffer()
					stderr := gbytes.NewBuffer()

					process, err := container.Run(garden.ProcessSpec{
						User: "alice",
						Path: "ifconfig",
						Args: []string{containerIfName(container)},
					}, garden.ProcessIO{
						Stdout: stdout,
						Stderr: stderr,
					})
					Expect(err).ToNot(HaveOccurred())
					rc, err := process.Wait()
					Expect(err).ToNot(HaveOccurred())
					Expect(rc).To(Equal(0))

					Expect(stdout.Contents()).To(ContainSubstring(" MTU:6789 "))
				})
			})

			Describe("hosts's network interface for a container", func() {
				It("has the correct MTU size", func() {
					out, err := exec.Command("ifconfig", hostIfName(container)).Output()
					Expect(err).ToNot(HaveOccurred())

					Expect(out).To(ContainSubstring(" MTU:6789 "))
				})
			})
		})

		Context("when container mtu is not specified by operator", func() {
			var outboundIfaceMtu int

			BeforeEach(func() {
				outboundIP, err := localip.LocalIP()
				Expect(err).ToNot(HaveOccurred())
				outboundIfaceMtu, err = mtu.MTU(outboundIP)
				Expect(err).ToNot(HaveOccurred())
			})

			Describe("container's network interface", func() {
				It("has the same MTU as the host outbound interface", func() {
					stdout := gbytes.NewBuffer()
					stderr := gbytes.NewBuffer()

					process, err := container.Run(garden.ProcessSpec{
						User: "alice",
						Path: "ifconfig",
						Args: []string{containerIfName(container)},
					}, garden.ProcessIO{
						Stdout: stdout,
						Stderr: stderr,
					})
					Expect(err).ToNot(HaveOccurred())
					rc, err := process.Wait()
					Expect(err).ToNot(HaveOccurred())
					Expect(rc).To(Equal(0))

					Expect(stdout.Contents()).To(ContainSubstring(fmt.Sprintf(" MTU:%d ", outboundIfaceMtu)))
				})
			})

			Describe("hosts's network interface for a container", func() {
				It("has the same MTU as the host outbound interface", func() {
					out, err := exec.Command("ifconfig", hostIfName(container)).Output()
					Expect(err).ToNot(HaveOccurred())

					Expect(out).To(ContainSubstring(fmt.Sprintf(" MTU:%d ", outboundIfaceMtu)))
				})
			})
		})
	})

	Describe("DNS servers", func() {
		var (
			hostNameservers []string
		)

		BeforeEach(func() {
			out, err := ioutil.ReadFile("/etc/resolv.conf")
			Expect(err).NotTo(HaveOccurred())
			hostNameservers = parseNameservers(string(out))
		})

		Context("when not provided with any DNS servers", func() {
			It("adds the host's non-127.0.0.0/24 DNS servers to the container's /etc/resolv.conf", func() {
				resolvConf := readResolvConf(container)

				for _, hostNameserver := range hostNameservers {
					Expect(resolvConf).To(ContainSubstring(hostNameserver))
					Expect(resolvConf).NotTo(ContainSubstring("127.0.0."))
				}
			})
		})

		Context("when --dns-server is provided", func() {
			BeforeEach(func() {
				args = []string{"--dns-server", "1.2.3.4"}
			})

			It("adds the IP address to the container's /etc/resolv.conf", func() {
				nameservers := getNameservers(container)
				Expect(nameservers).To(ContainElement("1.2.3.4"))
			})

			It("strips the host's DNS servers from the container's /etc/resolv.conf", func() {
				nameservers := getNameservers(container)

				for _, hostNameserver := range hostNameservers {
					Expect(nameservers).NotTo(ContainElement(hostNameserver))
				}
			})
		})

		Context("when --additional-dns-server is provided", func() {
			BeforeEach(func() {
				args = []string{"--additional-dns-server", "1.2.3.4"}
			})

			It("appends the IP address and the host's non-127.0.0.0/24 DNS servers to the container's /etc/resolv.conf", func() {
				resolvConf := readResolvConf(container)

				for _, hostNameserver := range hostNameservers {
					Expect(resolvConf).To(ContainSubstring(hostNameserver))
					Expect(resolvConf).NotTo(ContainSubstring("127.0.0."))
				}

				Expect(resolvConf).To(ContainSubstring("nameserver 1.2.3.4"))
			})
		})
	})

	Describe("comments added to iptables rules", func() {
		BeforeEach(func() {
			containerSpec.Handle = fmt.Sprintf("iptable-comment-handle-%d", GinkgoParallelNode())
		})

		Context("when creating a container", func() {
			Describe("filter table", func() {
				It("annotates rules with the container handle", func() {
					output, err := runIPTables("-t", "filter", "-L")
					Expect(err).NotTo(HaveOccurred())
					Expect(string(output)).To(ContainSubstring(fmt.Sprintf(`/* %s */`, containerSpec.Handle)))
				})
			})

			Describe("nat table", func() {
				It("annotates rules with the container handle", func() {
					output, err := runIPTables("-t", "nat", "-L")
					Expect(err).NotTo(HaveOccurred())
					Expect(string(output)).To(ContainSubstring(fmt.Sprintf(`/* %s */`, containerSpec.Handle)))
				})
			})
		})

		Context("when adding a netin rule to a container", func() {
			JustBeforeEach(func() {
				_, _, err := container.NetIn(0, 0)
				Expect(err).NotTo(HaveOccurred())
			})

			It("annotates the rule with the container handle", func() {
				output, err := runIPTables("-t", "nat", "-L")
				Expect(err).NotTo(HaveOccurred())
				Expect(string(output)).To(MatchRegexp(fmt.Sprintf(`DNAT.*/\* %s \*/`, containerSpec.Handle)))
			})
		})

		Context("when adding a netout rule to a container", func() {
			JustBeforeEach(func() {
				Expect(container.NetOut(garden.NetOutRule{
					Protocol: garden.ProtocolTCP,
					Networks: []garden.IPRange{garden.IPRangeFromIP(net.ParseIP("8.8.8.8"))},
					Ports:    []garden.PortRange{garden.PortRangeFromPort(53)},
				})).To(Succeed())
			})

			It("annotates the rule with the container handle", func() {
				output, err := runIPTables("-t", "filter", "-L")
				Expect(err).NotTo(HaveOccurred())
				Expect(string(output)).To(ContainSubstring(fmt.Sprintf(`/* %s */`, containerSpec.Handle)))
			})
		})
	})
})

var _ = Describe("IPTables Binary Flags", func() {
	var (
		client *runner.RunningGarden
		args   []string
	)

	BeforeEach(func() {
		args = []string{}
	})

	JustBeforeEach(func() {
		client = startGarden(args...)
	})

	Describe("--iptables-bin flag", func() {
		Context("when the path is valid", func() {
			BeforeEach(func() {
				args = append(args, "--iptables-bin", "/sbin/iptables")
			})

			AfterEach(func() {
				Expect(client.DestroyAndStop()).To(Succeed())
			})

			It("should succeed to start the server", func() {
				Expect(client.Ping()).To(Succeed())
			})
		})

		Context("when the path is invalid", func() {
			BeforeEach(func() {
				args = append(args, "--iptables-bin", "/path/to/iptables/bin")
			})

			It("should fail to start the server", func() {
				Expect(client.Ping()).To(HaveOccurred())
			})
		})

		Context("when the path is valid but it's not iptables", func() {
			BeforeEach(func() {
				args = append(args, "--iptables-bin", "/bin/ls")
			})

			It("should fail to start the server", func() {
				Expect(client.Ping()).To(HaveOccurred())
			})
		})
	})
})

func externalIP(container garden.Container) string {
	properties, err := container.Properties()
	Expect(err).NotTo(HaveOccurred())
	return properties[gardener.ExternalIPKey]
}

func containerIP(container garden.Container) string {
	properties, err := container.Properties()
	Expect(err).NotTo(HaveOccurred())
	return properties[gardener.ContainerIPKey]
}

func checkConnection(container garden.Container, ip string, port int) error {
	process, err := container.Run(garden.ProcessSpec{
		User: "alice",
		Path: "sh",
		Args: []string{"-c", fmt.Sprintf("echo hello | nc -w5 %s %d", ip, port)},
	}, garden.ProcessIO{Stdout: GinkgoWriter, Stderr: GinkgoWriter})
	if err != nil {
		return err
	}

	exitCode, err := process.Wait()
	if err != nil {
		return err
	}

	if exitCode == 0 {
		return nil
	} else {
		return fmt.Errorf("Request failed. Process exited with code %d", exitCode)
	}
}

func ipAddress(subnet string, index int) string {
	ip := strings.Split(subnet, "/")[0]
	pattern := regexp.MustCompile(".[0-9]+$")
	ip = pattern.ReplaceAllString(ip, fmt.Sprintf(".%d", index))
	return ip
}

func listenInContainer(container garden.Container, containerPort uint32) error {
	_, err := container.Run(garden.ProcessSpec{
		User: "alice",
		Path: "sh",
		Args: []string{"-c", fmt.Sprintf("echo %d | nc -l -p %d", containerPort, containerPort)},
	}, garden.ProcessIO{
		Stdout: GinkgoWriter,
		Stderr: GinkgoWriter,
	})
	Expect(err).ToNot(HaveOccurred())

	return err
}

func sendRequest(ip string, port uint32) *gexec.Session {
	sess, err := gexec.Start(exec.Command("nc", "-w5", "-v", ip, fmt.Sprintf("%d", port)), GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	return sess
}

func getContent(filename string) func() []byte {
	return func() []byte {
		bytes, err := ioutil.ReadFile(filename)
		Expect(err).NotTo(HaveOccurred())
		return bytes
	}
}

func containerIfName(container garden.Container) string {
	properties, err := container.Properties()
	Expect(err).NotTo(HaveOccurred())
	return properties["kawasaki.container-interface"]
}

func hostIfName(container garden.Container) string {
	properties, err := container.Properties()
	Expect(err).NotTo(HaveOccurred())
	return properties["kawasaki.host-interface"]
}

func getFlagValue(contentFile, flagName string) func() []byte {
	re := regexp.MustCompile(fmt.Sprintf("%s (.*)", flagName))
	return func() []byte {
		content := getContent(contentFile)()
		matches := re.FindSubmatch(content)
		Expect(matches).To(HaveLen(2))
		return matches[1]
	}
}

func readResolvConf(container garden.Container) string {
	stdout := gbytes.NewBuffer()

	process, err := container.Run(garden.ProcessSpec{
		Path: "cat",
		Args: []string{"/etc/resolv.conf"},
	}, garden.ProcessIO{
		Stdout: stdout,
	})
	Expect(err).ToNot(HaveOccurred())

	exitCode, err := process.Wait()
	Expect(err).ToNot(HaveOccurred())
	Expect(exitCode).To(Equal(0))
	return string(stdout.Contents())
}

func getNameservers(container garden.Container) []string {
	contents := readResolvConf(container)
	return parseNameservers(string(contents))
}

func parseNameservers(resolvConfContents string) []string {
	notAnIpSymbol := func(c rune) bool {
		return !(unicode.IsNumber(c) || c == '.')
	}
	return strings.FieldsFunc(resolvConfContents, notAnIpSymbol)
}
