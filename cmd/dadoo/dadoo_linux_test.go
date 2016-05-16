package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cloudfoundry-incubator/goci"
	"github.com/cloudfoundry-incubator/guardian/rundmc"
	"github.com/cloudfoundry-incubator/guardian/rundmc/dadoo"
	"github.com/cloudfoundry/gunk/command_runner/linux_command_runner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = FDescribe("Dadoo", func() {
	var (
		bundlePath string
		bundle     *goci.Bndl
	)

	BeforeEach(func() {
		setupCgroups()

		var err error
		bundlePath, err = ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())

		Expect(syscall.Mount("tmpfs", bundlePath, "tmpfs", 0, "")).To(Succeed())

		cmd := exec.Command("runc", "spec")
		cmd.Dir = bundlePath
		Expect(cmd.Run()).To(Succeed())

		loader := &goci.BndlLoader{}
		bundle, err = loader.Load(bundlePath)
		Expect(err).NotTo(HaveOccurred())

		cp, err := gexec.Start(exec.Command("cp", "-a", os.Getenv("GARDEN_TEST_ROOTFS"), filepath.Join(bundlePath, "root")), GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(cp, "2m").Should(gexec.Exit(0))

		bundle = bundle.
			WithProcess(specs.Process{Args: []string{"/bin/sh", "-c", "echo hello; exit 12"}, Cwd: "/"}).
			WithRootFS(path.Join(bundlePath, "root"))

		SetDefaultEventuallyTimeout(5 * time.Second)
	})

	JustBeforeEach(func() {
		Expect(bundle.Save(path.Join(bundlePath))).To(Succeed())
	})

	// AfterEach(func() {
	// 	Expect(syscall.Unmount(bundlePath, syscall.MNT_DETACH)).To(Succeed())
	// })

	Describe("dadoo run", func() {
		It("should return the exit code of the container process", func() {
			sess, err := gexec.Start(exec.Command(dadooBinPath, "run", "runc", bundlePath, filepath.Base(bundlePath)), GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(sess).Should(gexec.Exit(12))
		})

		It("should write logs to the requested file", func() {
			_, err := gexec.Start(exec.Command(dadooBinPath, "-log", path.Join(bundlePath, "foo.log"), "run", "runc", bundlePath, filepath.Base(bundlePath)), GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(filepath.Join(bundlePath, "foo.log")).Should(BeAnExistingFile())
		})

		It("should delete the container state correctly when it exits", func() {
			sess, err := gexec.Start(exec.Command(dadooBinPath, "run", "runc", bundlePath, filepath.Base(bundlePath)), GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(gexec.Exit())

			state, err := gexec.Start(exec.Command("runc", "state", filepath.Base(bundlePath)), GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(state).Should(gexec.Exit(1))
		})

		Describe("returning runc's exit code on fd3", func() {
			var pipeR, pipeW *os.File

			BeforeEach(func() {
				var err error
				pipeR, pipeW, err = os.Pipe()
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when launching succeeds", func() {
				It("should return 0 on fd3", func() {
					cmd := exec.Command(dadooBinPath, "run", "runc", bundlePath, filepath.Base(bundlePath))
					cmd.ExtraFiles = []*os.File{
						pipeW,
					}

					_, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())

					fd3 := make(chan byte)
					go func() {
						b := make([]byte, 1)
						pipeR.Read(b)

						fd3 <- b[0]
					}()

					Eventually(fd3).Should(Receive(BeEquivalentTo(0)))
				})

				Context("when running a long-running command", func() {
					BeforeEach(func() {
						bundle = bundle.WithProcess(specs.Process{
							Args: []string{
								"/bin/sh", "-c", "sleep 60",
							},
							Cwd: "/",
						})
					})

					It("should be able to be watched by WaitWatcher", func() {
						cmd := exec.Command(dadooBinPath, "run", "runc", bundlePath, filepath.Base(bundlePath))
						cmd.ExtraFiles = []*os.File{
							pipeW,
						}

						sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())

						fd3 := make(chan byte)
						go func() {
							b := make([]byte, 1)
							pipeR.Read(b)

							fd3 <- b[0]
						}()
						Eventually(fd3).Should(Receive(BeEquivalentTo(0)))

						ww := &dadoo.WaitWatcher{}
						ch, err := ww.Wait(filepath.Join(bundlePath, "exit.sock"))
						Expect(err).NotTo(HaveOccurred())
						Consistently(ch).ShouldNot(BeClosed())

						killCmd := exec.Command("runc", "kill", filepath.Base(bundlePath), "KILL")
						killCmd.Dir = bundlePath
						killSess, err := gexec.Start(killCmd, GinkgoWriter, GinkgoWriter)
						Expect(err).NotTo(HaveOccurred())
						Eventually(killSess).Should(gexec.Exit(0))

						Eventually(ch).Should(BeClosed())
						Expect(sess).NotTo(gexec.Exit(0))
					})
				})
			})

			Context("when launching fails", func() {
				BeforeEach(func() {
					bundle = bundle.WithRootFS("/path/to/nothing/at/all/potato")
				})

				It("should return runc's exit status on fd3", func() {
					cmd := exec.Command(dadooBinPath, "run", "runc", bundlePath, filepath.Base(bundlePath))
					cmd.ExtraFiles = []*os.File{
						pipeW,
					}

					_, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())

					fd3 := make(chan byte)
					go func() {
						b := make([]byte, 1)
						pipeR.Read(b)

						fd3 <- b[0]
					}()

					Eventually(fd3).Should(Receive(BeEquivalentTo(1)))
				})
			})

			It("it exits 2 and writes an error to fd3 if runc start fails", func() {
				cmd := exec.Command(dadooBinPath, "run", "some-binary-that-doesnt-exist", bundlePath, filepath.Base(bundlePath))
				cmd.ExtraFiles = []*os.File{
					pipeW,
				}

				sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(sess).Should(gexec.Exit(2))

				fd3 := make(chan byte)
				go func() {
					b := make([]byte, 1)
					pipeR.Read(b)

					fd3 <- b[0]
				}()

				Eventually(fd3).Should(Receive(BeEquivalentTo(2)))
			})
		})
	})

	Describe("dadoo exec", func() {
		var (
			containerSess *gexec.Session
		)

		BeforeEach(func() {
			bundle = bundle.WithProcess(specs.Process{
				Args: []string{"/bin/sleep", "1000"},
				Cwd:  "/",
			})
		})

		JustBeforeEach(func() {
			var err error

			r, w, err := os.Pipe()
			defer r.Close()
			defer w.Close()
			Expect(err).NotTo(HaveOccurred())

			cmd := exec.Command(dadooBinPath, "run", "runc", bundlePath, filepath.Base(bundlePath))
			cmd.ExtraFiles = []*os.File{w}
			containerSess, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			buffer := make([]byte, 1)
			_, err = r.Read(buffer)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			cmd := exec.Command("runc", "kill", filepath.Base(bundlePath), "KILL")
			cmd.Dir = bundlePath
			sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(gexec.Exit(0))

			Eventually(containerSess).Should(gexec.Exit())
		})

		Describe("using named pipes for stdout/in/error", func() {
			var (
				stdoutPipe, stdinPipe, stderrPipe string
			)

			BeforeEach(func() {
				tmp, err := ioutil.TempDir("", "dadoopipetest")
				Expect(err).NotTo(HaveOccurred())

				stdoutPipe = filepath.Join(tmp, "stdout.pipe")
				Expect(syscall.Mkfifo(stdoutPipe, 0)).To(Succeed())

				stderrPipe = filepath.Join(tmp, "stderr.pipe")
				Expect(syscall.Mkfifo(stderrPipe, 0)).To(Succeed())

				stdinPipe = filepath.Join(tmp, "stdin.pipe")
				Expect(syscall.Mkfifo(stdinPipe, 0)).To(Succeed())
			})

			AfterEach(func() {
				Expect(os.Remove(stdoutPipe)).To(Succeed())
				Expect(os.Remove(stderrPipe)).To(Succeed())
				Expect(os.Remove(stdinPipe)).To(Succeed())
			})

			It("should be able to pass named pipes to use as stdin and stdout", func() {
				cmd := exec.Command(dadooBinPath, "-stdout", stdoutPipe, "-stdin", stdinPipe, "-stderr", stderrPipe, "exec", "runc", bundlePath, filepath.Base(bundlePath))
				cmd.Stdin = processJSONReader(specs.Process{
					Args: []string{"/bin/sh", "-c", "cat <&0"},
					Cwd:  "/",
				})
				_, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				stdinP, err := os.OpenFile(stdinPipe, os.O_WRONLY, 0600)
				Expect(err).NotTo(HaveOccurred())

				stdoutP, err := os.Open(stdoutPipe)
				Expect(err).NotTo(HaveOccurred())

				_, err = os.Open(stderrPipe)
				Expect(err).NotTo(HaveOccurred())

				stdinP.WriteString("hello")
				Expect(stdinP.Close()).To(Succeed())

				stdout := make([]byte, len("hello"))
				_, err = stdoutP.Read(stdout)
				Expect(err).NotTo(HaveOccurred())

				Expect(string(stdout)).To(Equal("hello"))
			})

			Context("when output goes to stderr", func() {
				It("should be able to pass named pipes to use as stdin and stderr", func() {
					cmd := exec.Command(dadooBinPath, "-stdout", stdoutPipe, "-stdin", stdinPipe, "-stderr", stderrPipe, "exec", "runc", bundlePath, filepath.Base(bundlePath))
					cmd.Stdin = processJSONReader(specs.Process{
						Args: []string{"/bin/sh", "-c", "cat <&0 1>&2"},
						Cwd:  "/",
					})
					_, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())

					stdinP, err := os.OpenFile(stdinPipe, os.O_WRONLY, 0600)
					Expect(err).NotTo(HaveOccurred())

					_, err = os.Open(stdoutPipe)
					Expect(err).NotTo(HaveOccurred())

					stderrP, err := os.Open(stderrPipe)
					Expect(err).NotTo(HaveOccurred())

					stdinP.WriteString("hello")
					Expect(stdinP.Close()).To(Succeed())

					stderr := make([]byte, len("hello"))
					_, err = stderrP.Read(stderr)
					Expect(err).NotTo(HaveOccurred())

					Expect(string(stderr)).To(Equal("hello"))
				})
			})

			Describe("with a TTY", func() {
				It("assigns a TTY to the process", func() {
					cmd := exec.Command(dadooBinPath, "-tty", "-stdout", stdoutPipe, "exec", "runc", bundlePath, filepath.Base(bundlePath))
					cmd.Stdin = processJSONReader(specs.Process{
						Args:     []string{"/bin/sh", "-c", "stty -a"},
						Cwd:      "/",
						Terminal: true,
					})
					sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())

					stdoutP, err := os.Open(stdoutPipe)
					Expect(err).NotTo(HaveOccurred())

					Eventually(sess).Should(gexec.Exit(0))

					stdout := make([]byte, 1024)
					_, err = stdoutP.Read(stdout)
					Expect(err).NotTo(HaveOccurred())

					Expect(string(stdout)).To(ContainSubstring("speed 38400"))
				})

				It("forwards stdout and stdin to the pipes", func() {
					cmd := exec.Command(dadooBinPath, "-tty", "-stdout", stdoutPipe, "-stdin", stdinPipe, "exec", "runc", bundlePath, filepath.Base(bundlePath))
					cmd.Stdin = processJSONReader(specs.Process{
						Args: []string{"/bin/sh", "-c", "cat <&0"},
						Cwd:  "/",
					})
					_, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())

					stdinP, err := os.OpenFile(stdinPipe, os.O_WRONLY, 0600)
					Expect(err).NotTo(HaveOccurred())

					stdoutP, err := os.Open(stdoutPipe)
					Expect(err).NotTo(HaveOccurred())

					stdinP.WriteString("hello")
					Expect(stdinP.Close()).To(Succeed())

					stdout := make([]byte, len("hello"))
					_, err = stdoutP.Read(stdout)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(stdout)).To(Equal("hello"))
				})
			})
		})

		It("does not leak any files in the bundle directory", func() {
			entries, err := ioutil.ReadDir(bundlePath)
			Expect(err).NotTo(HaveOccurred())

			cmd := exec.Command(dadooBinPath, "exec", "runc", bundlePath, filepath.Base(bundlePath))
			cmd.Stdin = processJSONReader(specs.Process{
				Args: []string{"/bin/sh", "-c", "exit 0"},
				Cwd:  "/",
			})
			sess, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Eventually(sess).Should(gexec.Exit(0))

			newEntries, err := ioutil.ReadDir(bundlePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(len(newEntries)))
		})
	})
})

func mustOpen(path string) io.ReadCloser {
	r, err := os.Open(path)
	Expect(err).NotTo(HaveOccurred())

	return r
}

func setupCgroups() error {
	logger := lagertest.NewTestLogger("test")
	runner := linux_command_runner.New()

	starter := rundmc.NewStarter(logger, mustOpen("/proc/cgroups"), mustOpen("/proc/self/cgroup"), path.Join(os.TempDir(), fmt.Sprintf("cgroups-%d", GinkgoParallelNode())), runner)

	return starter.Start()
}

func processJSONReader(spec specs.Process) io.Reader {
	data, err := json.Marshal(spec)
	Expect(err).NotTo(HaveOccurred())

	return bytes.NewBuffer(data)
}
