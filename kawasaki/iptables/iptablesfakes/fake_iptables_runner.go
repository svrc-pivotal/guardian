// This file was generated by counterfeiter
package iptablesfakes

import (
	"os/exec"
	"sync"

	"github.com/cloudfoundry-incubator/guardian/kawasaki/iptables"
)

type FakeIPTablesRunner struct {
	RunStub        func(action string, cmd *exec.Cmd) error
	runMutex       sync.RWMutex
	runArgsForCall []struct {
		action string
		cmd    *exec.Cmd
	}
	runReturns struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeIPTablesRunner) Run(action string, cmd *exec.Cmd) error {
	fake.runMutex.Lock()
	fake.runArgsForCall = append(fake.runArgsForCall, struct {
		action string
		cmd    *exec.Cmd
	}{action, cmd})
	fake.recordInvocation("Run", []interface{}{action, cmd})
	fake.runMutex.Unlock()
	if fake.RunStub != nil {
		return fake.RunStub(action, cmd)
	} else {
		return fake.runReturns.result1
	}
}

func (fake *FakeIPTablesRunner) RunCallCount() int {
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	return len(fake.runArgsForCall)
}

func (fake *FakeIPTablesRunner) RunArgsForCall(i int) (string, *exec.Cmd) {
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	return fake.runArgsForCall[i].action, fake.runArgsForCall[i].cmd
}

func (fake *FakeIPTablesRunner) RunReturns(result1 error) {
	fake.RunStub = nil
	fake.runReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeIPTablesRunner) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	return fake.invocations
}

func (fake *FakeIPTablesRunner) recordInvocation(key string, args []interface{}) {
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

var _ iptables.IPTablesRunner = new(FakeIPTablesRunner)