package dadoo_test

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry-incubator/guardian/gardener/fakes"
	"github.com/cloudfoundry-incubator/guardian/rundmc/dadoo"
	"github.com/cloudfoundry/gunk/command_runner/fake_command_runner"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Exec", func() {
	var (
		fakeCommandRunner *fake_command_runner.FakeCommandRunner
		fakeLogger        *lagertest.TestLogger
		bundlePath        string
		pidGenerator      *fakes.FakeUidGenerator
		dadooExecer       *dadoo.DadooExecer
	)

	BeforeEach(func() {
		fakeCommandRunner = fake_command_runner.New()
		fakeLogger = lagertest.NewTestLogger("test")
		pidGenerator = new(fakes.FakeUidGenerator)
		dadooExecer = dadoo.NewExecer(fakeCommandRunner, pidGenerator)
		pidGenerator.GenerateReturns("the-process-id")

		var err error
		bundlePath, err = ioutil.TempDir("", "dadooexectracker")
		Expect(err).NotTo(HaveOccurred())

		// dadoo should open up its end of the named pipes
		fakeCommandRunner.WhenRunning(fake_command_runner.CommandSpec{
			Path: "dadoo",
		}, func(cmd *exec.Cmd) error {
			go func(cmd *exec.Cmd) {
				defer GinkgoRecover()

				fs := flag.NewFlagSet("something", flag.PanicOnError)
				stdin := fs.String("stdin", "", "")
				stdout := fs.String("stdout", "", "")
				stderr := fs.String("stderr", "", "")
				fs.Parse(cmd.Args[1:])

				si, err := os.Open(*stdin)
				Expect(err).NotTo(HaveOccurred())

				so, err := os.OpenFile(*stdout, os.O_APPEND|os.O_WRONLY, 0600)
				Expect(err).NotTo(HaveOccurred())

				se, err := os.OpenFile(*stderr, os.O_APPEND|os.O_WRONLY, 0600)
				Expect(err).NotTo(HaveOccurred())

				so.WriteString("hello")

				_, err = io.Copy(se, si)
				Expect(err).NotTo(HaveOccurred())
			}(cmd)

			return nil
		})
	})

	It("executes dadoo with the correct arguments", func() {
		dadooExecer.Exec(fakeLogger, bundlePath, "id", garden.ProcessSpec{}, garden.ProcessIO{})

		Expect(fakeCommandRunner.StartedCommands()[0].Path).To(Equal("dadoo"))
		Expect(fakeCommandRunner.StartedCommands()[0].Args).To(ConsistOf(
			"dadoo",
			"-stdin", filepath.Join(bundlePath, "processes/the-process-id.stdin"),
			"-stdout", filepath.Join(bundlePath, "processes/the-process-id.stdout"),
			"-stderr", filepath.Join(bundlePath, "processes/the-process-id.stderr"),
			"exec",
			"runc",
			bundlePath,
			"id",
		))
	})

	It("passes the process.json to STDIN", func() {
		dadooExecer.Exec(fakeLogger, bundlePath, "id", garden.ProcessSpec{Path: "ls"}, garden.ProcessIO{})

		Expect(ioutil.ReadAll(fakeCommandRunner.StartedCommands()[0].Stdin)).To(MatchJSON(`{"path":"ls","rlimits":{}}`))
	})

	Context("when dadoo writes data to the named pipe passed as -stdout", func() {
		It("is written to the passed io.Stdout writer", func() {
			stdout := gbytes.NewBuffer()

			dadooExecer.Exec(fakeLogger, bundlePath, "id",
				garden.ProcessSpec{Path: "ls"},
				garden.ProcessIO{
					Stdout: stdout,
				})
			Eventually(stdout).Should(gbytes.Say("hello"))
		})
	})

	Context("when dadoo writes data from the named pipe at -stdin to the named pipe at -stdout", func() {
		It("the data is passed from and to the processio.Stdin/err as appropriate", func() {
			stdin := bytes.NewBufferString("I am stdin")
			stderr := gbytes.NewBuffer()

			dadooExecer.Exec(fakeLogger, bundlePath, "id",
				garden.ProcessSpec{Path: "ls"},
				garden.ProcessIO{
					Stdin:  stdin,
					Stderr: stderr,
				})

			Eventually(stderr, "5s").Should(gbytes.Say("I am stdin"))
		})
	})

	FIt("waits on the started command to avoid zombies", func() {
		process, err := dadooExecer.Exec(fakeLogger, bundlePath, "id",
			garden.ProcessSpec{
				Path: "sh",
				Args: []string{"-c", "exit 42"},
			},
			garden.ProcessIO{})
		Expect(err).NotTo(HaveOccurred())

		exitStatus, err := process.Wait()
		Expect(err).NotTo(HaveOccurred())
		Expect(exitStatus).To(Equal(42))
	})

	PIt("all the panics..", func() {})
})
