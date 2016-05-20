package main_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("The Secret Garden", func() {
	var (
		stubProcess        string
		fakeDataDir        string
		realGraphParentDir string
		realGraphDir       string
		graphDir           string
		session            *gexec.Session
	)

	runSecretGarden := func(dataDir, realGraphParentDir, realGraphDir, graphDir, bin string, args ...string) *gexec.Session {
		cmd := exec.Command(theSecretGardenBin, append([]string{dataDir, realGraphParentDir, realGraphDir, graphDir, bin}, args...)...)
		cmd.Env = append(os.Environ(), "BAR=foo")
		sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		return sess
	}

	BeforeEach(func() {
		var err error
		fakeDataDir, err = ioutil.TempDir("", "data")
		Expect(err).NotTo(HaveOccurred())

		realGraphParentDir = filepath.Join(fakeDataDir, "realgraph")
		Expect(os.MkdirAll(realGraphParentDir, 0777)).To(Succeed())

		realGraphDir = filepath.Join(fakeDataDir, "realgraph", "graph")
		Expect(os.MkdirAll(realGraphDir, 0777)).To(Succeed())

		graphDir = filepath.Join(fakeDataDir, "graph")
		Expect(os.MkdirAll(graphDir, 0777)).To(Succeed())
	})

	AfterEach(func() {
		exec.Command("umount", fakeDataDir).Run()
		os.RemoveAll(fakeDataDir)
	})

	It("makes sure the data dir is mounted exactly once", func() {
		session = runSecretGarden(fakeDataDir, realGraphParentDir, realGraphDir, graphDir, "mount")
		Eventually(session).Should(gexec.Exit(0))

		exp := fmt.Sprintf("%s on %s", fakeDataDir, fakeDataDir)
		Expect(session.Out).To(gbytes.Say(exp))

		session = runSecretGarden(fakeDataDir, realGraphParentDir, realGraphDir, graphDir, "mount")
		Eventually(session).Should(gexec.Exit(0))

		Expect(regexp.MustCompile(exp).FindAll(session.Out.Contents(), -1)).To(HaveLen(1))
	})

	It("exits non-zero when it fails to mount --make-shared", func() {
		session = runSecretGarden("nnonexistent-dir", realGraphParentDir, realGraphDir, graphDir, "pwd")
		Eventually(session).ShouldNot(gexec.Exit(0))
	})

	It("changes the realGraphParentDir's permission accordingly", func() {
		session = runSecretGarden(fakeDataDir, realGraphParentDir, realGraphDir, graphDir, "sh", "-c", fmt.Sprintf("stat -c %%A %s", realGraphParentDir))
		Eventually(session).Should(gexec.Exit(0))

		// Checking if the x's are removed from the group and other
		// the os removes the write permissions of group and other
		exp := fmt.Sprintf("drwxr.-r.-")
		Expect(session.Out).To(gbytes.Say(exp))
	})

	It("exits non-zero when the realGraphParentDir is not a valid folder", func() {
		session = runSecretGarden(fakeDataDir, "spiderman-kaput-dir", realGraphDir, graphDir, "ls")
		Eventually(session).ShouldNot(gexec.Exit(0))
	})

	It("mounts the realgraph/graph on the graph path exactly once", func() {
		session = runSecretGarden(fakeDataDir, realGraphParentDir, realGraphDir, graphDir, "mount")
		Eventually(session).Should(gexec.Exit(0))

		exp := fmt.Sprintf("%s on %s", realGraphDir, graphDir)
		Expect(session.Out).To(gbytes.Say(exp))

		session = runSecretGarden(fakeDataDir, realGraphParentDir, realGraphDir, graphDir, "mount")
		Eventually(session).Should(gexec.Exit(0))

		Expect(regexp.MustCompile(exp).FindAll(session.Out.Contents(), -1)).To(HaveLen(1))
	})

	It("exits non-zero when it fails to mount the graph", func() {
		session = runSecretGarden(fakeDataDir, realGraphParentDir, "no-such-dir", graphDir, "mount")
		Eventually(session).ShouldNot(gexec.Exit(0))
	})

	It("exits non-zero when the command it execs fails", func() {
		session = runSecretGarden(fakeDataDir, realGraphParentDir, realGraphDir, graphDir, "no-such-cmd")
		Eventually(session).ShouldNot(gexec.Exit(0))
	})

	It("passes the correct environment to the final process", func() {
		session = runSecretGarden(fakeDataDir, realGraphParentDir, realGraphDir, graphDir, "sh", "-c", "echo $BAR")
		Eventually(session.Out).Should(gbytes.Say("foo"))
		Eventually(session).ShouldNot(gexec.Exit(0))
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
			stubProcess = filepath.Join(fakeDataDir, "create-mount.sh")
			secretDir = filepath.Join(fakeDataDir, "secret")
			Expect(ioutil.WriteFile(stubProcess, []byte(makeSecretMount), 0777)).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.Remove(stubProcess)).To(Succeed())
		})

		It("prevents the mount to be seen from outside the namespace", func() {
			session = runSecretGarden(fakeDataDir, realGraphParentDir, realGraphDir, graphDir, stubProcess, secretDir)

			Consistently(func() []os.FileInfo {
				fileInfo, _ := ioutil.ReadDir(secretDir)
				return fileInfo
			}).Should(BeEmpty())

			Eventually(session).Should(gexec.Exit(0))
		})
	})

	Context("when a mount is created outside the namespace", func() {
		const accessSharedMount = `#!/bin/bash
			set -x
			for i in $(seq 1 10); do
				echo trying $i
				sleep 1
				stat ${1}/myfile
				if [[ $? -eq 0 ]]; then
					exit
				fi
			done
		`
		var sharedDir string

		BeforeEach(func() {
			stubProcess = filepath.Join(fakeDataDir, "access-mount.sh")
			sharedDir = filepath.Join(fakeDataDir, "shared")
			Expect(ioutil.WriteFile(stubProcess, []byte(accessSharedMount), 0777)).To(Succeed())
		})

		AfterEach(func() {
			exec.Command("umount", sharedDir).Run()
			os.Remove(stubProcess)
		})

		It("is visible inside the unshared namespace", func() {
			session = runSecretGarden(fakeDataDir, realGraphParentDir, realGraphDir, graphDir, stubProcess, sharedDir)
			Eventually(func() string {
				out, err := exec.Command("mount").CombinedOutput()
				Expect(err).NotTo(HaveOccurred())
				return string(out)
			}).Should(ContainSubstring(fmt.Sprintf("%s on %s type none (rw,bind)", fakeDataDir, fakeDataDir)))

			Expect(exec.Command("mkdir", sharedDir).Run()).To(Succeed())
			Expect(exec.Command("mount", "-t", "tmpfs", "tmpfs", sharedDir).Run()).To(Succeed())

			Expect(exec.Command("touch", filepath.Join(sharedDir, "myfile")).Run()).To(Succeed())
			Expect(exec.Command("stat", filepath.Join(sharedDir, "myfile")).Run()).To(Succeed())

			Eventually(session, "10s").Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("shared/myfile"))
		})
	})

	Context("when the secret garden is terminated", func() {
		var sharedDir string

		const processes = `#!/bin/bash
			trap "echo terminating; exit 0" SIGTERM

			echo 'sleeping'
			sleep 1000 &
			wait
		`

		BeforeEach(func() {
			stubProcess = filepath.Join(fakeDataDir, "spawn-processes.sh")
			sharedDir = filepath.Join(fakeDataDir, "shared")
			Expect(ioutil.WriteFile(stubProcess, []byte(processes), 0777)).To(Succeed())
		})

		It("should kill the underlying process", func() {
			session = runSecretGarden(fakeDataDir, realGraphParentDir, realGraphDir, graphDir, stubProcess, sharedDir)
			Eventually(session.Out).Should(gbytes.Say("sleeping"))

			session.Terminate()

			Eventually(session.Out).Should(gbytes.Say("terminating"))
		})
	})

	Context("when the secret garden is killed", func() {
		var sharedDir string

		const processes = `#!/bin/bash
			echo "PID: $$"
			sleep 1000
		`

		BeforeEach(func() {
			stubProcess = filepath.Join(fakeDataDir, "spawn-processes.sh")
			sharedDir = filepath.Join(fakeDataDir, "shared")
			Expect(ioutil.WriteFile(stubProcess, []byte(processes), 0777)).To(Succeed())
		})

		It("should kill the underlying process", func() {
			session = runSecretGarden(fakeDataDir, realGraphParentDir, realGraphDir, graphDir, stubProcess, sharedDir)
			Eventually(session.Out).Should(gbytes.Say("PID: "))

			contents, err := ioutil.ReadAll(session.Out)
			Expect(err).NotTo(HaveOccurred())

			pid, err := strconv.ParseInt(strings.TrimSpace(string(contents)), 10, 32)
			Expect(err).NotTo(HaveOccurred())

			session.Kill()

			Eventually(fmt.Sprintf("/proc/%d", pid)).ShouldNot(BeADirectory())
		})
	})
})
