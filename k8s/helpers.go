package k8s

import (
	"errors"
	"fmt"

	"k8s.io/api/core/v1"
)

func mergeMaps(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

func int32ptr(i int) *int32 {
	u := int32(i)
	return &u
}

func int64ptr(i int) *int64 {
	u := int64(i)
	return &u
}

// The Kubernetes API expects multiple containers but we only ever need one.
func asMultipleContainers(container v1.Container) []v1.Container {
	return []v1.Container{container}
}

// Enforce our assumption that there's only ever exactly one container holding the app.
func assertSingleContainer(containers []v1.Container) {
	if len(containers) != 1 {
		message := fmt.Sprintf("Unexpectedly, container count is not 1 but %d.", len(containers))
		panic(errors.New(message))
	}
}

func toMap(envVars []v1.EnvVar) map[string]string {
	result := make(map[string]string)
	for _, env := range envVars {
		result[env.Name] = env.Value
	}
	return result
}
