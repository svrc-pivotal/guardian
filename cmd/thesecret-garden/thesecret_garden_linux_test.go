package main_test

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = FDescribe("ThesecretGarden", func() {
	var (
		fakeDataDir string
		err         error
	)

	BeforeEach(func() {
		fakeDataDir, err = ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())
	})

	It("", func() {
		script := fmt.Sprintf(`unshare -m %s %s;
		mkdir -p %s/secret;
		mount -t tmpfs tmpfs %s/secret;
		touch %s/secret/mysecret;
		echo hello > %s/secret/mysecret;
		sleep 60`,
			theSecretGarden, fakeDataDir, fakeDataDir, fakeDataDir, fakeDataDir, fakeDataDir)
		_, err := gexec.Start(exec.Command("/bin/sh", "-c", script), GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(1 * time.Second)

		// sess.Wait()
		Expect(ioutil.ReadDir(fmt.Sprintf("%s", fakeDataDir))).NotTo(BeEmpty())

		bytes, err := ioutil.ReadFile(fmt.Sprintf("%s/secret/mysecret", fakeDataDir))
		Expect(err).NotTo(HaveOccurred())
		Expect(bytes).To(BeEmpty())
	})
})
