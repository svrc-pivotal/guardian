package imageplugin_test

import (
	"errors"
	"os/exec"

	"code.cloudfoundry.org/garden-shed/rootfs_provider"
	"code.cloudfoundry.org/guardian/imageplugin"
	fakes "code.cloudfoundry.org/guardian/imageplugin/imagepluginfakes"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/cloudfoundry/gunk/command_runner/fake_command_runner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = FDescribe("CompositeImagePlugin", func() {

	var (
		imagePlugin *imageplugin.CompositeImagePlugin

		fakeUnprivilegedCommandCreator *fakes.FakeCommandCreator
		fakeCommandRunner              *fake_command_runner.FakeCommandRunner

		fakeLogger *lagertest.TestLogger
	)

	BeforeEach(func() {
		fakeUnprivilegedCommandCreator = new(fakes.FakeCommandCreator)
		fakeCommandRunner = fake_command_runner.New()

		fakeLogger = lagertest.NewTestLogger("composite-image-plugin")
	})

	JustBeforeEach(func() {
		imagePlugin = imageplugin.New(fakeUnprivilegedCommandCreator, fakeCommandRunner)
	})

	Describe("Create", func() {
		var (
			cmd *exec.Cmd

			handle             string
			rootfsProviderSpec rootfs_provider.Spec

			fakeImagePluginStdout string
			fakeImagePluginError  error

			createRootfs string
			createErr    error
		)

		BeforeEach(func() {
			cmd = exec.Command("unpriv-plugin", "create")
			fakeUnprivilegedCommandCreator.CreateCommandReturns(cmd, nil)

			handle = "test-handle"
			rootfsProviderSpec = rootfs_provider.Spec{}

			fakeImagePluginStdout = "/image-rootfs/\n"
			fakeImagePluginError = nil

			createErr = nil
			createRootfs = ""
		})

		JustBeforeEach(func() {
			fakeCommandRunner.WhenRunning(
				fake_command_runner.CommandSpec{
					Path: cmd.Path,
				},
				func(cmd *exec.Cmd) error {
					cmd.Stdout.Write([]byte(fakeImagePluginStdout))
					return fakeImagePluginError
				},
			)
			createRootfs, _, createErr = imagePlugin.Create(fakeLogger, handle, rootfsProviderSpec)
		})

		It("calls the plugin to generate a create command", func() {
			Expect(createErr).NotTo(HaveOccurred())
			Expect(fakeUnprivilegedCommandCreator.CreateCommandCallCount()).To(Equal(1))

			logArg, handleArg, specArg := fakeUnprivilegedCommandCreator.CreateCommandArgsForCall(0)
			Expect(logArg).To(Equal(fakeLogger))
			Expect(handleArg).To(Equal(handle))
			Expect(specArg).To(Equal(rootfsProviderSpec))
		})

		Context("when the plugin create-command creation fails", func() {
			BeforeEach(func() {
				fakeUnprivilegedCommandCreator.CreateCommandReturns(nil, errors.New("create-command-failed"))
			})

			It("returns a sensible error", func() {
				Expect(createErr).To(MatchError("creating image plugin create-command: create-command-failed"))
			})
		})

		It("runs the plugin command with the command runner", func() {
			Expect(createErr).NotTo(HaveOccurred())
			Expect(fakeCommandRunner.ExecutedCommands()).To(HaveLen(1))
			executedCmd := fakeCommandRunner.ExecutedCommands()[0]

			Expect(executedCmd).To(Equal(cmd))
		})

		Context("when running the image plugin create fails", func() {
			BeforeEach(func() {
				fakeImagePluginStdout = "image-plugin-exploded-due-to-oom"
				fakeImagePluginError = errors.New("image-plugin-create-failed")
			})

			It("returns the wrapped error and plugin stdout, with context", func() {
				Expect(createErr).To(MatchError("running image plugin create: image-plugin-exploded-due-to-oom: image-plugin-create-failed"))
			})
		})

		It("returns trimmed plugin stdout concatenated with 'rootfs'", func() {
			Expect(createRootfs).To(Equal("/image-rootfs/rootfs"))
		})
	})
})
