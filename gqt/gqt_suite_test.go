package gqt_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/guardian/gqt/runner"
	"code.cloudfoundry.org/guardian/guardiancmd"
	"code.cloudfoundry.org/guardian/sysinfo"
	flags "github.com/jessevdk/go-flags"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"encoding/json"
	"testing"
)

var defaultRuntime = map[string]string{
	"linux": "runc",
}

var ginkgoIO = garden.ProcessIO{Stdout: GinkgoWriter, Stderr: GinkgoWriter}

var ociRuntimeBin, gardenSetupBin, gardenBin, initBin, nstarBin, dadooBin, testImagePluginBin, inspectorGardenBin, testNetPluginBin, tarBin string

func TestGqt(t *testing.T) {
	RegisterFailHandler(Fail)

	SynchronizedBeforeSuite(func() []byte {
		var err error
		bins := make(map[string]string)

		bins["oci_runtime_path"] = os.Getenv("OCI_RUNTIME")
		if bins["oci_runtime_path"] == "" {
			bins["oci_runtime_path"] = defaultRuntime[runtime.GOOS]
		}

		if bins["oci_runtime_path"] != "" {
			bins["garden_setup_bin_path"], err = gexec.Build("code.cloudfoundry.org/guardian/cmd/guardian-setup")
			Expect(err).NotTo(HaveOccurred())

			bins["garden_bin_path"], err = gexec.Build("code.cloudfoundry.org/guardian/cmd/guardian", "-tags", "daemon", "-race", "-ldflags", "-extldflags '-static'")
			Expect(err).NotTo(HaveOccurred())

			bins["dadoo_bin_bin_bin"], err = gexec.Build("code.cloudfoundry.org/guardian/cmd/dadoo")
			Expect(err).NotTo(HaveOccurred())

			bins["init_bin_path"], err = gexec.Build("code.cloudfoundry.org/guardian/cmd/init")
			Expect(err).NotTo(HaveOccurred())

			bins["inspector-garden_bin_path"], err = gexec.Build("code.cloudfoundry.org/guardian/cmd/inspector-garden")
			Expect(err).NotTo(HaveOccurred())

			bins["test_net_plugin_bin_path"], err = gexec.Build("code.cloudfoundry.org/guardian/gqt/cmd/networkplugin")
			Expect(err).NotTo(HaveOccurred())

			bins["test_image_plugin_bin_path"], err = gexec.Build("code.cloudfoundry.org/guardian/gqt/cmd/fake_image_plugin")
			Expect(err).NotTo(HaveOccurred())

			cmd := exec.Command("make")
			cmd.Dir = "../rundmc/nstar"
			cmd.Stdout = GinkgoWriter
			cmd.Stderr = GinkgoWriter
			Expect(cmd.Run()).To(Succeed())
			bins["nstar_bin_path"] = "../rundmc/nstar/nstar"
		}

		data, err := json.Marshal(bins)
		Expect(err).NotTo(HaveOccurred())

		return data
	}, func(data []byte) {
		bins := make(map[string]string)
		Expect(json.Unmarshal(data, &bins)).To(Succeed())

		ociRuntimeBin = bins["oci_runtime_path"]
		gardenBin = bins["garden_bin_path"]
		gardenSetupBin = bins["garden_setup_bin_path"]
		nstarBin = bins["nstar_bin_path"]
		dadooBin = bins["dadoo_bin_bin_bin"]
		testImagePluginBin = bins["test_image_plugin_bin_path"]
		initBin = bins["init_bin_path"]
		inspectorGardenBin = bins["inspector-garden_bin_path"]
		testNetPluginBin = bins["test_net_plugin_bin_path"]

		tarBin = os.Getenv("GARDEN_TAR_PATH")
	})

	BeforeEach(func() {
		if ociRuntimeBin == "" {
			Skip("No OCI Runtime for Platform: " + runtime.GOOS)
		}

		if os.Getenv("GARDEN_TEST_ROOTFS") == "" {
			Skip("No Garden RootFS")
		}

		// chmod all the artifacts
		Expect(os.Chmod(filepath.Join(initBin, "..", ".."), 0755)).To(Succeed())
		filepath.Walk(filepath.Join(initBin, "..", ".."), func(path string, info os.FileInfo, err error) error {
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Chmod(path, 0755)).To(Succeed())
			return nil
		})
	})

	SetDefaultEventuallyTimeout(5 * time.Second)
	RunSpecs(t, "GQT Suite")
}

