package k8s

import (
	"code.cloudfoundry.org/eirini/models/cf"
	"code.cloudfoundry.org/eirini/opi"
	"k8s.io/api/apps/v1beta2"
	av1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ev2 "k8s.io/client-go/kubernetes/typed/apps/v1beta2"
)

//go:generate counterfeiter . StatefulSetManager
type StatefulSetManager interface {
	ListLRPs(namespace string) ([]opi.LRP, error)
	Delete(appName, namespace string) error
}

type statefulSetManager struct {
	client   kubernetes.Interface
	endpoint string
}

func NewStatefulsetManager(client kubernetes.Interface) StatefulSetManager {
	return &statefulSetManager{
		client: client,
	}
}

func (m *statefulSetManager) ListLRPs(namespace string) ([]opi.LRP, error) {
	statefulsets, err := m.statefulSets(namespace).List(av1.ListOptions{})
	if err != nil {
		return nil, err
	}

	lrps := statefulSetstoLRPs(statefulsets)

	return lrps, nil
}

func (m *statefulSetManager) Delete(appName, namespace string) error {
	backgroundPropagation := av1.DeletePropagationBackground
	return m.statefulSets(namespace).Delete(appName, &av1.DeleteOptions{PropagationPolicy: &backgroundPropagation})
}

func (m *statefulSetManager) statefulSets(namespace string) ev2.StatefulSetInterface {
	return m.client.AppsV1beta2().StatefulSets(namespace)
}

func statefulSetstoLRPs(statefulSets *v1beta2.StatefulSetList) []opi.LRP {
	lrps := []opi.LRP{}
	for _, d := range statefulSets.Items {
		lrp := opi.LRP{
			Metadata: map[string]string{
				cf.ProcessGUID: d.Annotations[cf.ProcessGUID],
				cf.LastUpdated: d.Annotations[cf.LastUpdated],
			},
		}
		lrps = append(lrps, lrp)
	}
	return lrps
}
