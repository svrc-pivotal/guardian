package dadoo_test

import (
	"github.com/cloudfoundry-incubator/guardian/rundmc/dadoo"
	"github.com/cloudfoundry-incubator/guardian/rundmc/dadoo/fakes"
	"github.com/cloudfoundry/gunk/command_runner/fake_command_runner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Process", func() {
	var (
		fakeCommandRunner *fake_command_runner.FakeCommandRunner

		fakeExitWaiter *fakes.FakeExitWaiter
		process        *dadoo.Process
	)

	BeforeEach(func() {
		fakeCommandRunner = fake_command_runner.New()

		fakeExitWaiter = new(fakes.FakeExitWaiter)
		process = dadoo.NewProcess(fakeExitWaiter, "process-id-1", "process.sock")
	})

	Describe("ID", func() {
		It("returns the process id", func() {
			Expect(process.ID()).To(Equal("process-id-1"))
		})
	})
})
