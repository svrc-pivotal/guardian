package gqt_test

import (
	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry-incubator/guardian/gqt/runner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = FDescribe("Port Pool", func() {

	Context("when the port pool is exhausted by container creation and then containers are destroyed", func() {
		var (
			containers   []garden.Container
			client       *runner.RunningGarden
			expectedPort uint32

			args = []string{"--port-pool-size", "5"}
		)

		BeforeEach(func() {
			containers = []garden.Container{}
			client = startGarden(args...)

			// Create containers and NetIn
			for i := 0; i < 5; i++ {
				container, err := client.Create(garden.ContainerSpec{})
				Expect(err).NotTo(HaveOccurred())

				hostPort, _, err := container.NetIn(0, 0)
				if i == 4 {
					expectedPort = hostPort
				}
				Expect(err).NotTo(HaveOccurred())

				containers = append(containers, container)
			}

			// Destroy containers in reverse order
			for i := 4; i >= 0; i-- {
				Expect(client.Destroy(containers[i].Handle())).To(Succeed())
			}
		})

		AfterEach(func() {
			Expect(client.DestroyAndStop()).To(Succeed())
		})

		It("should FIFO released/allocated ports", func() {
			for i := 0; i < 5; i++ {
				container, err := client.Create(garden.ContainerSpec{})
				Expect(err).NotTo(HaveOccurred())

				hostPort, _, err := container.NetIn(0, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(hostPort).To(Equal(uint32(60004 - i)))
			}
		})

		Context("when guardian is restarted", func() {
			BeforeEach(func() {
				client.Stop()
				client = startGarden(args...)
			})

			It("should persist the head of the port allocation queue", func() {
				container, err := client.Create(garden.ContainerSpec{})
				Expect(err).NotTo(HaveOccurred())

				hostPort, _, err := container.NetIn(0, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(expectedPort).To(Equal(uint32(hostPort)))
			})
		})
	})
})
