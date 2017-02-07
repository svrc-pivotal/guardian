package guardiancmd_test

import (
	. "code.cloudfoundry.org/guardian/guardiancmd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fakes "code.cloudfoundry.org/guardian/guardiancmd/guardiancmdfakes"
)

var _ = Describe("Setup", func() {
	var (
		setupCommand         *SetupCommand
		fakeSystemConfigurer *fakes.FakeSystemConfigurer
	)

	BeforeEach(func() {
		fakeSystemConfigurer = new(fakes.FakeSystemConfigurer)

		setupCommand = &SetupCommand{
			SystemConfigurer: fakeSystemConfigurer,
		}
	})

	It("calls Start on all the provided garden.Starters", func() {
		Expect(fakeSystemConfigurer.StartCallCount()).To(Equal(1))

	})

	Context("when one of the garden.Starters returns an error", func() {

	})
})
