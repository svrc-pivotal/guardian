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

	Context("when an unprivileged image plugin is provided", func() {
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
				"--image-plugin-extra-arg", filepath.Join(tmpDir, "args"),
				"--image-plugin-extra-arg", "\"--whoami-path\"",
				"--image-plugin-extra-arg", filepath.Join(tmpDir, "whoami"))
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})

		Context("and a container is created", func() {
			var (
				containerSpec garden.ContainerSpec
				container     garden.Container
				handle        string
			)

			JustBeforeEach(func() {
				var err error
				container, err = client.Create(containerSpec)
				Expect(err).NotTo(HaveOccurred())
				handle = container.Handle()
			})

			AfterEach(func() {
				Expect(client.Destroy(container.Handle())).To(Succeed())
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
					"--whoami-path", filepath.Join(tmpDir, "whoami"),
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

				whoamiBytes, err := ioutil.ReadFile(filepath.Join(tmpDir, "whoami"))
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
