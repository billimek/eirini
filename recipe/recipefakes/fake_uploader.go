// Code generated by counterfeiter. DO NOT EDIT.
package recipefakes

import (
	"sync"

	"code.cloudfoundry.org/eirini/recipe"
)

type FakeUploader struct {
	UploadStub        func(path, url string) error
	uploadMutex       sync.RWMutex
	uploadArgsForCall []struct {
		path string
		url  string
	}
	uploadReturns struct {
		result1 error
	}
	uploadReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeUploader) Upload(path string, url string) error {
	fake.uploadMutex.Lock()
	ret, specificReturn := fake.uploadReturnsOnCall[len(fake.uploadArgsForCall)]
	fake.uploadArgsForCall = append(fake.uploadArgsForCall, struct {
		path string
		url  string
	}{path, url})
	fake.recordInvocation("Upload", []interface{}{path, url})
	fake.uploadMutex.Unlock()
	if fake.UploadStub != nil {
		return fake.UploadStub(path, url)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.uploadReturns.result1
}

func (fake *FakeUploader) UploadCallCount() int {
	fake.uploadMutex.RLock()
	defer fake.uploadMutex.RUnlock()
	return len(fake.uploadArgsForCall)
}

func (fake *FakeUploader) UploadArgsForCall(i int) (string, string) {
	fake.uploadMutex.RLock()
	defer fake.uploadMutex.RUnlock()
	return fake.uploadArgsForCall[i].path, fake.uploadArgsForCall[i].url
}

func (fake *FakeUploader) UploadReturns(result1 error) {
	fake.UploadStub = nil
	fake.uploadReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeUploader) UploadReturnsOnCall(i int, result1 error) {
	fake.UploadStub = nil
	if fake.uploadReturnsOnCall == nil {
		fake.uploadReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.uploadReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeUploader) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.uploadMutex.RLock()
	defer fake.uploadMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeUploader) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ recipe.Uploader = new(FakeUploader)
