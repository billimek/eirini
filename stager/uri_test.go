package stager_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "code.cloudfoundry.org/eirini/stager"
)

var _ = Describe("HostnameEncoder", func() {

	var (
		encoder     URIEncoder
		replacement string
		url         string
		modified    string
	)

	JustBeforeEach(func() {
		encoder = &HostnameEncoder{
			Replacement: replacement,
		}
		modified = encoder.Encode(url)
	})

	assertReplaced := func(expected string) {
		Expect(modified).To(Equal(expected))
	}

	Context("When the url matches the regex", func() {

		BeforeEach(func() {
			replacement = "burr.io"
			url = "http://internal_user:kxdfemyfqjgs1dnh0efx@cloud-controller-ng.service.cf.internal:9022/internal/v3/staging/91809228-63ee-44d6-92d3-4f623583b577/build_completed?start=true"
		})

		It("should replace the hostname", func() {
			assertReplaced("http://internal_user:kxdfemyfqjgs1dnh0efx@burr.io:9022/internal/v3/staging/91809228-63ee-44d6-92d3-4f623583b577/build_completed?start=true")
		})

		Context("and the replacement contains http scheme", func() {

			BeforeEach(func() {
				replacement = "http://burr.io"
			})

			It("should replace the hostname", func() {
				assertReplaced("http://internal_user:kxdfemyfqjgs1dnh0efx@burr.io:9022/internal/v3/staging/91809228-63ee-44d6-92d3-4f623583b577/build_completed?start=true")
			})
		})

		Context("and the replacement contains https scheme", func() {

			BeforeEach(func() {
				replacement = "https://burr.io"
			})

			It("should replace the hostname", func() {
				assertReplaced("http://internal_user:kxdfemyfqjgs1dnh0efx@burr.io:9022/internal/v3/staging/91809228-63ee-44d6-92d3-4f623583b577/build_completed?start=true")
			})
		})

	})

	Context("When the url doesn't match the regex", func() {

		BeforeEach(func() {
			url = "http://cloud-controller-ng.service.cf.internal:9022/internal/v3/staging/91809228-63ee-44d6-92d3-4f623583b577/build_completed?start=true"
		})

		It("should return the same url", func() {
			Expect(modified).To(Equal(url))
		})

	})
})
