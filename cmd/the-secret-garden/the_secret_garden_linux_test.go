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
		sharedDir   string
		secretDir   string
		err         error
		session     *gexec.Session
	)

	//TODO: Remove env flags.
	unshareCmd := func(program string, args ...string) *gexec.Session {
		cmd := exec.Command("unshare", "-m", "/bin/bash")
		cmd.Env = []string{
			"PATH=/bin:/usr/bin:.",
			fmt.Sprintf("SECRET_DIR=%s", secretDir),
			fmt.Sprintf("SHARED_DIR=%s", sharedDir),
		}

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
		fakeDataDir, err = ioutil.TempDir("", "data")
		Expect(err).NotTo(HaveOccurred())
		sharedDir = filepath.Join(fakeDataDir, "shared")
		secretDir = filepath.Join(fakeDataDir, "secret")

		Expect(exec.Command("mount", "--bind", fakeDataDir, fakeDataDir).Run()).To(Succeed())
		Expect(exec.Command("mount", "--make-shared", fakeDataDir).Run()).To(Succeed())
	})

	AfterEach(func() {
		Expect(exec.Command("umount", fakeDataDir).Run()).To(Succeed())
		Expect(os.RemoveAll(fakeDataDir)).To(Succeed())
	})

	It("returns exit code when it fails to mount a slave mountpoint", func() {
		session = unshareCmd(theSecretGardenBin, "", "")
		Eventually(session).Should(gexec.Exit(1))
	})

	Context("when the process creates a mount", func() {
		const makeSecretMount = `#!/bin/sh
		set -e -x
		mkdir -p ${SECRET_DIR}
		mount -t tmpfs tmpfs ${SECRET_DIR}
		touch ${SECRET_DIR}/mysecret
		echo -n password > ${SECRET_DIR}/mysecret
	`

		BeforeEach(func() {
			stubProcess = "create-mount.sh"

			Expect(ioutil.WriteFile(stubProcess, []byte(makeSecretMount), 0777)).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.Remove(stubProcess)).To(Succeed())
		})

		It("prevents the mount to be seen from outside the namespace", func() {
			session = unshareCmd(theSecretGardenBin, fakeDataDir, stubProcess)

			Consistently(func() []os.FileInfo {
				fileInfo, _ := ioutil.ReadDir(secretDir)
				return fileInfo
			}, "5s").Should(BeEmpty())

			Eventually(session, "5s").Should(gexec.Exit(0))
		})
	})

	Context("when a mount is created outside the namespace", func() {
		const accessSharedMount = `#!/bin/sh
		set -e -x
		sleep 1
		stat ${SHARED_DIR}/myfile
	`

		BeforeEach(func() {
			stubProcess = "access-mount.sh"
			Expect(ioutil.WriteFile(stubProcess, []byte(accessSharedMount), 0777)).To(Succeed())
		})

		AfterEach(func() {
			Expect(exec.Command("umount", sharedDir).Run()).To(Succeed())
			Expect(os.Remove(stubProcess)).To(Succeed())
		})

		It("is visible inside the unshared namespace", func() {
			session = unshareCmd(theSecretGardenBin, fakeDataDir, stubProcess)

			Expect(exec.Command("mkdir", sharedDir).Run()).To(Succeed())
			Expect(exec.Command("mount", "-t", "tmpfs", "tmpfs", sharedDir).Run()).To(Succeed())
			Expect(exec.Command("touch", filepath.Join(sharedDir, "myfile")).Run()).To(Succeed())

			Eventually(session, "5s").Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("shared/myfile"))
		})
	})
})
