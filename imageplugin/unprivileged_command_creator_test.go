package imageplugin_test

import (
	"net/url"
	"os/exec"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/garden-shed/rootfs_provider"
	"code.cloudfoundry.org/guardian/imageplugin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

var _ = Describe("UnprivilegedCommandCreator", func() {
	var (
		commandCreator *imageplugin.UnprivilegedCommandCreator
		binPath        string
		extraArgs      []string
		idMappings     []specs.LinuxIDMapping
	)

	BeforeEach(func() {
		binPath = "/image-plugin"
		extraArgs = []string{}
		idMappings = []specs.LinuxIDMapping{
			specs.LinuxIDMapping{
				ContainerID: 0,
				HostID:      100,
				Size:        1,
			},
			specs.LinuxIDMapping{
				ContainerID: 1,
				HostID:      1,
				Size:        99,
			},
		}
	})

	JustBeforeEach(func() {
		commandCreator = &imageplugin.UnprivilegedCommandCreator{
			BinPath:    binPath,
			ExtraArgs:  extraArgs,
			IDMappings: idMappings,
		}
	})

	Describe("CreateCommand", func() {
		var (
			createCmd *exec.Cmd
			spec      rootfs_provider.Spec
		)

		BeforeEach(func() {
			rootfsURL, err := url.Parse("/fake-registry/image")
			Expect(err).NotTo(HaveOccurred())
			spec = rootfs_provider.Spec{RootFS: rootfsURL}
		})

		JustBeforeEach(func() {
			createCmd = commandCreator.CreateCommand(nil, "test-handle", spec)
		})

		It("returns a command with the correct image plugin path", func() {
			Expect(createCmd.Path).To(Equal(binPath))
		})

		It("returns a command with the create action", func() {
			Expect(createCmd.Args[1]).To(Equal("create"))
		})

		It("returns a command that has the id mappings as args", func() {
			Expect(createCmd.Args).To(HaveLen(12))
			Expect(createCmd.Args[2]).To(Equal("--uid-mapping"))
			Expect(createCmd.Args[3]).To(Equal("0:100:1"))
			Expect(createCmd.Args[4]).To(Equal("--gid-mapping"))
			Expect(createCmd.Args[5]).To(Equal("0:100:1"))
			Expect(createCmd.Args[6]).To(Equal("--uid-mapping"))
			Expect(createCmd.Args[7]).To(Equal("1:1:99"))
			Expect(createCmd.Args[8]).To(Equal("--gid-mapping"))
			Expect(createCmd.Args[9]).To(Equal("1:1:99"))
		})

		It("returns a command with the provided rootfs as image", func() {
			Expect(createCmd.Args[10]).To(Equal("/fake-registry/image"))
		})

		It("returns a command with the provided handle as id", func() {
			Expect(createCmd.Args[11]).To(Equal("test-handle"))
		})

		Context("when using a docker image", func() {
			BeforeEach(func() {
				var err error
				spec.RootFS, err = url.Parse("docker:///busybox#1.26.1")
				Expect(err).NotTo(HaveOccurred())
			})

			It("replaces the '#' with ':'", func() {
				Expect(createCmd.Args[10]).To(Equal("docker:///busybox:1.26.1"))
			})
		})

		Context("when disk quota is provided", func() {
			Context("and the quota size is = 0", func() {
				BeforeEach(func() {
					spec.QuotaSize = 0
				})

				It("returns a command without the quota", func() {
					Expect(createCmd.Args).NotTo(ContainElement("--disk-limit-size-bytes"))
				})
			})

			Context("and the quota size is > 0", func() {
				BeforeEach(func() {
					spec.QuotaSize = 100000
				})

				It("returns a command with the quota", func() {
					Expect(createCmd.Args[10]).To(Equal("--disk-limit-size-bytes"))
					Expect(createCmd.Args[11]).To(Equal("100000"))
				})

				Context("and it's got an exclusive scope", func() {
					BeforeEach(func() {
						spec.QuotaScope = garden.DiskLimitScopeExclusive
					})

					It("returns a command with the quota and an exclusive scope", func() {
						Expect(createCmd.Args[10]).To(Equal("--disk-limit-size-bytes"))
						Expect(createCmd.Args[11]).To(Equal("100000"))

						Expect(createCmd.Args).To(ContainElement("--exclude-image-from-quota"))
					})
				})

				Context("and it's got a total scope", func() {
					BeforeEach(func() {
						spec.QuotaScope = garden.DiskLimitScopeTotal
					})

					It("returns a command with the quota and a total scope", func() {
						Expect(createCmd.Args[10]).To(Equal("--disk-limit-size-bytes"))
						Expect(createCmd.Args[11]).To(Equal("100000"))

						Expect(createCmd.Args).NotTo(ContainElement("--exclude-image-from-quota"))
					})
				})
			})
		})

		Context("when extra args are provided", func() {
			BeforeEach(func() {
				extraArgs = []string{"foo", "bar"}
			})

			It("returns a command with the extra args as global args preceeding the action", func() {
				Expect(createCmd.Args[1]).To(Equal("foo"))
				Expect(createCmd.Args[2]).To(Equal("bar"))
				Expect(createCmd.Args[3]).To(Equal("create"))
			})
		})

		It("returns a command that runs as an unprivileged user", func() {
			Expect(createCmd.SysProcAttr.Credential.Uid).To(Equal(idMappings[0].HostID))
			Expect(createCmd.SysProcAttr.Credential.Gid).To(Equal(idMappings[0].HostID))
		})

	})
})

// import (
// 	"code.cloudfoundry.org/guardian/imageplugin"
// 	"code.cloudfoundry.org/lager/lagertest"
// 	. "github.com/onsi/ginkgo"
// 	. "github.com/onsi/gomega"
// )

// var _ = fdescribe("unprivilegedimageplugin", func() {

// 	var (
// 		imageplugin *imageplugin.unprivilegedimageplugin

// 		fakeimagepluginbin string
// 		extraargs          []string

// 		fakelogger *lagertest.testlogger
// 	)

// 	beforeeach(func() {
// 		fakelogger = lagertest.newtestlogger("image-plugin-test")

// 		fakeimagepluginbin = "fake-image-plugin"
// 		extraargs = []string{}
// 	})

// 	justbeforeeach(func() {
// 		imageplugin = &imageplugin.unprivilegedimageplugin{
// 			binpath:   fakeimagepluginbin,
// 			extraargs: extraargs,
// 		}
// 	})

// 	// describe("create", func() {
// 	// 	var (
// 	// 		createcmd *exec.cmd
// 	// 		createerr error
// 	// 	)

// 	// 	justbeforeeach(func() {
// 	// 		rootfs, err := url.parse("/fake-registry/image")
// 	// 		expect(err).notto(haveoccurred())
// 	// 		createcmd = imageplugin.createcommand(fakelogger, "test-handle", rootfs_provider.spec{
// 	// 			rootfs: rootfs,
// 	// 		})
// 	// 	})

// 	// 	it("executes the correct image plugin binary", func() {
// 	// 		imagemanagercmd := fakecommandrunner.executedcommands()[0]
// 	// 		expect(imagemanagercmd.path).to(equal(fakeimagepluginbin))
// 	// 	})

// 	// 	it("executes the image plugin binary with the create action", func() {
// 	// 		imagemanagercmd := fakecommandrunner.executedcommands()[0]
// 	// 		expect(imagemanagercmd.args[1]).to(equal("create"))
// 	// 	})

// 	// 	it("executes the image plugin binary with the provided rootfs as image", func() {
// 	// 		imagemanagercmd := fakecommandrunner.executedcommands()[0]
// 	// 		expect(imagemanagercmd.args[2]).to(equal("/fake-registry/image"))
// 	// 	})

// 	// 	it("executes the image plugin binary with the provided handle as id", func() {
// 	// 		imagemanagercmd := fakecommandrunner.executedcommands()[0]
// 	// 		expect(imagemanagercmd.args[3]).to(equal("test-handle"))
// 	// 	})

// 	// 	context("when extra args are provided", func() {
// 	// 		beforeeach(func() {
// 	// 			extraargs = []string{"foo", "bar"}
// 	// 		})

// 	// 		it("executes the image plugin binary with the extra args as global args preceeding the action", func() {
// 	// 			imagemanagercmd := fakecommandrunner.executedcommands()[0]
// 	// 			expect(imagemanagercmd.args[1]).to(equal("foo"))
// 	// 			expect(imagemanagercmd.args[2]).to(equal("bar"))
// 	// 			expect(imagemanagercmd.args[3]).to(equal("create"))
// 	// 		})
// 	// 	})

// 	// 	it("fowards stderr from the image plugin binary as logs", func() {

// 	// 	})

// 	// 	context("when running the image plugin fails", func() {
// 	// 		beforeeach(func() {
// 	// 			fakecommandrunner.whenrunning(fake_command_runner.commandspec{
// 	// 				path: fakeimagepluginbin,
// 	// 			}, func(cmd *exec.cmd) error {

// 	// 				cmd.stdout.write([]byte("image-plugin-exploded-due-to-oom"))
// 	// 				return errors.new(fmt.sprintf("%s failed", fakeimagepluginbin))
// 	// 			})
// 	// 		})

// 	// 		it("returns the wrapped error and binary's stdout, with context", func() {
// 	// 			expect(createerr).to(matcherror(fmt.sprintf("image-plugin-run-failed: %s: %s failed", "image-plugin-exploded-due-to-oom", fakeimagepluginbin)))
// 	// 		})
// 	// 	})
// 	// })
// })

// // var _ = pdescribe("imageplugin", func() {
// // 	var (
// // 		imageplugin *imageplugin.imageplugin

// // 		fakecommandrunner   *fake_command_runner.fakecommandrunner
// // 		logger              lager.logger
// // 		baseimage           *url.url
// // 		idmappings          []specs.linuxidmapping
// // 		defaultbaseimage    *url.url
// // 		fakecmdrunnerstdout string
// // 		fakecmdrunnerstderr string
// // 		fakecmdrunnererr    error
// // 		privilegedextraargs []string
// // 		extraargs           []string
// // 	)

// // 	beforeeach(func() {
// // 		fakecmdrunnerstdout = ""
// // 		fakecmdrunnerstderr = ""
// // 		fakecmdrunnererr = nil
// // 		privilegedextraargs = []string{"foo", "bar"}
// // 		extraargs = []string{"bar", "foo"}

// // 		logger = glager.newlogger("external-image-manager")
// // 		fakecommandrunner = fake_command_runner.new()

// // 		idmappings = []specs.linuxidmapping{
// // 			specs.linuxidmapping{
// // 				containerid: 0,
// // 				hostid:      100,
// // 				size:        1,
// // 			},
// // 			specs.linuxidmapping{
// // 				containerid: 1,
// // 				hostid:      1,
// // 				size:        99,
// // 			},
// // 		}

// // 		var err error
// // 		defaultbaseimage, err = url.parse("/default/image")
// // 		expect(err).tonot(haveoccurred())
// // 		imageplugin = imageplugin.new("/image-plugin", extraargs, fakecommandrunner)

// // 		baseimage, err = url.parse("/hello/image")
// // 		expect(err).tonot(haveoccurred())

// // 		fakecommandrunner.whenrunning(fake_command_runner.commandspec{
// // 			path: "/external-image-manager-bin",
// // 		}, func(cmd *exec.cmd) error {
// // 			if cmd.stdout != nil {
// // 				cmd.stdout.write([]byte(fakecmdrunnerstdout))
// // 			}
// // 			if cmd.stderr != nil {
// // 				cmd.stderr.write([]byte(fakecmdrunnerstderr))
// // 			}

// // 			return fakecmdrunnererr
// // 		})
// // 	})

// // 	fdescribe("create", func() {
// // 		beforeeach(func() {
// // 			fakecmdrunnerstdout = "/this-is/your\n"
// // 		})

// // 		fit("uses the correct image plugin binary", func() {
// // 			_, _, err := externalimagemanager.create(
// // 				logger, "hello", rootfs_provider.spec{
// // 					rootfs: baseimage,
// // 				},
// // 			)
// // 			expect(err).tonot(haveoccurred())

// // 			expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 			imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 			expect(imagemanagercmd.path).to(equal("/external-image-manager-bin"))
// // 		})

// // 		it("returns the env variables defined in the image configuration", func() {
// // 			imagepath, err := ioutil.tempdir("", "")
// // 			expect(err).notto(haveoccurred())
// // 			fakecmdrunnerstdout = imagepath

// // 			imageconfig := imageplugin_old.image{
// // 				config: imageplugin_old.imageconfig{
// // 					env: []string{"hello=there", "path=/my-path/bin"},
// // 				},
// // 			}

// // 			imageconfigfile, err := os.create(filepath.join(imagepath, "image.json"))
// // 			expect(err).notto(haveoccurred())
// // 			expect(json.newencoder(imageconfigfile).encode(imageconfig)).to(succeed())

// // 			_, envvariables, err := externalimagemanager.create(
// // 				logger, "hello", rootfs_provider.spec{
// // 					rootfs: baseimage,
// // 				},
// // 			)
// // 			expect(err).tonot(haveoccurred())
// // 			expect(envvariables).to(consistof([]string{"hello=there", "path=/my-path/bin"}))
// // 		})

// // 		context("when the image configuration is not defined", func() {
// // 			it("returns an empty list of environment variables", func() {
// // 				_, envvariables, err := externalimagemanager.create(
// // 					logger, "hello", rootfs_provider.spec{
// // 						rootfs: baseimage,
// // 					},
// // 				)
// // 				expect(err).tonot(haveoccurred())
// // 				expect(envvariables).to(beempty())
// // 			})
// // 		})

// // 		context("when the image configuration is not valid json", func() {
// // 			it("returns an error", func() {
// // 				imagepath, err := ioutil.tempdir("", "")
// // 				expect(err).notto(haveoccurred())
// // 				fakecmdrunnerstdout = imagepath
// // 				expect(ioutil.writefile(filepath.join(imagepath, "image.json"), []byte("what-image: is this: no"), 0666)).to(succeed())

// // 				_, _, err = externalimagemanager.create(
// // 					logger, "hello", rootfs_provider.spec{
// // 						rootfs: baseimage,
// // 					},
// // 				)
// // 				expect(err).to(matcherror(containsubstring("parsing image config")))
// // 			})
// // 		})

// // 		describe("external-image-manager parameters", func() {
// // 			it("uses the correct external-image-manager create command", func() {
// // 				_, _, err := externalimagemanager.create(
// // 					logger, "hello", rootfs_provider.spec{
// // 						rootfs: baseimage,
// // 					},
// // 				)
// // 				expect(err).tonot(haveoccurred())

// // 				expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 				imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 				expect(imagemanagercmd.args[3]).to(equal("create"))
// // 			})

// // 			context("when container is privileged", func() {
// // 				it("sets the correct extra arguments", func() {
// // 					_, _, err := externalimagemanager.create(
// // 						logger, "hello", rootfs_provider.spec{
// // 							rootfs:     baseimage,
// // 							namespaced: false,
// // 						},
// // 					)
// // 					expect(err).tonot(haveoccurred())

// // 					expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 					imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 					expect(imagemanagercmd.args[1]).to(equal("foo"))
// // 					expect(imagemanagercmd.args[2]).to(equal("bar"))
// // 				})
// // 			})

// // 			context("when container is unprivileged", func() {
// // 				it("sets the correct extra arguments", func() {
// // 					_, _, err := externalimagemanager.create(
// // 						logger, "hello", rootfs_provider.spec{
// // 							rootfs:     baseimage,
// // 							namespaced: true,
// // 						},
// // 					)
// // 					expect(err).tonot(haveoccurred())

// // 					expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 					imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 					expect(imagemanagercmd.args[1]).to(equal("bar"))
// // 					expect(imagemanagercmd.args[2]).to(equal("foo"))
// // 				})
// // 			})

// // 			it("sets the correct image input to external-image-manager", func() {
// // 				_, _, err := externalimagemanager.create(
// // 					logger, "hello", rootfs_provider.spec{
// // 						rootfs: baseimage,
// // 					},
// // 				)
// // 				expect(err).tonot(haveoccurred())

// // 				expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 				imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 				expect(imagemanagercmd.args[len(imagemanagercmd.args)-2]).to(equal("/hello/image"))
// // 			})

// // 			it("sets the correct id to external-image-manager", func() {
// // 				_, _, err := externalimagemanager.create(
// // 					logger, "hello", rootfs_provider.spec{
// // 						rootfs: baseimage,
// // 					},
// // 				)
// // 				expect(err).tonot(haveoccurred())

// // 				expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 				imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 				expect(imagemanagercmd.args[len(imagemanagercmd.args)-1]).to(equal("hello"))
// // 			})

// // 			context("when the base image is a remote image", func() {
// // 				it("sets the correct image input to external-image-manager", func() {
// // 					baseimage, err := url.parse("docker:///ubuntu#14.04")
// // 					expect(err).notto(haveoccurred())

// // 					_, _, err = externalimagemanager.create(
// // 						logger, "hello", rootfs_provider.spec{
// // 							rootfs: baseimage,
// // 						},
// // 					)
// // 					expect(err).tonot(haveoccurred())

// // 					expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 					imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 					expect(imagemanagercmd.args[len(imagemanagercmd.args)-2]).to(equal("docker:///ubuntu:14.04"))
// // 				})
// // 			})

// // 			context("when namespaced is true", func() {
// // 				it("passes the correct uid and gid mappings to the external-image-manager", func() {
// // 					_, _, err := externalimagemanager.create(
// // 						logger, "hello", rootfs_provider.spec{
// // 							rootfs:     baseimage,
// // 							namespaced: true,
// // 						},
// // 					)
// // 					expect(err).tonot(haveoccurred())

// // 					expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 					imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 					firstmap := fmt.sprintf("%d:%d:%d", idmappings[0].containerid, idmappings[0].hostid, idmappings[0].size)
// // 					secondmap := fmt.sprintf("%d:%d:%d", idmappings[1].containerid, idmappings[1].hostid, idmappings[1].size)

// // 					expect(imagemanagercmd.args[4:12]).to(equal([]string{
// // 						"--uid-mapping", firstmap,
// // 						"--gid-mapping", firstmap,
// // 						"--uid-mapping", secondmap,
// // 						"--gid-mapping", secondmap,
// // 					}))
// // 				})

// // 				it("runs the external-image-manager as the container root user", func() {
// // 					_, _, err := externalimagemanager.create(
// // 						logger, "hello", rootfs_provider.spec{
// // 							rootfs:     baseimage,
// // 							namespaced: true,
// // 						},
// // 					)
// // 					expect(err).tonot(haveoccurred())

// // 					expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 					imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 					expect(imagemanagercmd.sysprocattr.credential.uid).to(equal(idmappings[0].hostid))
// // 					expect(imagemanagercmd.sysprocattr.credential.gid).to(equal(idmappings[0].hostid))
// // 				})
// // 			})

// // 			context("when namespaced is false", func() {
// // 				it("does not pass any uid and gid mappings to the external-image-manager", func() {
// // 					_, _, err := externalimagemanager.create(
// // 						logger, "hello", rootfs_provider.spec{
// // 							rootfs: baseimage,
// // 						},
// // 					)
// // 					expect(err).tonot(haveoccurred())

// // 					expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 					imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 					expect(imagemanagercmd.args).notto(containelement("--uid-mapping"))
// // 					expect(imagemanagercmd.args).notto(containelement("--gid-mapping"))
// // 				})
// // 			})

// // 			context("when a disk quota is provided in the spec", func() {
// // 				it("passes the quota to the external-image-manager", func() {
// // 					_, _, err := externalimagemanager.create(
// // 						logger, "hello", rootfs_provider.spec{
// // 							rootfs:    baseimage,
// // 							quotasize: 1024,
// // 						},
// // 					)
// // 					expect(err).tonot(haveoccurred())

// // 					expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 					imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 					expect(imagemanagercmd.args[4]).to(equal("--disk-limit-size-bytes"))
// // 					expect(imagemanagercmd.args[5]).to(equal("1024"))
// // 				})
// // 			})

// // 			context("when quota scope is exclusive", func() {
// // 				it("passes the quota to the external-image-manager with the exclusive flag", func() {
// // 					_, _, err := externalimagemanager.create(
// // 						logger, "hello", rootfs_provider.spec{
// // 							rootfs:     baseimage,
// // 							quotasize:  1024,
// // 							quotascope: garden.disklimitscopeexclusive,
// // 						},
// // 					)
// // 					expect(err).tonot(haveoccurred())

// // 					expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 					imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 					expect(imagemanagercmd.args[4]).to(equal("--exclude-image-from-quota"))
// // 					expect(imagemanagercmd.args[5]).to(equal("--disk-limit-size-bytes"))
// // 					expect(imagemanagercmd.args[6]).to(equal("1024"))
// // 				})
// // 			})
// // 		})

// // 		it("returns rootfs location", func() {
// // 			returnedrootfs, _, err := externalimagemanager.create(
// // 				logger, "hello", rootfs_provider.spec{
// // 					rootfs: baseimage,
// // 				},
// // 			)
// // 			expect(err).tonot(haveoccurred())

// // 			expect(returnedrootfs).to(equal("/this-is/your/rootfs"))
// // 		})

// // 		context("when the command fails", func() {
// // 			beforeeach(func() {
// // 				fakecmdrunnerstdout = "could not find drax"
// // 				fakecmdrunnererr = errors.new("external-image-manager failure")
// // 			})

// // 			it("returns an error", func() {
// // 				_, _, err := externalimagemanager.create(
// // 					logger, "hello", rootfs_provider.spec{
// // 						rootfs: baseimage,
// // 					},
// // 				)

// // 				expect(err).to(matcherror(containsubstring("external image manager create failed")))
// // 				expect(err).to(matcherror(containsubstring("external-image-manager failure")))
// // 				expect(err).to(matcherror(containsubstring("could not find drax")))
// // 			})

// // 			it("returns the external-image-manager error output in the error", func() {
// // 				_, _, err := externalimagemanager.create(
// // 					logger, "hello", rootfs_provider.spec{
// // 						rootfs: baseimage,
// // 					},
// // 				)
// // 				expect(err).to(haveoccurred())

// // 				expect(logger).to(gbytes.say("could not find drax"))
// // 			})
// // 		})

// // 		context("when a rootfs is not provided in the rootfs_provider.spec", func() {
// // 			it("passes the default rootfs to the external-image-manager", func() {
// // 				_, _, err := externalimagemanager.create(
// // 					logger, "hello", rootfs_provider.spec{},
// // 				)
// // 				expect(err).notto(haveoccurred())

// // 				expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 				imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 				expect(imagemanagercmd.args[len(imagemanagercmd.args)-2]).to(equal(defaultbaseimage.string()))
// // 			})
// // 		})
// // 	})

// // 	describe("destroy", func() {
// // 		it("uses the correct external-image-manager binary", func() {
// // 			expect(externalimagemanager.destroy(
// // 				logger, "hello", "/store/0/images/123/rootfs",
// // 			)).to(succeed())
// // 			expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 			imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 			expect(imagemanagercmd.path).to(equal("/external-image-manager-bin"))
// // 		})

// // 		it("sets the correct extra arguments", func() {
// // 			expect(externalimagemanager.destroy(
// // 				logger, "hello", "/store/0/images/123/rootfs",
// // 			)).to(succeed())

// // 			expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 			imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 			expect(imagemanagercmd.args[1]).to(equal("bar"))
// // 			expect(imagemanagercmd.args[2]).to(equal("foo"))
// // 		})

// // 		describe("external-image-manager parameters", func() {
// // 			it("uses the correct external-image-manager delete command", func() {
// // 				expect(externalimagemanager.destroy(
// // 					logger, "hello", "/store/0/images/123/rootfs",
// // 				)).to(succeed())
// // 				expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 				imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 				expect(imagemanagercmd.args[3]).to(equal("delete"))
// // 			})

// // 			it("passes the correct image path to delete to/ the external-image-manager", func() {
// // 				expect(externalimagemanager.destroy(
// // 					logger, "hello", "/store/0/images/123/rootfs",
// // 				)).to(succeed())
// // 				expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 				imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 				expect(imagemanagercmd.args[len(imagemanagercmd.args)-1]).to(equal("/store/0/images/123"))
// // 			})
// // 		})

// // 		context("when the command fails", func() {
// // 			beforeeach(func() {
// // 				fakecmdrunnerstdout = "could not find drax"
// // 				fakecmdrunnererr = errors.new("external-image-manager failure")
// // 			})

// // 			it("returns an error", func() {
// // 				err := externalimagemanager.destroy(logger, "hello", "/store/0/images/123/rootfs")

// // 				expect(err).to(matcherror(containsubstring("external image manager destroy failed")))
// // 				expect(err).to(matcherror(containsubstring("external-image-manager failure")))
// // 				expect(err).to(matcherror(containsubstring("could not find drax")))
// // 			})

// // 			it("returns the external-image-manager error output in the error", func() {
// // 				expect(externalimagemanager.destroy(
// // 					logger, "hello", "/store/0/images/123/rootfs",
// // 				)).notto(succeed())

// // 				expect(logger).to(gbytes.say("could not find drax"))
// // 			})
// // 		})
// // 	})

// // 	describe("metrics", func() {
// // 		beforeeach(func() {
// // 			fakecmdrunnererr = nil
// // 			fakecmdrunnerstdout = `{"disk_usage": {"total_bytes_used": 1000, "exclusive_bytes_used": 2000}}`
// // 			fakecmdrunnerstderr = ""
// // 		})

// // 		it("uses the correct external-image-manager binary", func() {
// // 			_, err := externalimagemanager.metrics(logger, "", "/store/0/bundles/123/rootfs")
// // 			expect(err).tonot(haveoccurred())
// // 			expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 			imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 			expect(imagemanagercmd.path).to(equal("/external-image-manager-bin"))
// // 		})

// // 		it("sets the correct extra arguments", func() {
// // 			_, err := externalimagemanager.metrics(logger, "", "/store/0/bundles/123/rootfs")
// // 			expect(err).tonot(haveoccurred())

// // 			expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 			imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 			expect(imagemanagercmd.args[1]).to(equal("bar"))
// // 			expect(imagemanagercmd.args[2]).to(equal("foo"))
// // 		})

// // 		it("calls external-image-manager with the correct arguments", func() {
// // 			_, err := externalimagemanager.metrics(logger, "", "/store/0/bundles/123/rootfs")
// // 			expect(err).tonot(haveoccurred())
// // 			expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 			imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 			expect(imagemanagercmd.args[3]).to(equal("stats"))
// // 			expect(imagemanagercmd.args[4]).to(equal("/store/0/bundles/123"))
// // 		})

// // 		it("returns the correct containerdiskstat", func() {
// // 			stats, err := externalimagemanager.metrics(logger, "", "/store/0/bundles/123/rootfs")
// // 			expect(err).tonot(haveoccurred())

// // 			expect(stats.totalbytesused).to(equal(uint64(1000)))
// // 			expect(stats.exclusivebytesused).to(equal(uint64(2000)))
// // 		})

// // 		context("when the image plugin fails", func() {
// // 			it("returns an error", func() {
// // 				fakecmdrunnerstdout = "could not find drax"
// // 				fakecmdrunnererr = errors.new("failed to get metrics")
// // 				_, err := externalimagemanager.metrics(logger, "", "/store/0/bundles/123/rootfs")
// // 				expect(err).to(matcherror(containsubstring("failed to get metrics")))
// // 				expect(err).to(matcherror(containsubstring("could not find drax")))
// // 			})
// // 		})

// // 		context("when the image plugin output parsing fails", func() {
// // 			it("returns an error", func() {
// // 				fakecmdrunnerstdout = `{"silly" "json":"formating}"}}"`
// // 				_, err := externalimagemanager.metrics(logger, "", "/store/0/bundles/123/rootfs")
// // 				expect(err).to(matcherror(containsubstring("parsing metrics")))
// // 			})
// // 		})
// // 	})

// // 	describe("gc", func() {
// // 		var imagemanagercmd *exec.cmd

// // 		it("uses the correct external-image-manager binary", func() {
// // 			expect(externalimagemanager.gc(logger)).notto(haveoccurred())
// // 			imagemanagercmd = fakecommandrunner.executedcommands()[0]
// // 			expect(imagemanagercmd.path).to(equal("/external-image-manager-bin"))
// // 		})

// // 		it("sets the correct extra arguments", func() {
// // 			expect(externalimagemanager.gc(logger)).notto(haveoccurred())

// // 			expect(len(fakecommandrunner.executedcommands())).to(equal(1))
// // 			imagemanagercmd := fakecommandrunner.executedcommands()[0]

// // 			expect(imagemanagercmd.args[1]).to(equal("bar"))
// // 			expect(imagemanagercmd.args[2]).to(equal("foo"))
// // 		})

// // 		it("calls external image clean command", func() {
// // 			expect(externalimagemanager.gc(logger)).notto(haveoccurred())
// // 			imagemanagercmd = fakecommandrunner.executedcommands()[0]
// // 			expect(imagemanagercmd.args[3]).to(equal("clean"))
// // 		})

// // 		context("when the command fails", func() {
// // 			beforeeach(func() {
// // 				fakecmdrunnererr = errors.new("external-image-manager failure")
// // 				fakecmdrunnerstdout = "could not find drax"
// // 			})

// // 			it("returns an error", func() {
// // 				err := externalimagemanager.gc(logger)
// // 				expect(err).to(matcherror(containsubstring("external image manager clean failed")))
// // 				expect(err).to(matcherror(containsubstring("external-image-manager failure")))
// // 				expect(err).to(matcherror(containsubstring("could not find drax")))
// // 			})

// // 			it("forwards the external-image-manager error output", func() {
// // 				externalimagemanager.gc(logger)
// // 				expect(logger).to(gbytes.say("could not find drax"))
// // 			})
// // 		})
// // 	})
// // 	describe("logging", func() {
// // 		beforeeach(func() {
// // 			buffer := gbytes.newbuffer()
// // 			externallogger := lager.newlogger("external-plugin")
// // 			externallogger.registersink(lager.newwritersink(buffer, lager.debug))
// // 			externallogger.debug("debug-message", lager.data{"type": "debug"})
// // 			externallogger.info("info-message", lager.data{"type": "info"})
// // 			externallogger.error("error-message", errors.new("failed!"), lager.data{"type": "error"})

// // 			fakecmdrunnerstderr = string(buffer.contents())
// // 		})

// // 		context("create", func() {

// // 			it("relogs the image plugin logs", func() {
// // 				_, _, err := externalimagemanager.create(
// // 					logger, "hello", rootfs_provider.spec{
// // 						rootfs: baseimage,
// // 					},
// // 				)
// // 				expect(err).tonot(haveoccurred())

// // 				expect(logger).to(glager.containsequence(
// // 					glager.debug(
// // 						glager.message("external-image-manager.image-plugin-create.external-plugin.debug-message"),
// // 						glager.data("type", "debug"),
// // 					),
// // 					glager.info(
// // 						glager.message("external-image-manager.image-plugin-create.external-plugin.info-message"),
// // 						glager.data("type", "info"),
// // 					),
// // 					glager.error(
// // 						errors.new("failed!"),
// // 						glager.message("external-image-manager.image-plugin-create.external-plugin.error-message"),
// // 						glager.data("type", "error"),
// // 					),
// // 				))
// // 			})
// // 		})

// // 		context("destroy", func() {
// // 			it("relogs the image plugin logs", func() {
// // 				err := externalimagemanager.destroy(
// // 					logger, "hello", "/store/0/images/123/rootfs",
// // 				)

// // 				expect(err).tonot(haveoccurred())

// // 				expect(logger).to(glager.containsequence(
// // 					glager.debug(
// // 						glager.message("external-image-manager.image-plugin-destroy.external-plugin.debug-message"),
// // 						glager.data("type", "debug"),
// // 					),
// // 					glager.info(
// // 						glager.message("external-image-manager.image-plugin-destroy.external-plugin.info-message"),
// // 						glager.data("type", "info"),
// // 					),
// // 					glager.error(
// // 						errors.new("failed!"),
// // 						glager.message("external-image-manager.image-plugin-destroy.external-plugin.error-message"),
// // 						glager.data("type", "error"),
// // 					),
// // 				))
// // 			})
// // 		})

// // 		context("metrics", func() {
// // 			it("relogs the image plugin logs", func() {
// // 				fakecmdrunnerstdout = `{}`

// // 				_, err := externalimagemanager.metrics(
// // 					logger, "hello", "/store/0/images/123/rootfs",
// // 				)

// // 				expect(err).tonot(haveoccurred())

// // 				expect(logger).to(glager.containsequence(
// // 					glager.debug(
// // 						glager.message("external-image-manager.image-plugin-metrics.external-plugin.debug-message"),
// // 						glager.data("type", "debug"),
// // 					),
// // 					glager.info(
// // 						glager.message("external-image-manager.image-plugin-metrics.external-plugin.info-message"),
// // 						glager.data("type", "info"),
// // 					),
// // 					glager.error(
// // 						errors.new("failed!"),
// // 						glager.message("external-image-manager.image-plugin-metrics.external-plugin.error-message"),
// // 						glager.data("type", "error"),
// // 					),
// // 				))
// // 			})
// // 		})

// // 		context("gc", func() {
// // 			it("relogs the image plugin logs", func() {
// // 				err := externalimagemanager.gc(logger)

// // 				expect(err).tonot(haveoccurred())

// // 				expect(logger).to(glager.containsequence(
// // 					glager.debug(
// // 						glager.message("external-image-manager.image-plugin-gc.external-plugin.debug-message"),
// // 						glager.data("type", "debug"),
// // 					),
// // 					glager.info(
// // 						glager.message("external-image-manager.image-plugin-gc.external-plugin.info-message"),
// // 						glager.data("type", "info"),
// // 					),
// // 					glager.error(
// // 						errors.new("failed!"),
// // 						glager.message("external-image-manager.image-plugin-gc.external-plugin.error-message"),
// // 						glager.data("type", "error"),
// // 					),
// // 				))
// // 			})
// // 		})
// // 	})
// // })
