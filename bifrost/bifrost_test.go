package bifrost_test

import (
	"context"
	"errors"

	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/eirini"
	"code.cloudfoundry.org/eirini/bifrost"
	"code.cloudfoundry.org/eirini/bifrost/bifrostfakes"
	"code.cloudfoundry.org/eirini/opi"
	"code.cloudfoundry.org/eirini/opi/opifakes"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bifrost", func() {
	FContext("Transfer", func() {
		var (
			err     error
			bfrst   eirini.Bifrost
			request eirini.DesireLRPRequest
		)

		BeforeEach(func() {

			cnvrtr := new(bifrostfakes.FakeConverter)
			dsrr := new(opifakes.FakeDesirer)

			cnvrtr.convertMessageReturns(desiredLRP)
		})

		JustBeforeEach(func() {
			bfrst = &bifrost.Bifrost{
				Converter: cnvrtr,
				Desirer:   dsrr,
			}
			err = bfrst.Transfer(context.Background(), request)
		})

		Context("when lrp is desired succesfully", func() {

			BefureEach(func() {

			})

			It("should not return an error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})

	})

	Context("List", func() {
		var (
			opiClient *opifakes.FakeDesirer
			lager     lager.Logger
			bfrst     bifrost.Bifrost
			lrps      []opi.LRP
		)

		BeforeEach(func() {
			opiClient = new(opifakes.FakeDesirer)
			lager = lagertest.NewTestLogger("bifrost-test")
			bfrst = bifrost.Bifrost{
				Desirer: opiClient,
				Logger:  lager,
			}
		})

		JustBeforeEach(func() {
			opiClient.ListReturns(lrps, nil)
		})

		Context("When listing running LRPs", func() {

			BeforeEach(func() {
				lrps = []opi.LRP{
					opi.LRP{Name: "1234", Metadata: map[string]string{"process_guid": "abcd"}},
					opi.LRP{Name: "5678", Metadata: map[string]string{"process_guid": "efgh"}},
					opi.LRP{Name: "0213", Metadata: map[string]string{"process_guid": "ijkl"}},
				}
			})

			It("should translate []LRPs to []DesiredLRPSchedulingInfo", func() {
				desiredLRPSchedulingInfos, err := bfrst.List(context.Background())
				Expect(err).ToNot(HaveOccurred())

				Expect(desiredLRPSchedulingInfos[0].ProcessGuid).To(Equal("abcd"))
				Expect(desiredLRPSchedulingInfos[1].ProcessGuid).To(Equal("efgh"))
				Expect(desiredLRPSchedulingInfos[2].ProcessGuid).To(Equal("ijkl"))
			})
		})

		Context("When no running LRPs exist", func() {

			BeforeEach(func() {
				lrps = []opi.LRP{}
			})

			It("should return an empty list of DesiredLRPSchedulingInfo", func() {
				desiredLRPSchedulingInfos, err := bfrst.List(context.Background())
				Expect(err).ToNot(HaveOccurred())

				Expect(len(desiredLRPSchedulingInfos)).To(Equal(0))
			})
		})

		Context("When an error occurs", func() {

			JustBeforeEach(func() {
				opiClient.ListReturns(nil, errors.New("arrgh"))
			})

			It("should return a meaningful errormessage", func() {
				_, err := bfrst.List(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("failed to list desired LRPs"))
			})
		})
	})

	Context("Update an app", func() {

		var (
			bfrst         bifrost.Bifrost
			opiClient     *opifakes.FakeDesirer
			lager         lager.Logger
			updateRequest models.UpdateDesiredLRPRequest
			err           error
		)

		BeforeEach(func() {
			updateRequest = models.UpdateDesiredLRPRequest{
				ProcessGuid: "app_name",
			}
			opiClient = new(opifakes.FakeDesirer)

			lager = lagertest.NewTestLogger("bifrost-update-test")
		})

		JustBeforeEach(func() {
			bfrst = bifrost.Bifrost{
				Desirer: opiClient,
				Logger:  lager,
			}

			err = bfrst.Update(context.Background(), updateRequest)
		})

		Context("when the app exists", func() {

			BeforeEach(func() {
				lrp := opi.LRP{
					Name:            "app_name",
					TargetInstances: 2,
				}
				opiClient.GetReturns(&lrp, nil)
			})

			Context("with instance count modified", func() {

				BeforeEach(func() {
					updatedInstances := int32(5)
					updateRequest.Update = &models.DesiredLRPUpdate{Instances: &updatedInstances}
					opiClient.UpdateReturns(nil)
				})

				It("should get the existing LRP", func() {
					Expect(opiClient.GetCallCount()).To(Equal(1))
					_, appName := opiClient.GetArgsForCall(0)
					Expect(appName).To(Equal("app_name"))
				})

				It("should submit the updated LRP", func() {
					Expect(opiClient.UpdateCallCount()).To(Equal(1))
					_, lrp := opiClient.UpdateArgsForCall(0)
					Expect(lrp.Name).To(Equal("app_name"))
					Expect(lrp.TargetInstances).To(Equal(int(*updateRequest.Update.Instances)))
				})

				It("should not return an error", func() {
					Expect(err).ToNot(HaveOccurred())
				})

				Context("when the update fails", func() {
					BeforeEach(func() {
						opiClient.UpdateReturns(errors.New("failed to update app"))
					})

					It("should propagate the error", func() {
						Expect(err).To(HaveOccurred())
					})
				})
			})
		})

		Context("when the app does not exist", func() {

			BeforeEach(func() {
				opiClient.GetReturns(nil, errors.New("app does not exist"))
			})

			It("should try to get the LRP", func() {
				Expect(opiClient.GetCallCount()).To(Equal(1))
				_, appName := opiClient.GetArgsForCall(0)
				Expect(appName).To(Equal("app_name"))

			})

			It("should not submit anything to be updated", func() {
				Expect(opiClient.UpdateCallCount()).To(Equal(0))
			})

			It("should propagate the error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("get an App", func() {
		var (
			bfrst      bifrost.Bifrost
			opiClient  *opifakes.FakeDesirer
			lager      lager.Logger
			desiredLRP *models.DesiredLRP
			lrp        *opi.LRP
		)

		BeforeEach(func() {
			opiClient = new(opifakes.FakeDesirer)

			lager = lagertest.NewTestLogger("bifrost-update-test")
		})

		JustBeforeEach(func() {
			bfrst = bifrost.Bifrost{
				Desirer: opiClient,
				Logger:  lager,
			}

			desiredLRP = bfrst.Get(context.Background(), "app_name")
		})

		Context("when the app exists", func() {
			BeforeEach(func() {
				lrp = &opi.LRP{
					Name:            "app_name",
					TargetInstances: 5,
				}

				opiClient.GetReturns(lrp, nil)
			})

			It("should use the desirer to get the lrp", func() {
				Expect(opiClient.GetCallCount()).To(Equal(1))
				_, guid := opiClient.GetArgsForCall(0)
				Expect(guid).To(Equal("app_name"))
			})

			It("should return a DesiredLRP", func() {
				Expect(desiredLRP).ToNot(BeNil())
				Expect(desiredLRP.ProcessGuid).To(Equal("app_name"))
				Expect(desiredLRP.Instances).To(Equal(int32(5)))
			})
		})

		Context("when the app does not exist", func() {
			BeforeEach(func() {
				opiClient.GetReturns(nil, errors.New("Failed to get LRP"))
			})

			It("should return an error", func() {
				Expect(opiClient.GetCallCount()).To(Equal(1))
				Expect(desiredLRP).To(BeNil())
			})
		})

	})
})
