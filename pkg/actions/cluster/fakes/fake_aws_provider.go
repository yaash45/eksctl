// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"sync"

	"github.com/weaveworks/eksctl/pkg/actions/cluster"
	"github.com/weaveworks/eksctl/pkg/apis/eksctl.io/v1alpha5"
	"github.com/weaveworks/eksctl/pkg/eks"
)

type FakeProviderConstructor struct {
	Stub        func(*v1alpha5.ProviderConfig, *v1alpha5.ClusterConfig) (*eks.ClusterProvider, error)
	mutex       sync.RWMutex
	argsForCall []struct {
		arg1 *v1alpha5.ProviderConfig
		arg2 *v1alpha5.ClusterConfig
	}
	returns struct {
		result1 *eks.ClusterProvider
		result2 error
	}
	returnsOnCall map[int]struct {
		result1 *eks.ClusterProvider
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeProviderConstructor) Spy(arg1 *v1alpha5.ProviderConfig, arg2 *v1alpha5.ClusterConfig) (*eks.ClusterProvider, error) {
	fake.mutex.Lock()
	ret, specificReturn := fake.returnsOnCall[len(fake.argsForCall)]
	fake.argsForCall = append(fake.argsForCall, struct {
		arg1 *v1alpha5.ProviderConfig
		arg2 *v1alpha5.ClusterConfig
	}{arg1, arg2})
	stub := fake.Stub
	returns := fake.returns
	fake.recordInvocation("ProviderConstructor", []interface{}{arg1, arg2})
	fake.mutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return returns.result1, returns.result2
}

func (fake *FakeProviderConstructor) CallCount() int {
	fake.mutex.RLock()
	defer fake.mutex.RUnlock()
	return len(fake.argsForCall)
}

func (fake *FakeProviderConstructor) Calls(stub func(*v1alpha5.ProviderConfig, *v1alpha5.ClusterConfig) (*eks.ClusterProvider, error)) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.Stub = stub
}

func (fake *FakeProviderConstructor) ArgsForCall(i int) (*v1alpha5.ProviderConfig, *v1alpha5.ClusterConfig) {
	fake.mutex.RLock()
	defer fake.mutex.RUnlock()
	return fake.argsForCall[i].arg1, fake.argsForCall[i].arg2
}

func (fake *FakeProviderConstructor) Returns(result1 *eks.ClusterProvider, result2 error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.Stub = nil
	fake.returns = struct {
		result1 *eks.ClusterProvider
		result2 error
	}{result1, result2}
}

func (fake *FakeProviderConstructor) ReturnsOnCall(i int, result1 *eks.ClusterProvider, result2 error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.Stub = nil
	if fake.returnsOnCall == nil {
		fake.returnsOnCall = make(map[int]struct {
			result1 *eks.ClusterProvider
			result2 error
		})
	}
	fake.returnsOnCall[i] = struct {
		result1 *eks.ClusterProvider
		result2 error
	}{result1, result2}
}

func (fake *FakeProviderConstructor) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.mutex.RLock()
	defer fake.mutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeProviderConstructor) recordInvocation(key string, args []interface{}) {
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

var _ cluster.ProviderConstructor = new(FakeProviderConstructor).Spy
