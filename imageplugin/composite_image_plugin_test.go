package imageplugin_test

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

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
			createEnvs   []string
			createErr    error
		)

		BeforeEach(func() {
			cmd = exec.Command("unpriv-plugin", "create")
			fakeUnprivilegedCommandCreator.CreateCommandReturns(cmd, nil)

			handle = "test-handle"
			rootfsProviderSpec = rootfs_provider.Spec{}

			fakeImagePluginStdout = "/image-rootfs/\n"
			fakeImagePluginError = nil

			createRootfs = ""
			createEnvs = []string{}
			createErr = nil
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
			createRootfs, createEnvs, createErr = imagePlugin.Create(fakeLogger, handle, rootfsProviderSpec)
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

		Context("when the image.json is not defined", func() {
			It("returns an empty list of env vars", func() {
				Expect(createEnvs).To(BeEmpty())
			})
		})

		Context("when there is an image.json defined", func() {
			var imagePath string

			BeforeEach(func() {
				var err error
				imagePath, err = ioutil.TempDir("", "")
				Expect(err).NotTo(HaveOccurred())

				fakeImagePluginStdout = imagePath

				customImageJsonFile, err := os.Create(filepath.Join(imagePath, "image.json"))
				Expect(err).NotTo(HaveOccurred())
				imageJson := imageplugin.Image{
					Config: imageplugin.ImageConfig{
						Env: []string{
							"MY_VAR=set",
							"MY_SECOND_VAR=also_set",
						},
					},
				}
				Expect(json.NewEncoder(customImageJsonFile).Encode(imageJson)).To(Succeed())
				Expect(os.Chmod(customImageJsonFile.Name(), 0777)).To(Succeed())
			})

			AfterEach(func() {
				Expect(os.RemoveAll(imagePath)).To(Succeed())
			})

			It("returns the list of env variables to set", func() {
				Expect(createEnvs).To(ConsistOf([]string{"MY_VAR=set", "MY_SECOND_VAR=also_set"}))
			})

			Context("but it cannot be opened", func() {
				BeforeEach(func() {
					Expect(os.Chmod(filepath.Join(imagePath, "image.json"), 0000)).To(Succeed())
				})

				It("returns a wrapped error", func() {
					Expect(createErr).To(MatchError(ContainSubstring("reading image.json:")))
				})
			})

			Context("but it cannot be parsed", func() {
				BeforeEach(func() {
					err := ioutil.WriteFile(filepath.Join(imagePath, "image.json"), []byte("not-json"), 0777)
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns a wrapped error", func() {
					Expect(createErr).To(MatchError(ContainSubstring("reading image.json: parsing image config")))
				})
			})
		})
	})
})
