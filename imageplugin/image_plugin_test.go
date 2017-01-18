package imageplugin_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"code.cloudfoundry.org/garden-shed/rootfs_provider"
	"code.cloudfoundry.org/guardian/imageplugin"
	fakes "code.cloudfoundry.org/guardian/imageplugin/imagepluginfakes"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/gunk/command_runner/fake_command_runner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/st3v/glager"
)

var _ = Describe("ImagePlugin", func() {

	var (
		imagePlugin *imageplugin.ImagePlugin

		fakeUnprivilegedCommandCreator *fakes.FakeCommandCreator
		fakePrivilegedCommandCreator   *fakes.FakeCommandCreator
		fakeCommandRunner              *fake_command_runner.FakeCommandRunner

		fakeLogger lager.Logger

		defaultRootfs string
	)

	BeforeEach(func() {
		fakeUnprivilegedCommandCreator = new(fakes.FakeCommandCreator)
		fakePrivilegedCommandCreator = new(fakes.FakeCommandCreator)
		fakeCommandRunner = fake_command_runner.New()

		fakeLogger = glager.NewLogger("image-plugin")

		defaultRootfs = "/default-rootfs"
	})

	JustBeforeEach(func() {
		imagePlugin = imageplugin.New(fakeUnprivilegedCommandCreator, fakePrivilegedCommandCreator, fakeCommandRunner, defaultRootfs)
	})

	Describe("Create", func() {
		var (
			cmd *exec.Cmd

			handle             string
			rootfsProviderSpec rootfs_provider.Spec
			rootfs             string
			namespaced         bool

			fakeImagePluginStdout string
			fakeImagePluginStderr string
			fakeImagePluginError  error

			createRootfs string
			createEnvs   []string
			createErr    error
		)

		BeforeEach(func() {
			cmd = exec.Command("unpriv-plugin", "create")
			fakeUnprivilegedCommandCreator.CreateCommandReturns(cmd)
			fakePrivilegedCommandCreator.CreateCommandReturns(cmd)

			handle = "test-handle"
			rootfs = "docker:///busybox"
			namespaced = true //assume unprivileged by default

			fakeImagePluginStdout = "/image-rootfs/\n"
			fakeImagePluginStderr = ""
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
					cmd.Stderr.Write([]byte(fakeImagePluginStderr))
					return fakeImagePluginError
				},
			)

			rootfsURL, err := url.Parse(rootfs)
			Expect(err).NotTo(HaveOccurred())
			rootfsProviderSpec = rootfs_provider.Spec{RootFS: rootfsURL, Namespaced: namespaced}
			createRootfs, createEnvs, createErr = imagePlugin.Create(fakeLogger, handle, rootfsProviderSpec)
		})

		It("calls the unprivileged command creator to generate a create command", func() {
			Expect(createErr).NotTo(HaveOccurred())
			Expect(fakeUnprivilegedCommandCreator.CreateCommandCallCount()).To(Equal(1))
			Expect(fakePrivilegedCommandCreator.CreateCommandCallCount()).To(Equal(0))

			_, handleArg, specArg := fakeUnprivilegedCommandCreator.CreateCommandArgsForCall(0)
			Expect(handleArg).To(Equal(handle))
			Expect(specArg).To(Equal(rootfsProviderSpec))
		})

		Context("when destroying an unprivileged volume", func() {
			BeforeEach(func() {
				namespaced = false
			})

			It("calls the privileged command creator to generate a create command", func() {
				Expect(createErr).NotTo(HaveOccurred())
				Expect(fakePrivilegedCommandCreator.CreateCommandCallCount()).To(Equal(1))
				Expect(fakeUnprivilegedCommandCreator.CreateCommandCallCount()).To(Equal(0))

				_, handleArg, specArg := fakePrivilegedCommandCreator.CreateCommandArgsForCall(0)
				Expect(handleArg).To(Equal(handle))
				Expect(specArg).To(Equal(rootfsProviderSpec))
			})
		})

		Context("when spec.Rootfs is not defined", func() {
			BeforeEach(func() {
				rootfs = ""
			})

			It("uses the default rootfs instead", func() {
				Expect(createErr).NotTo(HaveOccurred())
				Expect(fakeUnprivilegedCommandCreator.CreateCommandCallCount()).To(Equal(1))

				_, _, specArg := fakeUnprivilegedCommandCreator.CreateCommandArgsForCall(0)
				Expect(specArg.RootFS.String()).To(Equal("/default-rootfs"))
			})

			Context("when there is an error parsing the default rootfs", func() {
				BeforeEach(func() {
					defaultRootfs = "%%"
				})

				It("returns the error", func() {
					Expect(createErr).To(MatchError(ContainSubstring("parsing default rootfs")))
				})
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

		Context("when the image plugin emits logs to stderr", func() {
			BeforeEach(func() {
				buffer := gbytes.NewBuffer()
				externalLogger := lager.NewLogger("external-plugin")
				externalLogger.RegisterSink(lager.NewWriterSink(buffer, lager.DEBUG))
				externalLogger.Info("info-message", lager.Data{"type": "info"})

				fakeImagePluginStderr = string(buffer.Contents())
			})

			It("relogs the log entries", func() {
				Expect(fakeLogger).To(glager.ContainSequence(
					glager.Info(
						glager.Message("image-plugin.image-plugin-create.external-plugin.info-message"),
						glager.Data("type", "info"),
					),
				))
			})
		})
	})

	Describe("Destroy", func() {
		var (
			cmd *exec.Cmd

			handle string

			fakeImagePluginStdout string
			fakeImagePluginStderr string
			fakeImagePluginError  error

			destroyErr error

			namespaced bool
		)

		BeforeEach(func() {
			cmd = exec.Command("unpriv-plugin", "destroy")
			fakeUnprivilegedCommandCreator.DestroyCommandReturns(cmd)
			fakePrivilegedCommandCreator.DestroyCommandReturns(cmd)

			handle = "test-handle"

			fakeImagePluginStdout = ""
			fakeImagePluginStderr = ""
			fakeImagePluginError = nil

			namespaced = true //assume unprivileged by default

			destroyErr = nil
		})

		JustBeforeEach(func() {
			fakeCommandRunner.WhenRunning(
				fake_command_runner.CommandSpec{
					Path: cmd.Path,
				},
				func(cmd *exec.Cmd) error {
					cmd.Stdout.Write([]byte(fakeImagePluginStdout))
					cmd.Stderr.Write([]byte(fakeImagePluginStderr))
					return fakeImagePluginError
				},
			)

			destroyErr = imagePlugin.Destroy(fakeLogger, handle, namespaced)
		})

		It("calls the unprivileged command creator to generate a destroy command", func() {
			Expect(destroyErr).NotTo(HaveOccurred())
			Expect(fakePrivilegedCommandCreator.CreateCommandCallCount()).To(Equal(0))
			Expect(fakeUnprivilegedCommandCreator.DestroyCommandCallCount()).To(Equal(1))

			_, handleArg := fakeUnprivilegedCommandCreator.DestroyCommandArgsForCall(0)
			Expect(handleArg).To(Equal(handle))
		})

		Context("when destroying an unprivileged volume", func() {
			BeforeEach(func() {
				namespaced = false
			})

			It("calls the privileged command creator to generate a destroy command", func() {
				Expect(destroyErr).NotTo(HaveOccurred())
				Expect(fakePrivilegedCommandCreator.DestroyCommandCallCount()).To(Equal(1))
				Expect(fakeUnprivilegedCommandCreator.DestroyCommandCallCount()).To(Equal(0))

				_, handleArg := fakePrivilegedCommandCreator.DestroyCommandArgsForCall(0)
				Expect(handleArg).To(Equal(handle))
			})
		})

		It("runs the plugin command with the command runner", func() {
			Expect(destroyErr).NotTo(HaveOccurred())
			Expect(fakeCommandRunner.ExecutedCommands()).To(HaveLen(1))
			executedCmd := fakeCommandRunner.ExecutedCommands()[0]

			Expect(executedCmd).To(Equal(cmd))
		})

		Context("when running the image plugin destroy fails", func() {
			BeforeEach(func() {
				fakeImagePluginStdout = "image-plugin-exploded-due-to-oom"
				fakeImagePluginError = errors.New("image-plugin-delete-failed")
			})

			It("returns the wrapped error and plugin stdout, with context", func() {
				str := fmt.Sprintf("running image plugin destroy: %s: %s",
					fakeImagePluginStdout, fakeImagePluginError)
				Expect(destroyErr).To(MatchError(str))
			})
		})

		Context("when the image plugin emits logs to stderr", func() {
			BeforeEach(func() {
				buffer := gbytes.NewBuffer()
				externalLogger := lager.NewLogger("external-plugin")
				externalLogger.RegisterSink(lager.NewWriterSink(buffer, lager.DEBUG))
				externalLogger.Error("error-message", errors.New("failed!"), lager.Data{"type": "error"})

				fakeImagePluginStderr = string(buffer.Contents())
			})

			It("relogs the log entries", func() {
				Expect(fakeLogger).To(glager.ContainSequence(
					glager.Error(
						errors.New("failed!"),
						glager.Message("image-plugin.image-plugin-destroy.external-plugin.error-message"),
						glager.Data("type", "error"),
					),
				))
			})
		})

		// var (
		// 	cmd *exec.Cmd

		// 	handle     string
		// 	namespaced bool

		// 	fakeImagePluginStdout string
		// 	fakeImagePluginStderr string
		// 	fakeImagePluginError  error

		// 	destroyErr error
		// )

		// BeforeEach(func() {
		// 	cmd = exec.Command("unpriv-plugin", "destroy")
		// 	fakeUnprivilegedCommandCreator.DestroyCommandReturns(cmd, nil)
		// 	fakePrivilegedCommandCreator.DestroyCommandReturns(cmd, nil)
		// 	handle = "test-handle"

		// 	fakeImagePluginStdout = "something\n"
		// 	fakeImagePluginStderr = ""
		// 	fakeImagePluginError = nil

		// 	destroyErr = nil
		// 	namespaced = true

		// })

		// JustBeforeEach(func() {
		// 	fakeCommandRunner.WhenRunning(
		// 		fake_command_runner.CommandSpec{
		// 			Path: cmd.Path,
		// 		},
		// 		func(cmd *exec.Cmd) error {
		// 			cmd.Stdout.Write([]byte(fakeImagePluginStdout))
		// 			cmd.Stderr.Write([]byte(fakeImagePluginStderr))
		// 			return fakeImagePluginError
		// 		},
		// 	)

		// 	destroyErr = imagePlugin.Destroy(fakeLogger, handle, namespaced)
		// })

		// It("calls the plugin to generate a destroy command", func() {
		// 	Expect(destroyErr).NotTo(HaveOccurred())
		// 	Expect(fakeUnprivilegedCommandCreator.DestroyCommandCallCount()).To(Equal(1))
		// 	Expect(fakePrivilegedCommandCreator.DestroyCommandCallCount()).To(Equal(0))

		// 	_, handleArg := fakeUnprivilegedCommandCreator.DestroyCommandArgsForCall(0)
		// 	Expect(handleArg).To(Equal(handle))
		// })

		// Context("when running with a privileged image plugin", func() {
		// 	BeforeEach(func() {
		// 		namespaced = false
		// 	})

		// 	It("calls the plugin to generate a create command with a privileged creator", func() {
		// 		Expect(destroyErr).NotTo(HaveOccurred())
		// 		Expect(fakeUnprivilegedCommandCreator.DestroyCommandCallCount()).To(Equal(0))
		// 		Expect(fakePrivilegedCommandCreator.DestroyCommandCallCount()).To(Equal(1))

		// 		_, handleArg := fakePrivilegedCommandCreator.DestroyCommandArgsForCall(0)
		// 		Expect(handleArg).To(Equal(handle))

		// 	})
		// })

		// // DISCUSS 1
		// // It("returns trimmed plugin stdout concatenated with 'rootfs'", func() {
		// // 	Expect(createRootfs).To(Equal("/image-rootfs/rootfs"))
		// // })

		// Context("when running the image plugin destroy fails", func() {
		// 	BeforeEach(func() {
		// 		fakeImagePluginStdout = "image-plugin-exploded-due-to-oom"
		// 		fakeImagePluginError = errors.New("image-plugin-delete-failed")
		// 	})

		// 	It("returns the wrapped error and plugin stdout, with context", func() {
		// 		str := fmt.Sprintf("running image plugin destroy: %s: %s",
		// 			fakeImagePluginStdout, fakeImagePluginError)
		// 		Expect(destroyErr).To(MatchError(str))
		// 	})
		// })

		// Context("when the image plugin emits logs to stderr", func() {
		// 	BeforeEach(func() {
		// 		buffer := gbytes.NewBuffer()
		// 		externalLogger := lager.NewLogger("external-plugin")
		// 		externalLogger.RegisterSink(lager.NewWriterSink(buffer, lager.DEBUG))
		// 		externalLogger.Error("error-message", errors.New("failed!"), lager.Data{"type": "error"})

		// 		fakeImagePluginStderr = string(buffer.Contents())
		// 	})

		// 	It("relogs the log entries", func() {
		// 		Expect(fakeLogger).To(glager.ContainSequence(
		// 			glager.Error(
		// 				errors.New("failed!"),
		// 				glager.Message("image-plugin.image-plugin-destroy.external-plugin.error-message"),
		// 				glager.Data("type", "error"),
		// 			),
		// 		))
		// 	})
		// })

	})
})
