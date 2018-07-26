package k8s

import (
	"code.cloudfoundry.org/eirini"
	"code.cloudfoundry.org/eirini/opi"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//go:generate counterfeiter . ServiceManager
type ServiceManager interface {
	Create(lrp *opi.LRP) error
	Delete(appName string) error
}

type serviceManager struct {
	client    kubernetes.Interface
	namespace string
}

func NewServiceManager(namespace string, client kubernetes.Interface) ServiceManager {
	return &serviceManager{
		client:    client,
		namespace: namespace,
	}
}

func (m *serviceManager) Delete(appName string) error {
	serviceName := eirini.GetInternalServiceName(appName)
	return m.client.CoreV1().Services(m.namespace).Delete(serviceName, &meta_v1.DeleteOptions{})
}

func (m *serviceManager) Create(lrp *opi.LRP) error {
	return nil
}
