package gqt_test

import (
	"io/ioutil"
	"os"
	"strings"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/guardian/gqt/runner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
			tmpDir     string
			outputFile *os.File
		)

		BeforeEach(func() {
			var err error
			tmpDir, err = ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			Expect(os.Chmod(tmpDir, 0777)).To(Succeed())

			outputFile, err = ioutil.TempFile(tmpDir, "")
			Expect(err).NotTo(HaveOccurred())

			args = append(args,
				"--image-plugin", testImagePluginBin,
				"--image-plugin-extra-arg", "\"--image-path\"",
				"--image-plugin-extra-arg", tmpDir,
				"--image-plugin-extra-arg", "\"--output-path\"",
				"--image-plugin-extra-arg", outputFile.Name())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})

		Context("and a container is created", func() {
			var (
				containerSpec garden.ContainerSpec
				container     garden.Container
			)

			JustBeforeEach(func() {
				var err error
				container, err = client.Create(containerSpec)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				Expect(client.Destroy(container.Handle())).To(Succeed())
			})

			It("executes the plugin, passing the extra args", func() {
				pluginArgsBytes, err := ioutil.ReadFile(outputFile.Name())
				Expect(err).ToNot(HaveOccurred())

				pluginArgs := strings.Split(string(pluginArgsBytes), " ")
				Expect(pluginArgs).To(ContainElement("--output-path"))
				Expect(pluginArgs).To(ContainElement(outputFile.Name()))
			})
		})
	})

})
