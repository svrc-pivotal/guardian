// This file was generated by counterfeiter
package kawasakifakes

import (
	"sync"

	"github.com/cloudfoundry-incubator/guardian/kawasaki"
)

type FakeDnsResolvConfFactory struct {
	CreateDNSResolvConfigurerStub        func(pid int, cfg kawasaki.NetworkConfig) kawasaki.DnsResolvConfigurer
	createDNSResolvConfigurerMutex       sync.RWMutex
	createDNSResolvConfigurerArgsForCall []struct {
		pid int
		cfg kawasaki.NetworkConfig
	}
	createDNSResolvConfigurerReturns struct {
		result1 kawasaki.DnsResolvConfigurer
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeDnsResolvConfFactory) CreateDNSResolvConfigurer(pid int, cfg kawasaki.NetworkConfig) kawasaki.DnsResolvConfigurer {
	fake.createDNSResolvConfigurerMutex.Lock()
	fake.createDNSResolvConfigurerArgsForCall = append(fake.createDNSResolvConfigurerArgsForCall, struct {
		pid int
		cfg kawasaki.NetworkConfig
	}{pid, cfg})
	fake.recordInvocation("CreateDNSResolvConfigurer", []interface{}{pid, cfg})
	fake.createDNSResolvConfigurerMutex.Unlock()
	if fake.CreateDNSResolvConfigurerStub != nil {
		return fake.CreateDNSResolvConfigurerStub(pid, cfg)
	} else {
		return fake.createDNSResolvConfigurerReturns.result1
	}
}

func (fake *FakeDnsResolvConfFactory) CreateDNSResolvConfigurerCallCount() int {
	fake.createDNSResolvConfigurerMutex.RLock()
	defer fake.createDNSResolvConfigurerMutex.RUnlock()
	return len(fake.createDNSResolvConfigurerArgsForCall)
}

func (fake *FakeDnsResolvConfFactory) CreateDNSResolvConfigurerArgsForCall(i int) (int, kawasaki.NetworkConfig) {
	fake.createDNSResolvConfigurerMutex.RLock()
	defer fake.createDNSResolvConfigurerMutex.RUnlock()
	return fake.createDNSResolvConfigurerArgsForCall[i].pid, fake.createDNSResolvConfigurerArgsForCall[i].cfg
}

func (fake *FakeDnsResolvConfFactory) CreateDNSResolvConfigurerReturns(result1 kawasaki.DnsResolvConfigurer) {
	fake.CreateDNSResolvConfigurerStub = nil
	fake.createDNSResolvConfigurerReturns = struct {
		result1 kawasaki.DnsResolvConfigurer
	}{result1}
}

func (fake *FakeDnsResolvConfFactory) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.createDNSResolvConfigurerMutex.RLock()
	defer fake.createDNSResolvConfigurerMutex.RUnlock()
	return fake.invocations
}

func (fake *FakeDnsResolvConfFactory) recordInvocation(key string, args []interface{}) {
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

var _ kawasaki.DnsResolvConfFactory = new(FakeDnsResolvConfFactory)