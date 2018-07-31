package integration_test

import (
	"code.cloudfoundry.org/eirini/k8s"
	"code.cloudfoundry.org/eirini/models/cf"
	"code.cloudfoundry.org/eirini/opi"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/apps/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Instance", func() {

	var instanceManager k8s.InstanceManager

	cleanupStatefulSet := func(appName string) {
		backgroundPropagation := metav1.DeletePropagationBackground
		clientset.AppsV1beta2().StatefulSets(namespace).Delete(appName, &metav1.DeleteOptions{PropagationPolicy: &backgroundPropagation})
	}

	listStatefulSets := func() []v1beta2.StatefulSet {
		list, err := clientset.AppsV1beta2().StatefulSets(namespace).List(metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		return list.Items
	}

	JustBeforeEach(func() {
		instanceManager = k8s.NewInstanceManager(
			clientset,
			namespace,
			k8s.UseStatefulSets,
		)
	})

	Context("When creating an LRP", func() {
		var lrp *opi.LRP

		BeforeEach(func() {
			lrp = createLRP("odin")
		})

		AfterEach(func() {
			cleanupStatefulSet(lrp.Name)
			Eventually(listStatefulSets, timeout).Should(BeEmpty())
		})

		It("should create an StatefulSet with an associated pod", func() {
			err := instanceManager.Create(lrp)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() error {
				_, err = clientset.CoreV1().Pods(namespace).Get(
					lrp.Name+"-0",
					metav1.GetOptions{},
				)
				return err
			}, timeout).ShouldNot(HaveOccurred())
		})
	})
})

func createLRP(name string) *opi.LRP {
	return &opi.LRP{
		Name: name,
		Command: []string{
			"/bin/sh",
			"-c",
			"while true; do echo hello; sleep 10;done",
		},
		TargetInstances: 1,
		Image:           "busybox",
		Metadata: map[string]string{
			cf.ProcessGUID: name,
		},
	}
}
