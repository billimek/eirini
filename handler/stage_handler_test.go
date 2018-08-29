package handler_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/eirini/eirinifakes"
	. "code.cloudfoundry.org/eirini/handler"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
)

var _ = FDescribe("StageHandler", func() {

	var (
		//ts     *httptest.Server
		logger lager.Logger
		client *http.Client

		stagingClient *eirinifakes.FakeStager
		bifrost       *eirinifakes.FakeBifrost
		//responseRecorder   *httptest.ResponseRecorder
		stagingHandler     *Stage
		stagingRequestJSON string
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		client = &http.Client{}
		stagingClient = new(eirinifakes.FakeStager)
		bifrost = new(eirinifakes.FakeBifrost)
		stagingHandler = NewStageHandler(stagingClient, logger)
		stagingRequestJSON = `{"app_id":"myapp", "lifecycle":"kube-backend"}`
	})

	Context("when somthing happens", func() {
		It("should happen", func() {
			Expect(true).To(Equal(true))
		})
	})

})
