package gqt_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/guardian/gqt/runner"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("gdn setup", func() {
	var (
		cgroupsMountpoint string
		iptablesPrefix    string
		args              []string
	)

	BeforeEach(func() {
		cgroupsMountpoint = filepath.Join(os.TempDir(), fmt.Sprintf("cgroups-%d", GinkgoParallelNode()))
		iptablesPrefix = fmt.Sprintf("w-%d", GinkgoParallelNode())
		args = []string{"setup", "--tag", fmt.Sprintf("%d", GinkgoParallelNode())}
	})

	JustBeforeEach(func() {
		setupProcess, err := gexec.Start(exec.Command(gardenBin, args...), GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(setupProcess).Should(gexec.Exit(0))
	})

	AfterEach(func() {
		umountCmd := exec.Command("sh", "-c", fmt.Sprintf("umount %s/*", cgroupsMountpoint))
		Expect(umountCmd.Run()).To(Succeed(), "unmount %s", cgroupsMountpoint)
		umountCmd = exec.Command("sh", "-c", fmt.Sprintf("umount %s", cgroupsMountpoint))
		Expect(umountCmd.Run()).To(Succeed(), "unmount %s", cgroupsMountpoint)

		//Remove custom rules from  INPUT/FORWARD CHAINS
		//I assume is number 1 rule but this might not the case !!! CHANGE
		cmd := exec.Command("iptables", "-D", "INPUT", "1")
		Expect(cmd.Run()).To(Succeed(), "iptables -D INPUT 1")
		cmd = exec.Command("iptables", "-D", "FORWARD", "1")
		Expect(cmd.Run()).To(Succeed(), "iptables -D FORWARD 1")
		//Flush and then delete CHAINS
		for _, suffix := range []string{"-input", "-default", "-forward"} {
			cmd := exec.Command("iptables", "-F", iptablesPrefix+suffix)
			Expect(cmd.Run()).To(Succeed(), "iptables -F %s", iptablesPrefix+suffix)
			cmd = exec.Command("iptables", "-X", iptablesPrefix+suffix)
			Expect(cmd.Run()).To(Succeed(), "iptables -X %s", iptablesPrefix+suffix)
		}
	})

	It("sets up cgroups", func() {
		mountpointCmd := exec.Command("mountpoint", "-q", cgroupsMountpoint+"/")
		Expect(mountpointCmd.Run()).To(Succeed())
	})

	It("sets up iptables", func() {
		out, err := exec.Command("iptables", "-nvL", "INPUT").CombinedOutput()
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring(iptablesPrefix + "-input"))

		out, err = exec.Command("iptables", "-nvL", "FORWARD").CombinedOutput()
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring(iptablesPrefix + "-forward"))

		for _, suffix := range []string{"-input", "-default", "-forward"} {
			_, err := exec.Command("iptables", "-nvL", iptablesPrefix+suffix).CombinedOutput()
			Expect(err).NotTo(HaveOccurred())
		}
	})

	//TODO. check that every flag is working as expected
	// DestroyContainersOnStartup bool
	// IPTables                   FileFlag
	// DenyNetworks               []CIDRFlag
	// AllowHostAccess            bool

	// Not validated that the args are passed properly to gdn setup
	// Context("when custom flag is passed", func() {
	// 	BeforeEach(func() {
	// 		args = append(args, []string{"--AllowHostAccess"}...)
	// 	})

	// 	It("iptables should have the relevant entry ", func() {
	// 		out, err := exec.Command("iptables", "-nvL").CombinedOutput()
	// 		Expect(err).NotTo(HaveOccurred())
	// 		Expect(out).To(ContainSubstring("specific output"))
	// 	})

	// })
})

var _ = Describe("running gdn setup before starting server", func() {
	var client *runner.RunningGarden

	BeforeEach(func() {
		setupProcess, err := gexec.Start(exec.Command(gardenBin, "setup"), GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(setupProcess).Should(gexec.Exit(0))
		client = startGarden("server")
	})

	AfterEach(func() {
		Expect(client.DestroyAndStop()).To(Succeed())
	})

	It("creates a container", func() {
		_, err := client.Create(garden.ContainerSpec{})
		Expect(err).NotTo(HaveOccurred())
	})
})
