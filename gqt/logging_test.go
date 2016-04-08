package gqt_test

import (
	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry-incubator/guardian/gqt/runner"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = FDescribe("Logging", func() {
	var (
		client        *runner.RunningGarden
		container     garden.Container
		containerSpec garden.ContainerSpec
	)

	BeforeEach(func() {
		containerSpec = garden.ContainerSpec{
			RootFSPath: runner.RootFSPath,
		}
	})

	JustBeforeEach(func() {
		var err error
		client = startGarden()
		container, err = client.Create(containerSpec)
		Expect(err).ToNot(HaveOccurred())
	})

	It("forwards RunC logs", func() {
		Eventually(client, "1s").Should(gbytes.Say("guardian-runc-logging-test-start"))
		Eventually(client, "1s").Should(gbytes.Say("peshod"))
		Eventually(client, "1s").Should(gbytes.Say("peshoi"))
		Eventually(client, "1s").Should(gbytes.Say("peshow"))
	})
})
