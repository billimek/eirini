package bifrost_test

import (
	"context"
	"errors"
	"net/http"

	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/eirini"
	"code.cloudfoundry.org/eirini/bifrost"
	"code.cloudfoundry.org/eirini/opi"
	"code.cloudfoundry.org/eirini/opi/opifakes"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/runtimeschema/cc_messages"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bifrost", func() {
	Context("Transfer", func() {
		It("simply converts and desires every single CCRequest to a LRP", func() {
			converted := make([]cc_messages.DesireAppRequestFromCC, 0)
			desired := make([][]opi.LRP, 0)
			bifrost := bifrost.Bifrost{
				Converter: bifrost.ConvertFunc(func(
					msg cc_messages.DesireAppRequestFromCC,
					regUrl string,
					regIP string,
					cfClient eirini.CfClient,
					client *http.Client,
					log lager.Logger,
				) opi.LRP {
					converted = append(converted, msg)
					return opi.LRP{Image: msg.DockerImageUrl}
				}),
				Desirer: opi.DesireFunc(func(_ context.Context, lrps []opi.LRP) error {
					desired = append(desired, lrps)
					return nil
				}),
			}

			err := bifrost.Transfer(context.Background(), []cc_messages.DesireAppRequestFromCC{
				{
					DockerImageUrl: "msg1",
				},
				{
					DockerImageUrl: "msg2",
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(converted).To(HaveLen(2))
			Expect(desired).To(HaveLen(1))
			Expect(desired[0]).To(HaveLen(2))
			Expect(desired[0][0].Image).To(Equal("msg1"))
			Expect(desired[0][1].Image).To(Equal("msg2"))
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
					opi.LRP{Name: "1234"},
					opi.LRP{Name: "5678"},
					opi.LRP{Name: "0213"},
				}
			})

			It("should translate []LRPs to []DesiredLRPSchedulingInfo", func() {
				desiredLRPSchedulingInfos, err := bfrst.List(context.Background())
				Expect(err).ToNot(HaveOccurred())

				Expect(desiredLRPSchedulingInfos[0].ProcessGuid).To(Equal("1234"))
				Expect(desiredLRPSchedulingInfos[1].ProcessGuid).To(Equal("5678"))
				Expect(desiredLRPSchedulingInfos[2].ProcessGuid).To(Equal("0213"))
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
})
