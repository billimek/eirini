package k8s_test

import (
	"code.cloudfoundry.org/eirini"
	"code.cloudfoundry.org/eirini/models/cf"
	"code.cloudfoundry.org/eirini/opi"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	. "code.cloudfoundry.org/eirini/k8s"
)

var _ = Describe("Service", func() {

	var (
		fakeClient     kubernetes.Interface
		serviceManager ServiceManager
	)

	const (
		namespace = "midgard"
	)

	JustBeforeEach(func() {
		fakeClient = fake.NewSimpleClientset()
		serviceManager = NewServiceManager(fakeClient, namespace)
	})

	Context("When exposing an existing LRP", func() {

		var (
			lrp *opi.LRP
			err error
		)

		BeforeEach(func() {
			lrp = createLRP("baldur", "54321.0")
		})

		Context("When creating a usual service", func() {

			JustBeforeEach(func() {
				err = serviceManager.Create(lrp)
			})

			It("should not fail", func() {
				Expect(err).ToNot(HaveOccurred())
			})

			It("should create a service", func() {
				serviceName := eirini.GetInternalServiceName("baldur")
				service, err := fakeClient.CoreV1().Services(namespace).Get(serviceName, meta.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(service).To(Equal(toService(lrp, namespace)))
			})

			Context("When recreating a existing service", func() {

				BeforeEach(func() {
					lrp = createLRP("baldur", "54321.0")
				})

				JustBeforeEach(func() {
					err = serviceManager.Create(lrp)
				})

				It("should return an error", func() {
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("When creating a headless service", func() {

			JustBeforeEach(func() {
				err = serviceManager.CreateHeadless(lrp)
			})

			It("should not fail", func() {
				Expect(err).ToNot(HaveOccurred())
			})

			It("should create a service", func() {
				serviceName := eirini.GetInternalHeadlessServiceName("baldur")
				service, err := fakeClient.CoreV1().Services(namespace).Get(serviceName, meta.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(service).To(Equal(toHeadlessService(lrp, namespace)))
			})

			Context("When recreating a existing service", func() {

				BeforeEach(func() {
					lrp = createLRP("baldur", "54321.0")
				})

				JustBeforeEach(func() {
					err = serviceManager.CreateHeadless(lrp)
				})

				It("should return an error", func() {
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("When deleting a service", func() {

			var lrps []*opi.LRP

			BeforeEach(func() {
				lrps = []*opi.LRP{
					createLRP("odin", "1234.5"),
					createLRP("thor", "4567.8"),
					createLRP("mimir", "9012.3"),
				}
			})

			JustBeforeEach(func() {
				for _, l := range lrps {
					fakeClient.CoreV1().Services(namespace).Create(toService(l, namespace))
				}
			})

			It("deletes the service", func() {
				err := serviceManager.Delete("odin")
				Expect(err).ToNot(HaveOccurred())

				services, _ := fakeClient.CoreV1().Services(namespace).List(meta.ListOptions{})
				Expect(services.Items).To(HaveLen(2))
				Expect(getServicesNames(services)).To(ConsistOf(eirini.GetInternalServiceName("mimir"), eirini.GetInternalServiceName("thor")))
			})

			Context("when the service does not exist", func() {
				It("returns an error", func() {
					err := serviceManager.Delete("non-existing")
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})

func getServicesNames(services *v1.ServiceList) []string {
	serviceNames := []string{}
	for _, s := range services.Items {
		serviceNames = append(serviceNames, s.Name)
	}
	return serviceNames
}

func toService(lrp *opi.LRP, namespace string) *v1.Service {
	service := &v1.Service{
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name: "service",
					Port: 8080,
				},
			},
			Selector: map[string]string{
				"name": lrp.Name,
			},
		},
	}

	service.Name = eirini.GetInternalServiceName(lrp.Name)
	service.Namespace = namespace
	service.Labels = map[string]string{
		"name": lrp.Name,
	}

	service.Annotations = map[string]string{
		"routes": lrp.Metadata[cf.VcapAppUris],
	}

	return service
}

func toHeadlessService(lrp *opi.LRP, namespace string) *v1.Service {
	service := &v1.Service{
		Spec: v1.ServiceSpec{
			ClusterIP: "None",
			Ports: []v1.ServicePort{
				{
					Name: "service",
					Port: 8080,
				},
			},
			Selector: map[string]string{
				"name": lrp.Name,
			},
		},
	}

	service.Name = eirini.GetInternalHeadlessServiceName(lrp.Name)
	service.Namespace = namespace
	service.Labels = map[string]string{
		"name": lrp.Name,
	}

	return service
}
