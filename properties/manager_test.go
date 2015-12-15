package properties_test

import (
	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry-incubator/guardian/properties"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Properties", func() {
	var (
		propertyManager *properties.Manager
	)

	Describe("Manager", func() {
		BeforeEach(func() {
			propertyManager = properties.NewManager()

			err := propertyManager.CreateKeySpace("handle")
			Expect(err).NotTo(HaveOccurred())

			err = propertyManager.Set("handle", "name", "value")
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("CreateKeySpace", func() {
			It("does not reinitialize key space if present", func() {
				val, err := propertyManager.Get("handle", "name")
				Expect(err).NotTo(HaveOccurred())
				Expect(val).To(Equal("value"))

				err = propertyManager.CreateKeySpace("handle")
				Expect(err).NotTo(HaveOccurred())

				val, err = propertyManager.Get("handle", "name")
				Expect(err).NotTo(HaveOccurred())
				Expect(val).To(Equal("value"))
			})
		})

		Describe("DestroyKeySpace", func() {
			It("removes key space", func() {
				err := propertyManager.DestroyKeySpace("handle")
				Expect(err).NotTo(HaveOccurred())

				err = propertyManager.DestroyKeySpace("handle")
				Expect(err).To(MatchError("No such key space: handle"))
			})
		})

		Describe("All", func() {
			It("returns the properties", func() {
				props, err := propertyManager.All("handle")
				Expect(err).NotTo(HaveOccurred())

				Expect(props).To(HaveLen(1))
				Expect(props).To(HaveKeyWithValue("name", "value"))
			})
		})

		Describe("Get", func() {
			It("returns a specific property when passed a name", func() {
				property, err := propertyManager.Get("handle", "name")

				Expect(err).NotTo(HaveOccurred())
				Expect(property).To(Equal("value"))
			})
		})

		Describe("Remove", func() {
			It("removes properties", func() {
				props, err := propertyManager.All("handle")
				Expect(err).NotTo(HaveOccurred())

				Expect(props).To(HaveLen(1))

				err = propertyManager.Remove("handle", "name")
				Expect(err).NotTo(HaveOccurred())

				_, err = propertyManager.Get("handle", "name")
				Expect(err).To(MatchError("No such property: name"))
			})

			Context("when attempting to remove a property that doesn't exist", func() {
				It("returns a NoSuchPropertyError", func() {
					err := propertyManager.Remove("handle", "missing")
					Expect(err).To(MatchError("No such property: missing"))
				})
			})
		})

		Describe("Set", func() {
			Context("when the property already exists", func() {
				It("updates the property value", func() {
					err := propertyManager.Set("handle", "name", "some-other-value")
					Expect(err).NotTo(HaveOccurred())

					props, err := propertyManager.All("handle")
					Expect(err).NotTo(HaveOccurred())
					Expect(props).To(HaveKeyWithValue("name", "some-other-value"))
				})
			})
		})

		Describe("MatchesAll", func() {
			Context("when the properties list is empty", func() {
				It("matches", func() {
					Expect(propertyManager.MatchesAll("", garden.Properties{})).To(BeTrue())
				})
			})

			Context("when the properties list contains a single property", func() {
				Context("which isn't in the keyspace", func() {
					It("does not match", func() {
						match := propertyManager.MatchesAll("", garden.Properties{"fred": "bob"})
						Expect(match).To(BeFalse())
					})
				})

				Context("...which is in the keyspace", func() {
					BeforeEach(func() {
						propertyManager.CreateKeySpace("flintstones")
						propertyManager.Set("flintstones", "wilma", "fred")
					})

					It("matches", func() {
						match := propertyManager.MatchesAll("flintstones", garden.Properties{"wilma": "fred"})
						Expect(match).To(BeTrue())
					})

					Context("with the wrong value", func() {
						It("does not match", func() {
							match := propertyManager.MatchesAll("flintstones", garden.Properties{"wilma": "pebbles"})
							Expect(match).To(BeFalse())
						})
					})
				})
			})

			Context("when the properties list contains many properties", func() {
				Context("all of which are in the keyspace", func() {
					BeforeEach(func() {
						propertyManager.CreateKeySpace("flintstones")
						propertyManager.Set("flintstones", "wilma", "fred")
						propertyManager.Set("flintstones", "betty", "barney")
					})

					It("matches", func() {
						match := propertyManager.MatchesAll("flintstones",
							garden.Properties{"wilma": "fred", "betty": "barney"})
						Expect(match).To(BeTrue())
					})
				})

				Context("only some of which are in the namespace", func() {
					BeforeEach(func() {
						propertyManager.CreateKeySpace("flintstones")
						propertyManager.Set("flintstones", "wilma", "fred")
						propertyManager.Set("flintstones", "betty", "barney")
					})

					It("does not match", func() {
						match := propertyManager.MatchesAll("flintstones",
							garden.Properties{"wilma": "fred", "pebbles": "bambam", "betty": "barney"})
						Expect(match).To(BeFalse())
					})
				})
			})
		})
	})
})
