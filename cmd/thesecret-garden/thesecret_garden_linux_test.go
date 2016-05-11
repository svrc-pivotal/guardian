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
	"github.com/onsi/gomega/gexec"
)

var _ = FDescribe("The Secret Garden", func() {
	const script = `
	  echo $DATA_DIR
	`
	// set -e -x
	// echo ${DATA_DIR}
	// mkdir -p ${DATA_DIR}
	// mount -t tmpfs tmpfs ${DATA_DIR}
	// touch ${DATA_DIR}/mysecret
	// echo -n password > ${DATA_DIR}/mysecret

	var (
		fakeDataDir string
		err         error
		session     *gexec.Session
	)

	unshareCmd := func(program string, args ...string) *os.Process {
		cmd := exec.Command("unshare", "-m", "/bin/bash")
		cmd.Env = []string{
			"PATH=/bin:/usr/bin",
			fmt.Sprintf("DATA_DIR=%s", filepath.Join(fakeDataDir, "secret")),
		}

		stdinPipe, _ := cmd.StdinPipe()
		bw := bufio.NewWriter(stdinPipe)
		bw.WriteString(fmt.Sprintf("%s %s", program, args...))
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

	BeforeEach(func() {
		fakeDataDir, err = ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())
		unshareCmd(theSecretGardenBin, "/var/vcap/data", script)
	})

	It("", func() {
		Eventually(session).Should(Succeed())
	})
})
