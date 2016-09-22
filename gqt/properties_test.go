package gqt_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/guardian/gardener"
	"code.cloudfoundry.org/guardian/gqt/runner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Properties", func() {
	var (
		args          []string
		client        *runner.RunningGarden
		container     garden.Container
		props         garden.Properties
		propertiesDir string
	)

	BeforeEach(func() {
		var err error
		propertiesDir, err = ioutil.TempDir("", "props")
		// poolDir, err := ioutil.TempDir("", "pool")
		// Expect(err).NotTo(HaveOccurred())
		// args = append(args, "--properties-path", path.Join(propertiesDir, "props.json"), "--destroy-containers-on-startup", "--port-pool-properties-path", path.Join(poolDir, "pool"))
		// args = append(args, "--properties-path", path.Join(propertiesDir, "props.json"), "--port-pool-size", "2", "--destroy-containers-on-startup", "--port-pool-properties-path", path.Join(poolDir, "pool"))
		args = append(args, "--properties-path", path.Join(propertiesDir, "props.json"), "--destroy-containers-on-startup")

		client = startGarden(args...)
		props = garden.Properties{"somename": "somevalue"}

		container, err = client.Create(garden.ContainerSpec{
			Properties: props,
		})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(client.DestroyAndStop()).To(Succeed())
		Expect(os.RemoveAll(propertiesDir)).To(Succeed())
	})

	FIt("doesn't leak iptables rules over a restart, even if the depot dir doesn't exist", func() {
		// Create a container (happens in BeforeEach)

		// Add a couple of NetIns to the container
		// Set hostPort = 0 to force garden to acquire a port for us
		_, _, err := container.NetIn(0, 8080)
		Expect(err).NotTo(HaveOccurred())
		_, _, err = container.NetIn(0, 2222)
		Expect(err).NotTo(HaveOccurred())

		// Stop the server (thus writing props.json to disk)
		Expect(client.Stop()).To(Succeed())

		// rm -rf bundle dir
		// this simulates a partial/incomplete container destroy
		// e.g. as may (or may not) happen during evacuation (???)
		bundleDir := filepath.Join(client.DepotDir, container.Handle())
		Expect(os.RemoveAll(bundleDir)).To(Succeed())

		// Start garden server
		// --destroy-containers-on-startup will now not find and destroy the partially cleaned up container
		// because it won't see the container's bundle dir in the depot
		// BUT networking/properties etc. for the container will still be hanging around ...
		client = startGarden(args...)

		// Create another container
		container2, err := client.Create(garden.ContainerSpec{})
		Expect(err).NotTo(HaveOccurred())

		// Add a couple of NetIns to the container
		// Set hostPort = 0 to force garden to acquire a port for us
		// Only this time .. it will probably re-assign the host port it assigned from before
		// because we never cleaned it up properly
		_, _, err = container2.NetIn(0, 8080)
		Expect(err).NotTo(HaveOccurred())
		_, _, err = container2.NetIn(0, 2222)
		Expect(err).NotTo(HaveOccurred())

		// Check status of iptables on box to confirm suspicions
		time.Sleep(time.Hour)
	})

	It("can get properties", func() {
		properties, err := container.Properties()
		Expect(err).NotTo(HaveOccurred())
		Expect(properties).To(HaveKeyWithValue("somename", "somevalue"))
	})

	It("can set a single property", func() {
		err := container.SetProperty("someothername", "someothervalue")
		Expect(err).NotTo(HaveOccurred())

		properties, err := container.Properties()
		Expect(err).NotTo(HaveOccurred())
		Expect(properties).To(HaveKeyWithValue("somename", "somevalue"))
		Expect(properties).To(HaveKeyWithValue("someothername", "someothervalue"))
	})

	It("can get a single property", func() {
		err := container.SetProperty("bing", "bong")
		Expect(err).NotTo(HaveOccurred())

		value, err := container.Property("bing")
		Expect(err).NotTo(HaveOccurred())
		Expect(value).To(Equal("bong"))
	})

	It("can remove a single property", func() {
		err := container.SetProperty("bing", "bong")
		Expect(err).NotTo(HaveOccurred())

		err = container.RemoveProperty("bing")
		Expect(err).NotTo(HaveOccurred())

		_, err = container.Property("bing")
		Expect(err).To(HaveOccurred())
	})

	It("can filter containers based on their properties", func() {
		_, err := client.Create(garden.ContainerSpec{
			Properties: garden.Properties{
				"somename": "wrongvalue",
			},
		})
		Expect(err).NotTo(HaveOccurred())

		containers, err := client.Containers(props)
		Expect(err).NotTo(HaveOccurred())
		Expect(containers).To(HaveLen(1))
		Expect(containers).To(ConsistOf(container))
	})

	It("can get the default properties", func() {
		props, err := container.Properties()
		Expect(err).ToNot(HaveOccurred())

		Expect(props).To(HaveKey("kawasaki.bridge-interface"))
		Expect(props).To(HaveKey(gardener.BridgeIPKey))
		Expect(props).To(HaveKey(gardener.ContainerIPKey))
		Expect(props).To(HaveKey("kawasaki.host-interface"))
		Expect(props).To(HaveKey("kawasaki.iptable-inst"))
		Expect(props).To(HaveKey("kawasaki.subnet"))
		Expect(props).To(HaveKey("kawasaki.container-interface"))
		Expect(props).To(HaveKey(gardener.ExternalIPKey))
		Expect(props).To(HaveKey("kawasaki.mtu"))
	})

	Context("after a server restart", func() {
		It("can still get the container's properties", func() {
			beforeProps, err := container.Properties()
			Expect(err).NotTo(HaveOccurred())

			Expect(client.Stop()).To(Succeed())
			client = startGarden(args...)

			afterProps, err := container.Properties()
			Expect(err).NotTo(HaveOccurred())

			Expect(beforeProps).To(Equal(afterProps))
		})
	})
})
