package main_test

import (
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var theSecretGarden string

func TestThesecretGarden(t *testing.T) {
	RegisterFailHandler(Fail)
	skip := os.Getenv("GARDEN_TEST_ROOTFS") == ""

	SynchronizedBeforeSuite(func() []byte {
		var err error
		bins := make(map[string]string)

		if skip {
			return nil
		}

		bins["the_secret_garden"], err = gexec.Build("github.com/cloudfoundry-incubator/guardian/cmd/thesecret-garden")
		Expect(err).NotTo(HaveOccurred())

		data, err := json.Marshal(bins)
		Expect(err).NotTo(HaveOccurred())

		return data
	}, func(data []byte) {
		if skip {
			return
		}

		bins := make(map[string]string)
		Expect(json.Unmarshal(data, &bins)).To(Succeed())

		theSecretGarden = bins["the_secret_garden"]
	})

	BeforeEach(func() {
		if skip {
			Skip("thesecret-garden requires linux")
		}
	})

	RunSpecs(t, "ThesecretGarden Suite")
}
