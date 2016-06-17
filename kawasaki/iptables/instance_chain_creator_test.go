package iptables_test

import (
	"errors"
	"fmt"
	"net"
	"os/exec"

	"github.com/cloudfoundry-incubator/guardian/kawasaki/iptables"
	"github.com/cloudfoundry/gunk/command_runner/fake_command_runner"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/cloudfoundry/gunk/command_runner/fake_command_runner/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = FDescribe("Create", func() {
	var (
		fakeRunner         *fake_command_runner.FakeCommandRunner
		creator            *iptables.InstanceChainCreator
		bridgeName         string
		ip                 net.IP
		network            *net.IPNet
		logger             lager.Logger
		handle             string
		iptablesController *iptables.IPTablesController
	)

	BeforeEach(func() {
		var err error

		fakeRunner = fake_command_runner.New()
		logger = lagertest.NewTestLogger("test")

		handle = "some-handle-that-is-longer-than-29-characters-long"
		bridgeName = "some-bridge"
		ip, network, err = net.ParseCIDR("1.2.3.4/28")
		Expect(err).NotTo(HaveOccurred())

		iptablesController = iptables.New("iptables", fakeRunner, "prefix-")
		creator = iptables.NewInstanceChainCreator(iptablesController)
	})

	Describe("Container Creation", func() {
		var specs []fake_command_runner.CommandSpec

		BeforeEach(func() {
			specs = []fake_command_runner.CommandSpec{
				{
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf("%s --wait --table nat -N prefix-instance-some-id", iptablesController.BinPath)},
				},
				{
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf("%s --wait --table nat -A prefix-prerouting --jump prefix-instance-some-id", iptablesController.BinPath)},
				},
				{
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf(
						`(%s --wait --table nat -S %s | grep "\-j MASQUERADE\b" | grep -q -F -- "-s %s") || %s --wait --table nat -A %s --source %s ! --destination %s --jump MASQUERADE`,
						iptablesController.BinPath, "prefix-postrouting", network.String(), iptablesController.BinPath,
						"prefix-postrouting", network.String(), network.String(),
					)},
				},
				{
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf("%s --wait --table filter -N prefix-instance-some-id", iptablesController.BinPath)},
				},
				{
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf("%s --wait -A prefix-instance-some-id -s %s -d %s -j ACCEPT", iptablesController.BinPath, network.String(), network.String())},
				},
				{
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf("%s --wait -A prefix-instance-some-id --goto prefix-default", iptablesController.BinPath)},
				},
				{
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf("%s --wait -I prefix-forward 2 --in-interface %s --source %s --goto prefix-instance-some-id", iptablesController.BinPath, bridgeName, ip.String())},
				},
				{
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf("%s --wait --table filter -N prefix-instance-some-id-log", iptablesController.BinPath)},
				},
				{
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf("%s --wait -A prefix-instance-some-id-log -m conntrack --ctstate NEW,UNTRACKED,INVALID --protocol tcp --jump LOG --log-prefix some-handle-that-is-longer-th", iptablesController.BinPath)},
				},
				{
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf("%s --wait -A prefix-instance-some-id-log --jump RETURN", iptablesController.BinPath)},
				},
			}
		})

		It("should set up the chain", func() {
			Expect(creator.Create(logger, handle, "some-id", bridgeName, ip, network)).To(Succeed())
			Expect(fakeRunner).To(HaveExecutedSerially(specs...))
		})

		DescribeTable("iptables failures",
			func(specIndex int, errorString string) {
				fakeRunner.WhenRunning(specs[specIndex], func(cmd *exec.Cmd) error {
					cmd.Stderr.Write([]byte("iptables failed"))
					return errors.New("Exit status blah")
				})

				Expect(creator.Create(logger, handle, "some-id", bridgeName, ip, network)).To(MatchError(errorString))
			},
			Entry("create nat instance chain", 0, "iptables create-instance-chains: iptables failed"),
			Entry("bind nat instance chain to nat prerouting chain", 1, "iptables create-instance-chains: iptables failed"),
			Entry("enable NAT for traffic coming from containers", 2, "iptables create-instance-chains: iptables failed"),
			Entry("create logging instance chain", 7, "iptables create-instance-chains: iptables failed"),
			Entry("append logging to instance chain", 8, "iptables create-instance-chains: iptables failed"),
			Entry("return from logging instance chain", 9, "iptables create-instance-chains: iptables failed"),
		)
	})

	Describe("ContainerTeardown", func() {
		var specs []fake_command_runner.CommandSpec

		BeforeEach(func() {
			specs = []fake_command_runner.CommandSpec{
				{
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf(
						`iptables --wait --table nat -S %s 2> /dev/null | grep "\-j %s\b" | sed -e "s/-A/-D/" | xargs --no-run-if-empty --max-lines=1 iptables --wait --table nat`,
						"prefix-prerouting", "prefix-instance-some-id",
					)},
				},
				{
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf(
						`iptables --wait --table nat -F %s 2> /dev/null || true`,
						"prefix-instance-some-id",
					)},
				},
				{
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf(
						`iptables --wait --table nat -X %s 2> /dev/null || true`,
						"prefix-instance-some-id",
					)},
				},
				{
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf(
						`iptables --wait -S %s 2> /dev/null | grep "\-g %s\b" | sed -e "s/-A/-D/" | xargs --no-run-if-empty --max-lines=1 iptables --wait`,
						"prefix-forward", "prefix-instance-some-id",
					)},
				},
				{
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf("iptables --wait --table filter -F %s 2> /dev/null || true", "prefix-instance-some-id")},
				},
				{
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf("iptables --wait --table filter -X %s 2> /dev/null || true", "prefix-instance-some-id")},
				},
				{
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf("iptables --wait --table filter -F %s 2> /dev/null || true", "prefix-instance-some-id-log")},
				},
				{
					Path: "sh",
					Args: []string{"-c", fmt.Sprintf("iptables --wait --table filter -X %s 2> /dev/null || true", "prefix-instance-some-id-log")},
				},
			}
		})

		It("should tear down the chain", func() {
			Expect(creator.Destroy(logger, "some-id")).To(Succeed())
			Expect(fakeRunner).To(HaveExecutedSerially(specs...))
		})

		Describe("iptables failure", func() {
			It("returns an error", func() {
				fakeRunner.WhenRunning(specs[0], func(cmd *exec.Cmd) error {
					cmd.Stderr.Write([]byte("iptables failed"))
					return errors.New("exit status foo")
				})

				Expect(creator.Destroy(logger, "some-id")).To(MatchError("iptables prune-prerouting-chain: iptables failed"))
			})
		})
	})
})
