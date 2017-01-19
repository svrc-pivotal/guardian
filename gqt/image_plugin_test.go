package gqt_test

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/guardian/gqt/runner"
	"code.cloudfoundry.org/guardian/imageplugin"
	"code.cloudfoundry.org/guardian/sysinfo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = FDescribe("Image Plugin", func() {

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
		Expect(client.Stop()).To(Succeed())
	})

	Context("when only an unprivileged image plugin is provided", func() {
		var (
			tmpDir string
		)

		BeforeEach(func() {
			var err error
			tmpDir, err = ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			Expect(os.Chmod(tmpDir, 0777)).To(Succeed())

			args = append(args,
				"--image-plugin", testImagePluginBin,
				"--image-plugin-extra-arg", "\"--image-path\"",
				"--image-plugin-extra-arg", tmpDir,
				"--image-plugin-extra-arg", "\"--args-path\"",
				"--image-plugin-extra-arg", filepath.Join(tmpDir, "args"))
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})

		Context("and an unprivileged container is successfully created", func() {
			var (
				containerSpec garden.ContainerSpec
				container     garden.Container
				handle        string

				destroyContainer bool
			)

			BeforeEach(func() {
				destroyContainer = true

				args = append(args,
					"--image-plugin-extra-arg", "\"--create-whoami-path\"",
					"--image-plugin-extra-arg", filepath.Join(tmpDir, "create-whoami"))
			})

			JustBeforeEach(func() {
				var err error
				container, err = client.Create(containerSpec)
				Expect(err).NotTo(HaveOccurred())
				handle = container.Handle()
			})

			AfterEach(func() {
				if destroyContainer {
					Expect(client.Destroy(container.Handle())).To(Succeed())
				}
			})

			It("executes the plugin, passing the correct args", func() {
				maxId := uint32(sysinfo.Min(sysinfo.MustGetMaxValidUID(), sysinfo.MustGetMaxValidGID()))

				pluginArgsBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "args"))
				Expect(err).ToNot(HaveOccurred())

				pluginArgs := strings.Split(string(pluginArgsBytes), " ")
				Expect(pluginArgs).To(Equal([]string{
					testImagePluginBin,
					"--image-path", tmpDir,
					"--args-path", filepath.Join(tmpDir, "args"),
					"--create-whoami-path", filepath.Join(tmpDir, "create-whoami"),
					"create",
					"--uid-mapping", fmt.Sprintf("0:%d:1", maxId),
					"--gid-mapping", fmt.Sprintf("0:%d:1", maxId),
					"--uid-mapping", fmt.Sprintf("1:1:%d", maxId-1),
					"--gid-mapping", fmt.Sprintf("1:1:%d", maxId-1),
					os.Getenv("GARDEN_TEST_ROOTFS"),
					handle,
				}))
			})

			It("executes the plugin as the correct user", func() {
				maxId := uint32(sysinfo.Min(sysinfo.MustGetMaxValidUID(), sysinfo.MustGetMaxValidGID()))

				whoamiBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "create-whoami"))
				Expect(err).NotTo(HaveOccurred())

				Expect(string(whoamiBytes)).To(ContainSubstring(fmt.Sprintf("%d - %d", maxId, maxId)))
			})

			Context("when there are env vars", func() {
				BeforeEach(func() {
					customImageJsonFile, err := ioutil.TempFile("", "")
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

					args = append(args,
						"--image-plugin-extra-arg", "\"--json-file-to-copy\"",
						"--image-plugin-extra-arg", customImageJsonFile.Name(),
					)

					gardenDefaultRootfs := os.Getenv("GARDEN_TEST_ROOTFS")
					Expect(copyFile(filepath.Join(gardenDefaultRootfs, "bin", "env"),
						filepath.Join(tmpDir, "rootfs", "env"))).To(Succeed())
				})

				It("loads the image.json env variables", func() {
					buffer := gbytes.NewBuffer()
					process, err := container.Run(garden.ProcessSpec{
						Path: "/env",
						Dir:  "/",
					}, garden.ProcessIO{Stdout: buffer, Stderr: buffer})
					Expect(err).NotTo(HaveOccurred())
					exitCode, err := process.Wait()
					Expect(err).NotTo(HaveOccurred())
					Expect(exitCode).To(BeZero())

					Eventually(buffer).Should(gbytes.Say("MY_VAR=set"))
					Eventually(buffer).Should(gbytes.Say("MY_SECOND_VAR=also_set"))
				})
			})

			Context("when rootfs is not specified", func() {
				BeforeEach(func() {
					containerSpec.RootFSPath = ""
				})

				It("uses the default rootfs", func() {
					pluginArgsBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "args"))
					Expect(err).ToNot(HaveOccurred())

					Expect(string(pluginArgsBytes)).To(ContainSubstring(os.Getenv("GARDEN_TEST_ROOTFS")))
				})
			})

			Context("when using tagged docker images", func() {
				BeforeEach(func() {
					containerSpec.RootFSPath = "docker:///busybox#1.26.1"
				})

				It("replaces the # with :", func() {
					pluginArgsBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "args"))
					Expect(err).ToNot(HaveOccurred())

					Expect(string(pluginArgsBytes)).To(ContainSubstring("docker:///busybox:1.26.1"))
				})
			})

			Context("when specifying a quota", func() {
				BeforeEach(func() {
					containerSpec.Limits.Disk.ByteHard = 100000
				})

				It("calls the image plugin setting the quota", func() {
					pluginArgsBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "args"))
					Expect(err).ToNot(HaveOccurred())

					Expect(string(pluginArgsBytes)).To(ContainSubstring("--disk-limit-size-bytes 100000"))
				})

				Context("when the quota is total", func() {
					BeforeEach(func() {
						containerSpec.Limits.Disk.Scope = garden.DiskLimitScopeTotal
					})

					It("calls the image plugin without the exclusive flag", func() {
						pluginArgsBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "args"))
						Expect(err).ToNot(HaveOccurred())

						Expect(string(pluginArgsBytes)).NotTo(ContainSubstring("--exclude-image-from-quota"))
					})
				})

				Context("when the quota is exclusive", func() {
					BeforeEach(func() {
						containerSpec.Limits.Disk.Scope = garden.DiskLimitScopeExclusive
					})

					It("calls the image plugin setting the exclusive flag", func() {
						pluginArgsBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "args"))
						Expect(err).ToNot(HaveOccurred())

						Expect(string(pluginArgsBytes)).To(ContainSubstring("--exclude-image-from-quota"))
					})
				})

				Context("when the plugin logs to stderr", func() {
					BeforeEach(func() {
						args = append(args,
							"--image-plugin-extra-arg", "\"--create-log-content\"",
							"--image-plugin-extra-arg", "CREATE-FAKE-LOG-LINE")
					})

					It("relogs the plugin's stderr to the garden logs", func() {
						Eventually(client).Should(gbytes.Say("CREATE-FAKE-LOG-LINE"))
					})
				})
			})

			Context("and metrics are collected on that container", func() {
				BeforeEach(func() {
					args = append(args,
						"--image-plugin-extra-arg", "\"--metrics-whoami-path\"",
						"--image-plugin-extra-arg", filepath.Join(tmpDir, "metrics-whoami"))
				})

				JustBeforeEach(func() {
					_, err := container.Metrics()
					Expect(err).NotTo(HaveOccurred())
				})

				It("executes the plugin, passing the correct args", func() {
					pluginArgsBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "args"))
					Expect(err).ToNot(HaveOccurred())

					pluginArgs := strings.Split(string(pluginArgsBytes), " ")
					Expect(pluginArgs).To(Equal([]string{
						testImagePluginBin,
						"--image-path", tmpDir,
						"--args-path", filepath.Join(tmpDir, "args"),
						"--create-whoami-path", filepath.Join(tmpDir, "create-whoami"),
						"--metrics-whoami-path", filepath.Join(tmpDir, "metrics-whoami"),
						"stats",
						handle,
					}))
				})

				It("executes the plugin as the correct user", func() {
					maxId := uint32(sysinfo.Min(sysinfo.MustGetMaxValidUID(), sysinfo.MustGetMaxValidGID()))

					whoamiBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "metrics-whoami"))
					Expect(err).NotTo(HaveOccurred())

					Expect(string(whoamiBytes)).To(ContainSubstring(fmt.Sprintf("%d - %d", maxId, maxId)))
				})

				Context("when the plugin logs to stderr", func() {
					BeforeEach(func() {
						args = append(args,
							"--image-plugin-extra-arg", "\"--metrics-log-content\"",
							"--image-plugin-extra-arg", "METRICS-FAKE-LOG-LINE")
					})

					It("relogs the plugin's stderr to the garden logs", func() {
						Eventually(client).Should(gbytes.Say("METRICS-FAKE-LOG-LINE"))
					})
				})
			})

			Context("and that container is destroyed", func() {
				BeforeEach(func() {
					destroyContainer = false
					args = append(args,
						"--image-plugin-extra-arg", "\"--destroy-whoami-path\"",
						"--image-plugin-extra-arg", filepath.Join(tmpDir, "destroy-whoami"))
				})

				JustBeforeEach(func() {
					Expect(client.Destroy(container.Handle())).Should(Succeed())
				})

				It("executes the plugin, passing the correct args", func() {
					pluginArgsBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "args"))
					Expect(err).ToNot(HaveOccurred())

					pluginArgs := strings.Split(string(pluginArgsBytes), " ")
					Expect(pluginArgs).To(Equal([]string{
						testImagePluginBin,
						"--image-path", tmpDir,
						"--args-path", filepath.Join(tmpDir, "args"),
						"--create-whoami-path", filepath.Join(tmpDir, "create-whoami"),
						"--destroy-whoami-path", filepath.Join(tmpDir, "destroy-whoami"),
						"delete",
						handle,
					}))
				})

				It("executes the plugin as the correct user", func() {
					maxId := uint32(sysinfo.Min(sysinfo.MustGetMaxValidUID(), sysinfo.MustGetMaxValidGID()))

					whoamiBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "destroy-whoami"))
					Expect(err).NotTo(HaveOccurred())

					Expect(string(whoamiBytes)).To(ContainSubstring(fmt.Sprintf("%d - %d", maxId, maxId)))
				})

				Context("when the plugin logs to stderr", func() {
					BeforeEach(func() {
						args = append(args,
							"--image-plugin-extra-arg", "\"--destroy-log-content\"",
							"--image-plugin-extra-arg", "DESTROY-FAKE-LOG-LINE")
					})

					It("relogs the plugin's stderr to the garden logs", func() {
						Eventually(client).Should(gbytes.Say("DESTROY-FAKE-LOG-LINE"))
					})
				})
			})
		})

		Context("but we attempt to create a privileged container", func() {
			It("returns an informative error", func() {
				_, err := client.Create(garden.ContainerSpec{Privileged: true})
				Expect(err).To(MatchError(ContainSubstring("no privileged_image_plugin provided")))
			})
		})
	})

	Context("when only a privileged image plugin is provided", func() {
		var (
			tmpDir string
		)

		BeforeEach(func() {
			var err error
			tmpDir, err = ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			Expect(os.Chmod(tmpDir, 0777)).To(Succeed())

			args = append(args,
				"--privileged-image-plugin", testImagePluginBin,
				"--privileged-image-plugin-extra-arg", "\"--image-path\"",
				"--privileged-image-plugin-extra-arg", tmpDir,
				"--privileged-image-plugin-extra-arg", "\"--args-path\"",
				"--privileged-image-plugin-extra-arg", filepath.Join(tmpDir, "args"))
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})

		Context("and a container is created", func() {
			var (
				containerSpec garden.ContainerSpec
				container     garden.Container
				handle        string

				destroyContainer bool
			)

			BeforeEach(func() {
				containerSpec.Privileged = true
				destroyContainer = false

				args = append(args,
					"--privileged-image-plugin-extra-arg", "\"--create-whoami-path\"",
					"--privileged-image-plugin-extra-arg", filepath.Join(tmpDir, "create-whoami"))
			})

			JustBeforeEach(func() {
				var err error
				container, err = client.Create(containerSpec)
				Expect(err).NotTo(HaveOccurred())
				handle = container.Handle()
			})

			AfterEach(func() {
				if destroyContainer {
					Expect(client.Destroy(container.Handle())).To(Succeed())
				}
			})

			It("executes the plugin, passing the correct args", func() {
				pluginArgsBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "args"))
				Expect(err).ToNot(HaveOccurred())

				pluginArgs := strings.Split(string(pluginArgsBytes), " ")
				Expect(pluginArgs).To(Equal([]string{
					testImagePluginBin,
					"--image-path", tmpDir,
					"--args-path", filepath.Join(tmpDir, "args"),
					"--create-whoami-path", filepath.Join(tmpDir, "create-whoami"),
					"create",
					os.Getenv("GARDEN_TEST_ROOTFS"),
					handle,
				}))
			})

			It("executes the plugin as the correct user", func() {
				whoamiBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "create-whoami"))
				Expect(err).NotTo(HaveOccurred())

				Expect(string(whoamiBytes)).To(ContainSubstring("0 - 0"))
			})

			Context("when there are env vars", func() {
				BeforeEach(func() {
					customImageJsonFile, err := ioutil.TempFile("", "")
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

					args = append(args,
						"--privileged-image-plugin-extra-arg", "\"--json-file-to-copy\"",
						"--privileged-image-plugin-extra-arg", customImageJsonFile.Name(),
					)

					gardenDefaultRootfs := os.Getenv("GARDEN_TEST_ROOTFS")
					Expect(copyFile(filepath.Join(gardenDefaultRootfs, "bin", "env"),
						filepath.Join(tmpDir, "rootfs", "env"))).To(Succeed())
				})

				It("loads the image.json env variables", func() {
					buffer := gbytes.NewBuffer()
					process, err := container.Run(garden.ProcessSpec{
						Path: "/env",
						Dir:  "/",
					}, garden.ProcessIO{Stdout: buffer, Stderr: buffer})
					Expect(err).NotTo(HaveOccurred())
					exitCode, err := process.Wait()
					Expect(err).NotTo(HaveOccurred())
					Expect(exitCode).To(BeZero())

					Eventually(buffer).Should(gbytes.Say("MY_VAR=set"))
					Eventually(buffer).Should(gbytes.Say("MY_SECOND_VAR=also_set"))
				})
			})

			Context("when rootfs is not specified", func() {
				BeforeEach(func() {
					containerSpec.RootFSPath = ""
				})

				It("uses the default rootfs", func() {
					pluginArgsBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "args"))
					Expect(err).ToNot(HaveOccurred())

					Expect(string(pluginArgsBytes)).To(ContainSubstring(os.Getenv("GARDEN_TEST_ROOTFS")))
				})
			})

			Context("when using tagged docker images", func() {
				BeforeEach(func() {
					containerSpec.RootFSPath = "docker:///busybox#1.26.1"
				})

				It("replaces the # with :", func() {
					pluginArgsBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "args"))
					Expect(err).ToNot(HaveOccurred())

					Expect(string(pluginArgsBytes)).To(ContainSubstring("docker:///busybox:1.26.1"))
				})
			})

			Context("when specifying a quota", func() {
				BeforeEach(func() {
					containerSpec.Limits.Disk.ByteHard = 100000
				})

				It("calls the image plugin setting the quota", func() {
					pluginArgsBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "args"))
					Expect(err).ToNot(HaveOccurred())

					Expect(string(pluginArgsBytes)).To(ContainSubstring("--disk-limit-size-bytes 100000"))
				})

				Context("when the quota is total", func() {
					BeforeEach(func() {
						containerSpec.Limits.Disk.Scope = garden.DiskLimitScopeTotal
					})

					It("calls the image plugin without the exclusive flag", func() {
						pluginArgsBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "args"))
						Expect(err).ToNot(HaveOccurred())

						Expect(string(pluginArgsBytes)).NotTo(ContainSubstring("--exclude-image-from-quota"))
					})
				})

				Context("when the quota is exclusive", func() {
					BeforeEach(func() {
						containerSpec.Limits.Disk.Scope = garden.DiskLimitScopeExclusive
					})

					It("calls the image plugin setting the exclusive flag", func() {
						pluginArgsBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "args"))
						Expect(err).ToNot(HaveOccurred())

						Expect(string(pluginArgsBytes)).To(ContainSubstring("--exclude-image-from-quota"))
					})
				})
			})

			Context("and metrics are collected on that container", func() {
				BeforeEach(func() {
					args = append(args,
						"--privileged-image-plugin-extra-arg", "\"--metrics-whoami-path\"",
						"--privileged-image-plugin-extra-arg", filepath.Join(tmpDir, "metrics-whoami"))
				})

				JustBeforeEach(func() {
					_, err := container.Metrics()
					Expect(err).NotTo(HaveOccurred())
				})

				It("executes the plugin, passing the correct args", func() {
					pluginArgsBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "args"))
					Expect(err).ToNot(HaveOccurred())

					pluginArgs := strings.Split(string(pluginArgsBytes), " ")
					Expect(pluginArgs).To(Equal([]string{
						testImagePluginBin,
						"--image-path", tmpDir,
						"--args-path", filepath.Join(tmpDir, "args"),
						"--create-whoami-path", filepath.Join(tmpDir, "create-whoami"),
						"--metrics-whoami-path", filepath.Join(tmpDir, "metrics-whoami"),
						"stats",
						handle,
					}))
				})

				It("executes the plugin as the correct user", func() {
					whoamiBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "metrics-whoami"))
					Expect(err).NotTo(HaveOccurred())

					Expect(string(whoamiBytes)).To(ContainSubstring("0 - 0"))
				})

				Context("when the plugin logs to stderr", func() {
					BeforeEach(func() {
						args = append(args,
							"--privileged-image-plugin-extra-arg", "\"--metrics-log-content\"",
							"--privileged-image-plugin-extra-arg", "METRICS-FAKE-LOG-LINE")
					})

					It("relogs the plugin's stderr to the garden logs", func() {
						Eventually(client).Should(gbytes.Say("METRICS-FAKE-LOG-LINE"))
					})
				})
			})

			Context("and that container is destroyed", func() {
				BeforeEach(func() {
					destroyContainer = false

					args = append(args,
						"--privileged-image-plugin-extra-arg", "\"--destroy-whoami-path\"",
						"--privileged-image-plugin-extra-arg", filepath.Join(tmpDir, "destroy-whoami"))
				})

				JustBeforeEach(func() {
					Expect(client.Destroy(container.Handle())).Should(Succeed())
				})

				It("executes the plugin, passing the correct args", func() {
					pluginArgsBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "args"))
					Expect(err).ToNot(HaveOccurred())

					pluginArgs := strings.Split(string(pluginArgsBytes), " ")
					Expect(pluginArgs).To(Equal([]string{
						testImagePluginBin,
						"--image-path", tmpDir,
						"--args-path", filepath.Join(tmpDir, "args"),
						"--create-whoami-path", filepath.Join(tmpDir, "create-whoami"),
						"--destroy-whoami-path", filepath.Join(tmpDir, "destroy-whoami"),
						"delete",
						handle,
					}))
				})

				It("executes the plugin as the correct user", func() {
					whoamiBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "destroy-whoami"))
					Expect(err).NotTo(HaveOccurred())

					Expect(string(whoamiBytes)).To(ContainSubstring("0 - 0"))
				})

				Context("when the plugin logs to stderr", func() {
					BeforeEach(func() {
						args = append(args,
							"--privileged-image-plugin-extra-arg", "\"--destroy-log-content\"",
							"--privileged-image-plugin-extra-arg", "DESTROY-FAKE-LOG-LINE")
					})

					It("relogs the plugin's stderr to the garden logs", func() {
						Eventually(client).Should(gbytes.Say("DESTROY-FAKE-LOG-LINE"))
					})
				})
			})
		})

		FContext("but we attempt to create an unprivileged container", func() {
			It("returns an informative error", func() {
				_, err := client.Create(garden.ContainerSpec{Privileged: false})
				Expect(err).To(MatchError(ContainSubstring("no image_plugin provided")))
			})
		})
	})
})

func copyFile(srcPath, dstPath string) error {
	dirPath := filepath.Dir(dstPath)
	if err := os.MkdirAll(dirPath, 0777); err != nil {
		return err
	}

	reader, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	writer, err := os.Create(dstPath)
	if err != nil {
		reader.Close()
		return err
	}

	if _, err := io.Copy(writer, reader); err != nil {
		writer.Close()
		reader.Close()
		return err
	}

	writer.Close()
	reader.Close()

	return os.Chmod(writer.Name(), 0777)
}
