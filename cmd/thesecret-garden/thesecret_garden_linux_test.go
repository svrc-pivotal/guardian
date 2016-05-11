package main_test

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = FDescribe("ThesecretGarden", func() {
	const script = `
	  set -e -x
	  echo ${DATA_DIR}
		mkdir -p ${DATA_DIR}
		mount -t tmpfs tmpfs ${DATA_DIR}
		touch ${DATA_DIR}/mysecret
		echo -n password > ${DATA_DIR}/mysecret
	`

	var (
		fakeDataDir string
		err         error
	)

	BeforeEach(func() {
		fakeDataDir, err = ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())
	})

	It("", func() {
		unshare := func() *os.Process {
			cmd := exec.Command("unshare", "-m", "/bin/bash")
			cmd.Env = []string{
				"PATH=/bin:/usr/bin",
				fmt.Sprintf("DATA_DIR=%s", filepath.Join(fakeDataDir, "secret")),
			}

			stdinPipe, _ := cmd.StdinPipe()
			bw := bufio.NewWriter(stdinPipe)
			bw.WriteString(in)
			bw.Flush()
			stdinPipe.Close()

			err = cmd.Start()
			Expect(err).ToNot(HaveOccurred())

			go func() {
				err = cmd.Wait()
				Expect(err).ToNot(HaveOccurred())
			}()

			return cmd.Process
		}

		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		// Eventually(func() []os.FileInfo {
		// 	info, err := ioutil.ReadDir(fakeDataDir)
		// 	Expect(err).NotTo(HaveOccurred())
		// 	return info
		// }, "10s").ShouldNot(BeEmpty())

		Eventually(session).Should(gexec.Exit(0))
		Expect(session.Out).To(gbytes.Say("blah"))

		bytes, err := ioutil.ReadFile(fmt.Sprintf("%s/secret/mysecret", fakeDataDir))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(bytes)).To(Equal("password"))
	})
})
