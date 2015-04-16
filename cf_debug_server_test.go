package cf_debug_server_test

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	cf_debug_server "github.com/cloudfoundry-incubator/cf-debug-server"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("CF Debug Server", func() {
	var (
		logBuf *gbytes.Buffer
		sink   *lager.ReconfigurableSink

		address string
		process ifrit.Process
	)

	BeforeEach(func() {
		address = "127.0.0.1:10003"
		logBuf = gbytes.NewBuffer()
		sink = lager.NewReconfigurableSink(
			lager.NewWriterSink(logBuf, lager.DEBUG),
			// permit no logging by default, for log reconfiguration below
			lager.FATAL+1,
		)
	})

	AfterEach(func() {
		ginkgomon.Interrupt(process)
	})

	Describe("AddFlags", func() {
		It("adds flags to the flagset", func() {
			flags := flag.NewFlagSet("test", flag.ContinueOnError)
			cf_debug_server.AddFlags(flags)

			f := flags.Lookup(cf_debug_server.DebugFlag)
			Ω(f).ShouldNot(BeNil())
		})
	})

	Describe("DebugAddress", func() {
		Context("when flags are not added", func() {
			It("returns the empty string", func() {
				flags := flag.NewFlagSet("test", flag.ContinueOnError)
				Ω(cf_debug_server.DebugAddress(flags)).Should(Equal(""))
			})
		})

		Context("when flags are added", func() {
			var flags *flag.FlagSet
			BeforeEach(func() {
				flags = flag.NewFlagSet("test", flag.ContinueOnError)
				cf_debug_server.AddFlags(flags)
			})

			Context("when set", func() {
				It("returns the address", func() {
					flags.Parse([]string{"-debugAddr", address})

					Ω(cf_debug_server.DebugAddress(flags)).Should(Equal(address))
				})
			})

			Context("when not set", func() {
				It("returns the empty string", func() {
					Ω(cf_debug_server.DebugAddress(flags)).Should(Equal(""))
				})
			})
		})
	})

	Describe("Run", func() {
		It("serves debug information", func() {
			var err error
			process, err = cf_debug_server.Run(address, sink)
			Ω(err).ShouldNot(HaveOccurred())

			debugResponse, err := http.Get(fmt.Sprintf("http://%s/debug/pprof/goroutine", address))
			Ω(err).ShouldNot(HaveOccurred())

			debugInfo, err := ioutil.ReadAll(debugResponse.Body)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(debugInfo).Should(ContainSubstring("goroutine profile: total"))

		})

		Context("when the address is already in use", func() {
			It("returns an error", func() {
				_, err := net.Listen("tcp", address)
				Ω(err).ShouldNot(HaveOccurred())

				process, err = cf_debug_server.Run(address, sink)
				Ω(err).Should(HaveOccurred())
				Ω(err).Should(BeAssignableToTypeOf(&net.OpError{}))
				netErr := err.(*net.OpError)
				Ω(netErr.Op).Should(Equal("listen"))
			})
		})
	})
})
