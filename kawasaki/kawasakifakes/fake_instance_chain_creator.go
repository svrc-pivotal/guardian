// This file was generated by counterfeiter
package kawasakifakes

import (
	"net"
	"sync"

	"code.cloudfoundry.org/guardian/kawasaki"
	"code.cloudfoundry.org/lager"
)

type FakeInstanceChainCreator struct {
	CreateStub        func(logger lager.Logger, handle, instanceChain, bridgeName string, ip net.IP, network *net.IPNet) error
	createMutex       sync.RWMutex
	createArgsForCall []struct {
		logger        lager.Logger
		handle        string
		instanceChain string
		bridgeName    string
		ip            net.IP
		network       *net.IPNet
	}
	createReturns struct {
		result1 error
	}
	createReturnsOnCall map[int]struct {
		result1 error
	}
	DestroyStub        func(logger lager.Logger, instanceChain string) error
	destroyMutex       sync.RWMutex
	destroyArgsForCall []struct {
		logger        lager.Logger
		instanceChain string
	}
	destroyReturns struct {
		result1 error
	}
	destroyReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeInstanceChainCreator) Create(logger lager.Logger, handle string, instanceChain string, bridgeName string, ip net.IP, network *net.IPNet) error {
	fake.createMutex.Lock()
	ret, specificReturn := fake.createReturnsOnCall[len(fake.createArgsForCall)]
	fake.createArgsForCall = append(fake.createArgsForCall, struct {
		logger        lager.Logger
		handle        string
		instanceChain string
		bridgeName    string
		ip            net.IP
		network       *net.IPNet
	}{logger, handle, instanceChain, bridgeName, ip, network})
	fake.recordInvocation("Create", []interface{}{logger, handle, instanceChain, bridgeName, ip, network})
	fake.createMutex.Unlock()
	if fake.CreateStub != nil {
		return fake.CreateStub(logger, handle, instanceChain, bridgeName, ip, network)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.createReturns.result1
}

func (fake *FakeInstanceChainCreator) CreateCallCount() int {
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	return len(fake.createArgsForCall)
}

func (fake *FakeInstanceChainCreator) CreateArgsForCall(i int) (lager.Logger, string, string, string, net.IP, *net.IPNet) {
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	return fake.createArgsForCall[i].logger, fake.createArgsForCall[i].handle, fake.createArgsForCall[i].instanceChain, fake.createArgsForCall[i].bridgeName, fake.createArgsForCall[i].ip, fake.createArgsForCall[i].network
}

func (fake *FakeInstanceChainCreator) CreateReturns(result1 error) {
	fake.CreateStub = nil
	fake.createReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeInstanceChainCreator) CreateReturnsOnCall(i int, result1 error) {
	fake.CreateStub = nil
	if fake.createReturnsOnCall == nil {
		fake.createReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.createReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeInstanceChainCreator) Destroy(logger lager.Logger, instanceChain string) error {
	fake.destroyMutex.Lock()
	ret, specificReturn := fake.destroyReturnsOnCall[len(fake.destroyArgsForCall)]
	fake.destroyArgsForCall = append(fake.destroyArgsForCall, struct {
		logger        lager.Logger
		instanceChain string
	}{logger, instanceChain})
	fake.recordInvocation("Destroy", []interface{}{logger, instanceChain})
	fake.destroyMutex.Unlock()
	if fake.DestroyStub != nil {
		return fake.DestroyStub(logger, instanceChain)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.destroyReturns.result1
}

func (fake *FakeInstanceChainCreator) DestroyCallCount() int {
	fake.destroyMutex.RLock()
	defer fake.destroyMutex.RUnlock()
	return len(fake.destroyArgsForCall)
}

func (fake *FakeInstanceChainCreator) DestroyArgsForCall(i int) (lager.Logger, string) {
	fake.destroyMutex.RLock()
	defer fake.destroyMutex.RUnlock()
	return fake.destroyArgsForCall[i].logger, fake.destroyArgsForCall[i].instanceChain
}

func (fake *FakeInstanceChainCreator) DestroyReturns(result1 error) {
	fake.DestroyStub = nil
	fake.destroyReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeInstanceChainCreator) DestroyReturnsOnCall(i int, result1 error) {
	fake.DestroyStub = nil
	if fake.destroyReturnsOnCall == nil {
		fake.destroyReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.destroyReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeInstanceChainCreator) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.createMutex.RLock()
	defer fake.createMutex.RUnlock()
	fake.destroyMutex.RLock()
	defer fake.destroyMutex.RUnlock()
	return fake.invocations
}

func (fake *FakeInstanceChainCreator) recordInvocation(key string, args []interface{}) {
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

var _ kawasaki.InstanceChainCreator = new(FakeInstanceChainCreator)
