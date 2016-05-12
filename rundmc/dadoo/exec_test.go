package dadoo_test

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry-incubator/guardian/rundmc/dadoo"
	"github.com/cloudfoundry/gunk/command_runner/fake_command_runner"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Exec", func() {
	var (
		fakeCommandRunner *fake_command_runner.FakeCommandRunner
		bundlePath        string
	)

	BeforeEach(func() {
		fakeCommandRunner = fake_command_runner.New()

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

				fmt.Println("About to write to stdout")
				so.WriteString("hello")

				fmt.Println("About to copy io in stderr")
				n, err := io.Copy(se, si)
				fmt.Printf("copied %d bytes: %s", n, err)
				Expect(err).NotTo(HaveOccurred())
			}(cmd)

			return nil
		})
	})

	It("executes dadoo with the correct arguments", func() {
		doo := dadoo.NewTracker(fakeCommandRunner)
		doo.Exec(bundlePath, "id", "the-process-id", bytes.NewBufferString("blah"), garden.ProcessIO{}, garden.TTYSpec{})

		Expect(fakeCommandRunner.StartedCommands()[0].Path).To(Equal("dadoo"))
		Expect(fakeCommandRunner.StartedCommands()[0].Args).To(ConsistOf(
			"dadoo",
			"-stdout", filepath.Join(bundlePath, "processes/the-process-id.stdout"),
			"-stderr", filepath.Join(bundlePath, "processes/the-process-id.stderr"),
			"exec",
			"runc",
			bundlePath,
			"id",
		))
	})

	It("passes the process.json to STDIN", func() {
		doo := dadoo.NewTracker(fakeCommandRunner)
		doo.Exec(bundlePath, "id", "process-id", bytes.NewBufferString("json"), garden.ProcessIO{}, garden.TTYSpec{})

		Expect(ioutil.ReadAll(fakeCommandRunner.StartedCommands()[0].Stdin)).To(Equal([]byte("json")))
	})

	Context("when dadoo writes data to the named pipe passed as -stdout", func() {
		It("is written to the passed io.Stdout writer", func() {
			stdout := gbytes.NewBuffer()
			doo := dadoo.NewTracker(fakeCommandRunner)

			doo.Exec(bundlePath, "id", "process-id", bytes.NewBufferString("json"), garden.ProcessIO{Stdout: stdout}, garden.TTYSpec{})
			Eventually(stdout).Should(gbytes.Say("hello"))
		})
	})

	Context("when dadoo writes data from the named pipe at -stdin to the named pipe at -stdout", func() {
		FIt("the data is passed from and to the processio.Stdin/err as appropriate", func() {
			stdin := bytes.NewBufferString("I am stdin")
			stderr := gbytes.NewBuffer()

			doo := dadoo.NewTracker(fakeCommandRunner)
			doo.Exec(bundlePath, "id", "process-id", bytes.NewBufferString("json"), garden.ProcessIO{Stderr: stderr, Stdout: GinkgoWriter, Stdin: stdin}, garden.TTYSpec{})

			Eventually(stderr, "5s").Should(gbytes.Say("I am stdin"))
		})
	})

	PIt("all the panics..", func() {})
	PIt("waits on the started command to avoid zombies", func() {})
})
