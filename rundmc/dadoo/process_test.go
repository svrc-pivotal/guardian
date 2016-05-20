package dadoo_test

import (
	"github.com/cloudfoundry-incubator/guardian/rundmc/dadoo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Process", func() {
	var (
		process *dadoo.Process
	)

	BeforeEach(func() {
		process = dadoo.NewProcess("process-id-1")
	})

	Describe("ID", func() {
		It("returns the process id", func() {
			Expect(process.ID()).To(Equal("process-id-1"))
		})
	})

	// 	FDescribe("Wait", func() {
	// 		It("waits for the process to finish", func() {
	// 			Expect(process.Start(exec.Command("sh", "-c", "echo 42 > /proc/self/fd/3"))).To(Succeed())

	// 			exitCode, err := process.Wait()
	// 			Expect(err).NotTo(HaveOccurred())
	// 			Expect(exitCode).To(Equal(42))
	// 		})
	// 	})

})
