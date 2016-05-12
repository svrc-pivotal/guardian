package main_test

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = FDescribe("The Secret Garden", func() {
	var (
		stubProcess string
		fakeDataDir string
		session     *gexec.Session
	)

	//TODO: Remove env flags.
	unshareCmd := func(program string, args ...string) *gexec.Session {
		cmd := exec.Command("unshare", "-m", "/bin/bash")
		cmd.Env = []string{"PATH=/bin:/usr/bin:."}

		stdinPipe, err := cmd.StdinPipe()
		Expect(err).NotTo(HaveOccurred())

		bw := bufio.NewWriter(stdinPipe)
		bw.WriteString(fmt.Sprintf("%s %s\n", program, strings.Join(args, " ")))
		bw.Flush()
		stdinPipe.Close()

		sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)

		Expect(err).NotTo(HaveOccurred())

		return sess
	}

	BeforeEach(func() {
		var err error
		fakeDataDir, err = ioutil.TempDir("", "data")
		Expect(err).NotTo(HaveOccurred())

		Expect(exec.Command("mount", "--bind", fakeDataDir, fakeDataDir).Run()).To(Succeed())
		Expect(exec.Command("mount", "--make-shared", fakeDataDir).Run()).To(Succeed())
	})

	AfterEach(func() {
		Expect(exec.Command("umount", fakeDataDir).Run()).To(Succeed())
		Expect(os.RemoveAll(fakeDataDir)).To(Succeed())
	})

	It("exits non-zero when it fails to mount --make-shared", func() {
		session = unshareCmd(theSecretGardenBin, "nnonexistent-dir", "pwd")
		Eventually(session).Should(gexec.Exit(1))
		Expect(session.Out).To(gbytes.Say("Failed to make slave 'nnonexistent-dir'"))
	})

	It("exits non-zero when the command it execs fails", func() {
		session = unshareCmd(theSecretGardenBin, fakeDataDir, "no-such-cmd")
		Eventually(session).Should(gexec.Exit(1))
		Expect(session.Out).To(gbytes.Say("Failed to execute command 'no-such-cmd'"))
	})

	Context("when the process creates a mount", func() {
		const makeSecretMount = `#!/bin/sh
			set -e -x
			mkdir -p ${1}
			mount -t tmpfs tmpfs ${1}
			touch ${1}/mysecret
			echo -n password > ${1}/mysecret
		`

		var secretDir string

		BeforeEach(func() {
			stubProcess = "create-mount.sh"
			secretDir = filepath.Join(fakeDataDir, "secret")
			Expect(ioutil.WriteFile(stubProcess, []byte(makeSecretMount), 0777)).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.Remove(stubProcess)).To(Succeed())
		})

		It("prevents the mount to be seen from outside the namespace", func() {
			session = unshareCmd(theSecretGardenBin, fakeDataDir, stubProcess, secretDir)

			Consistently(func() []os.FileInfo {
				fileInfo, _ := ioutil.ReadDir(secretDir)
				return fileInfo
			}).Should(BeEmpty())

			Eventually(session).Should(gexec.Exit(0))
		})
	})

	Context("when a mount is created outside the namespace", func() {
		const accessSharedMount = `#!/bin/sh
			set -e -x
			sleep 1
			stat ${1}/myfile
		`
		var sharedDir string

		BeforeEach(func() {
			stubProcess = "access-mount.sh"
			sharedDir = filepath.Join(fakeDataDir, "shared")
			Expect(ioutil.WriteFile(stubProcess, []byte(accessSharedMount), 0777)).To(Succeed())
		})

		AfterEach(func() {
			Expect(exec.Command("umount", sharedDir).Run()).To(Succeed())
			Expect(os.Remove(stubProcess)).To(Succeed())
		})

		It("is visible inside the unshared namespace", func() {
			session = unshareCmd(theSecretGardenBin, fakeDataDir, stubProcess, sharedDir)

			Expect(exec.Command("mkdir", sharedDir).Run()).To(Succeed())
			Expect(exec.Command("mount", "-t", "tmpfs", "tmpfs", sharedDir).Run()).To(Succeed())
			Expect(exec.Command("touch", filepath.Join(sharedDir, "myfile")).Run()).To(Succeed())

			Eventually(session, "3s").Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("shared/myfile"))
		})
	})
})
