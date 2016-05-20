package runrunc_test

import (
	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry-incubator/guardian/rundmc/runrunc"
	"github.com/cloudfoundry-incubator/guardian/rundmc/runrunc/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DistributingRuncRunner", func() {
	var (
		execer             runrunc.RuncExecer
		fakeDadooExecer    *fakes.FakeRuncExecer
		fakeIoDaemonExecer *fakes.FakeRuncExecer
	)

	BeforeEach(func() {
		fakeDadooExecer = new(fakes.FakeRuncExecer)
		fakeIoDaemonExecer = new(fakes.FakeRuncExecer)
		execer = runrunc.NewDistributingExecer(fakeIoDaemonExecer, fakeDadooExecer)
	})

	Describe("Exec", func() {
		Context("when a TTY spec is passed in", func() {
			It("delegates to the iodaemon runc execer", func() {
				_, err := execer.Exec(
					nil,
					"bundlePath",
					"handle",
					garden.ProcessSpec{
						TTY: &garden.TTYSpec{},
					},
					garden.ProcessIO{},
				)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeIoDaemonExecer.ExecCallCount()).To(Equal(1))
				Expect(fakeDadooExecer.ExecCallCount()).To(Equal(0))
			})
		})

		Context("when a TTY spec is not passed in", func() {
			It("delegates to the dadoo runc execer", func() {
				_, err := execer.Exec(
					nil,
					"bundlePath",
					"handle",
					garden.ProcessSpec{},
					garden.ProcessIO{},
				)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeDadooExecer.ExecCallCount()).To(Equal(1))
				Expect(fakeIoDaemonExecer.ExecCallCount()).To(Equal(0))
			})
		})
	})

	Describe("Attach", func() {
		It("delegates to the iodaemon RuncExecer", func() {
			_, err := execer.Attach(
				nil,
				"bundlePath",
				"handle",
				"pid",
				garden.ProcessIO{},
			)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeIoDaemonExecer.AttachCallCount()).To(Equal(1))
			Expect(fakeDadooExecer.AttachCallCount()).To(Equal(0))
		})
	})
})
