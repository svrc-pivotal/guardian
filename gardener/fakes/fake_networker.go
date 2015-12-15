// This file was generated by counterfeiter
package fakes

import (
	"sync"

	"github.com/cloudfoundry-incubator/guardian/gardener"
	"github.com/pivotal-golang/lager"
)

type FakeNetworker struct {
	NetworkStub        func(log lager.Logger, handle, spec string) (string, error)
	networkMutex       sync.RWMutex
	networkArgsForCall []struct {
		log    lager.Logger
		handle string
		spec   string
	}
	networkReturns struct {
		result1 string
		result2 error
	}
	CapacityStub        func() uint64
	capacityMutex       sync.RWMutex
	capacityArgsForCall []struct{}
	capacityReturns     struct {
		result1 uint64
	}
	DestroyStub        func(log lager.Logger, handle string) error
	destroyMutex       sync.RWMutex
	destroyArgsForCall []struct {
		log    lager.Logger
		handle string
	}
	destroyReturns struct {
		result1 error
	}
}

func (fake *FakeNetworker) Network(log lager.Logger, handle string, spec string) (string, error) {
	fake.networkMutex.Lock()
	fake.networkArgsForCall = append(fake.networkArgsForCall, struct {
		log    lager.Logger
		handle string
		spec   string
	}{log, handle, spec})
	fake.networkMutex.Unlock()
	if fake.NetworkStub != nil {
		return fake.NetworkStub(log, handle, spec)
	} else {
		return fake.networkReturns.result1, fake.networkReturns.result2
	}
}

func (fake *FakeNetworker) NetworkCallCount() int {
	fake.networkMutex.RLock()
	defer fake.networkMutex.RUnlock()
	return len(fake.networkArgsForCall)
}

func (fake *FakeNetworker) NetworkArgsForCall(i int) (lager.Logger, string, string) {
	fake.networkMutex.RLock()
	defer fake.networkMutex.RUnlock()
	return fake.networkArgsForCall[i].log, fake.networkArgsForCall[i].handle, fake.networkArgsForCall[i].spec
}

func (fake *FakeNetworker) NetworkReturns(result1 string, result2 error) {
	fake.NetworkStub = nil
	fake.networkReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeNetworker) Capacity() uint64 {
	fake.capacityMutex.Lock()
	fake.capacityArgsForCall = append(fake.capacityArgsForCall, struct{}{})
	fake.capacityMutex.Unlock()
	if fake.CapacityStub != nil {
		return fake.CapacityStub()
	} else {
		return fake.capacityReturns.result1
	}
}

func (fake *FakeNetworker) CapacityCallCount() int {
	fake.capacityMutex.RLock()
	defer fake.capacityMutex.RUnlock()
	return len(fake.capacityArgsForCall)
}

func (fake *FakeNetworker) CapacityReturns(result1 uint64) {
	fake.CapacityStub = nil
	fake.capacityReturns = struct {
		result1 uint64
	}{result1}
}

func (fake *FakeNetworker) Destroy(log lager.Logger, handle string) error {
	fake.destroyMutex.Lock()
	fake.destroyArgsForCall = append(fake.destroyArgsForCall, struct {
		log    lager.Logger
		handle string
	}{log, handle})
	fake.destroyMutex.Unlock()
	if fake.DestroyStub != nil {
		return fake.DestroyStub(log, handle)
	} else {
		return fake.destroyReturns.result1
	}
}

func (fake *FakeNetworker) DestroyCallCount() int {
	fake.destroyMutex.RLock()
	defer fake.destroyMutex.RUnlock()
	return len(fake.destroyArgsForCall)
}

func (fake *FakeNetworker) DestroyArgsForCall(i int) (lager.Logger, string) {
	fake.destroyMutex.RLock()
	defer fake.destroyMutex.RUnlock()
	return fake.destroyArgsForCall[i].log, fake.destroyArgsForCall[i].handle
}

func (fake *FakeNetworker) DestroyReturns(result1 error) {
	fake.DestroyStub = nil
	fake.destroyReturns = struct {
		result1 error
	}{result1}
}

var _ gardener.Networker = new(FakeNetworker)
