package iptables_test

import (
	"errors"
	"os/exec"

	"github.com/cloudfoundry-incubator/guardian/kawasaki/iptables"
	"github.com/cloudfoundry/gunk/command_runner/fake_command_runner"
	. "github.com/cloudfoundry/gunk/command_runner/fake_command_runner/matchers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Runner", func() {
	var (
		fakeRunner *fake_command_runner.FakeCommandRunner
		runner     *iptables.IPTablesLoggingRunner
	)

	BeforeEach(func() {
		fakeRunner = fake_command_runner.New()
		runner = &iptables.IPTablesLoggingRunner{
			Runner: fakeRunner,
		}
	})

	It("runs a given command", func() {
		cmd := exec.Command("/bin/echo", "hello")
		Expect(runner.Run("action", cmd)).To(Succeed())

		Expect(fakeRunner).To(HaveExecutedSerially(
			fake_command_runner.CommandSpec{
				Path: "/bin/echo",
				Args: []string{"hello"},
			},
		))
	})

	Context("when the command fails", func() {
		BeforeEach(func() {
			fakeRunner.WhenRunning(fake_command_runner.CommandSpec{}, func(cmd *exec.Cmd) error {
				cmd.Stderr.Write([]byte("banana error"))
				return errors.New("failed")
			})
		})

		It("returns an error", func() {
			cmd := exec.Command("/bin/echo", "hello")
			Expect(runner.Run("action", cmd)).To(HaveOccurred())
		})

		It("forwads stderr contents in the error", func() {
			cmd := exec.Command("/bin/echo", "hello")
			Expect(runner.Run("action", cmd)).To(MatchError("iptables action: banana error"))
		})
	})
})