// type GuardianSetupArgs struct {
// 	Logger                     guardiancmd.LagerFlag
// 	ServerTag                  string                 `long:"tag" description:"Optional 2-character identifier used for namespacing global configuration."`
// 	IPTableBinPath             string                 `long:"iptables-bin"  default:"/sbin/iptables" description:"path to the iptables binary"`
// 	DestroyContainersOnStartup bool                   `long:"destroy-containers-on-startup" description:"Clean up all the existing containers on startup."`
// 	AllowHostAccess            bool                   `long:"allow-host-access" description:"Allow network access to the host machine."`
// 	DenyNetworks               []guardiancmd.CIDRFlag `long:"deny-network"      description:"Network ranges to which traffic from containers will be denied. Can be specified multiple times."`
// }

func stringArgs(parsedArgs *guardiancmd.GuardianCommand) []string {
	args := []string{}
	args = append(args, "--log-level", parsedArgs.Logger.LogLevel)
	args = append(args, "--tag", parsedArgs.Server.Tag)
	args = append(args, "--iptables-bin", parsedArgs.Bin.IPTables.Path())

	if parsedArgs.Containers.DestroyContainersOnStartup {
		args = append(args, "--destroy-containers-on-startup")
	}

	if parsedArgs.Network.AllowHostAccess {
		args = append(args, "--allow-host-access")
	}

	for _, cidr := range parsedArgs.Network.DenyNetworks {
		args = append(args, "--deny-network", cidr.String())
	}

	return args
}

func runGardenSetup(args ...string) {
	parsedArgs := &guardiancmd.GuardianCommand{}
	parser := flags.NewParser(parsedArgs, flags.Default)
	parser.NamespaceDelimiter = "-"

	// parse args
	args, err := parser.ParseArgs(args)
	Expect(err).NotTo(HaveOccurred())

	cmd := exec.Command(gardenSetupBin, stringArgs(parsedArgs)...)
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	Expect(cmd.Run()).To(Succeed())
}

func startGarden(argv ...string) *runner.RunningGarden {
	rootfs := os.Getenv("GARDEN_TEST_ROOTFS")

	// depotDir := filepath.Join(os.TempDir(), fmt.Sprintf("test-garden-%d", GinkgoParallelNode()), "containers")
	// Expect(os.MkdirAll(depotDir, 0755)).To(Succeed())
	runGardenSetup(append(argv, []string{"--tag", fmt.Sprintf("%d", GinkgoParallelNode()), "--depot", "/tmp", "--init-bin", initBin, "--nstar-bin", nstarBin, "--dadoo-bin", dadooBin, "--tar-bin", tarBin}...)...)
	return runner.Start(gardenBin, initBin, nstarBin, dadooBin, testImagePluginBin, rootfs, tarBin, nil, argv...)
}

func startGardenRootless(argv ...string) *runner.RunningGarden {
	runGardenSetup()
	rootfs := os.Getenv("GARDEN_TEST_ROOTFS")
	maxId := uint32(sysinfo.Min(sysinfo.MustGetMaxValidUID(), sysinfo.MustGetMaxValidGID()))
	maximus := &syscall.Credential{Uid: maxId, Gid: maxId}

	return runner.Start(gardenBin, initBin, nstarBin, dadooBin, testImagePluginBin, rootfs, tarBin, maximus, argv...)
}

func restartGarden(client *runner.RunningGarden, argv ...string) {
	Expect(client.Ping()).To(Succeed(), "tried to restart garden while it was not running")
	Expect(client.Stop()).To(Succeed())
	client = startGarden(argv...)
}

func startGardenWithoutDefaultRootfs(argv ...string) *runner.RunningGarden {
	return runner.Start(gardenBin, initBin, nstarBin, dadooBin, testImagePluginBin, "", tarBin, nil, argv...)
}
