package properties_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/guardian/properties"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SaveLoad", func() {
	var (
		depotPath string
	)

	BeforeEach(func() {
		var err error
		depotPath, err = ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(depotPath)).To(Succeed())
	})

	It("returns a new manager when the file is not found", func() {
		mgr, err := properties.Load("/path/does/not/exist")
		Expect(err).NotTo(HaveOccurred())
		Expect(mgr).NotTo(BeNil())
	})

	// It("save and load the state of the property manager", func() {
	// 	mgr := properties.NewManager()
	// 	mgr.Set("foo", "bar", "baz")

	// 	Expect(properties.Save(path.Join(propPath, "props.json"), mgr)).To(Succeed())
	// 	newMgr, err := properties.Load(path.Join(propPath, "props.json"))
	// 	Expect(err).NotTo(HaveOccurred())

	// 	val, ok := newMgr.Get("foo", "bar")
	// 	Expect(ok).To(BeTrue())
	// 	Expect(val).To(Equal("baz"))
	// })

	// It("returns an error when decoding fails", func() {
	// 	Expect(ioutil.WriteFile(path.Join(propPath, "props.json"), []byte("{teest: banana"), 0655)).To(Succeed())

	// 	_, err := properties.Load(path.Join(propPath, "props.json"))
	// 	Expect(err).To(HaveOccurred())
	// })
	//
	FDescribe("Load", func() {
		Context("when there are no properties files", func() {
			It("returns a new properties manager", func() {
				mgr, err := properties.Load("/path/does/not/exist")
				Expect(err).NotTo(HaveOccurred())
				Expect(mgr).NotTo(BeNil())
			})
		})

		Context("when there is one properties file in the dir tree", func() {

			var bundlePath string

			BeforeEach(func() {
				var err error
				bundlePath, err = ioutil.TempDir(depotPath, "")
				Expect(err).NotTo(HaveOccurred())

				propertiesFile := path.Join(bundlePath, "props.json")
				Expect(properties.Save(propertiesFile, garden.Properties{"key": "val"})).To(Succeed())
			})

			FIt("loads the properties into the manager", func() {
				mgr, err := properties.Load(depotPath)
				Expect(err).NotTo(HaveOccurred())

				val, ok := mgr.Get(filepath.Base(bundlePath), "key")

				Expect(ok).To(BeTrue())
				Expect(val).To(Equal("val"))
			})

		})
	})

	Describe("Save", func() {
		It("stores the given properties in the properties file", func() {
			propertiesFile := path.Join(depotPath, "props.json")
			Expect(properties.Save(propertiesFile, garden.Properties{"key": "val"})).To(Succeed())

			file, err := os.Open(propertiesFile)
			Expect(err).NotTo(HaveOccurred())

			properties := garden.Properties{}
			decoder := json.NewDecoder(file)
			Expect(decoder.Decode(&properties)).To(Succeed())

			Expect(properties).To(HaveKeyWithValue("key", "val"))
		})

		It("returns an error when cannot write to the file", func() {
			Expect(properties.Save("/path/to/non/existing.json", garden.Properties{})).To(HaveOccurred())
		})
	})
})
