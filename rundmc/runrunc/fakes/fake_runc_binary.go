// This file was generated by counterfeiter
package fakes

import (
	"os/exec"
	"sync"

	"github.com/cloudfoundry-incubator/guardian/rundmc/runrunc"
)

type FakeRuncBinary struct {
	StartCommandStub        func(path, id string) *exec.Cmd
	startCommandMutex       sync.RWMutex
	startCommandArgsForCall []struct {
		path string
		id   string
	}
	startCommandReturns struct {
		result1 *exec.Cmd
	}
	ExecCommandStub        func(id, processJSONPath, pidFilePath string) *exec.Cmd
	execCommandMutex       sync.RWMutex
	execCommandArgsForCall []struct {
		id              string
		processJSONPath string
		pidFilePath     string
	}
	execCommandReturns struct {
		result1 *exec.Cmd
	}
	KillCommandStub        func(id, signal string) *exec.Cmd
	killCommandMutex       sync.RWMutex
	killCommandArgsForCall []struct {
		id     string
		signal string
	}
	killCommandReturns struct {
		result1 *exec.Cmd
	}
}

func (fake *FakeRuncBinary) StartCommand(path string, id string) *exec.Cmd {
	fake.startCommandMutex.Lock()
	fake.startCommandArgsForCall = append(fake.startCommandArgsForCall, struct {
		path string
		id   string
	}{path, id})
	fake.startCommandMutex.Unlock()
	if fake.StartCommandStub != nil {
		return fake.StartCommandStub(path, id)
	} else {
		return fake.startCommandReturns.result1
	}
}

func (fake *FakeRuncBinary) StartCommandCallCount() int {
	fake.startCommandMutex.RLock()
	defer fake.startCommandMutex.RUnlock()
	return len(fake.startCommandArgsForCall)
}

func (fake *FakeRuncBinary) StartCommandArgsForCall(i int) (string, string) {
	fake.startCommandMutex.RLock()
	defer fake.startCommandMutex.RUnlock()
	return fake.startCommandArgsForCall[i].path, fake.startCommandArgsForCall[i].id
}

func (fake *FakeRuncBinary) StartCommandReturns(result1 *exec.Cmd) {
	fake.StartCommandStub = nil
	fake.startCommandReturns = struct {
		result1 *exec.Cmd
	}{result1}
}

func (fake *FakeRuncBinary) ExecCommand(id string, processJSONPath string, pidFilePath string) *exec.Cmd {
	fake.execCommandMutex.Lock()
	fake.execCommandArgsForCall = append(fake.execCommandArgsForCall, struct {
		id              string
		processJSONPath string
		pidFilePath     string
	}{id, processJSONPath, pidFilePath})
	fake.execCommandMutex.Unlock()
	if fake.ExecCommandStub != nil {
		return fake.ExecCommandStub(id, processJSONPath, pidFilePath)
	} else {
		return fake.execCommandReturns.result1
	}
}

func (fake *FakeRuncBinary) ExecCommandCallCount() int {
	fake.execCommandMutex.RLock()
	defer fake.execCommandMutex.RUnlock()
	return len(fake.execCommandArgsForCall)
}

func (fake *FakeRuncBinary) ExecCommandArgsForCall(i int) (string, string, string) {
	fake.execCommandMutex.RLock()
	defer fake.execCommandMutex.RUnlock()
	return fake.execCommandArgsForCall[i].id, fake.execCommandArgsForCall[i].processJSONPath, fake.execCommandArgsForCall[i].pidFilePath
}

func (fake *FakeRuncBinary) ExecCommandReturns(result1 *exec.Cmd) {
	fake.ExecCommandStub = nil
	fake.execCommandReturns = struct {
		result1 *exec.Cmd
	}{result1}
}

func (fake *FakeRuncBinary) KillCommand(id string, signal string) *exec.Cmd {
	fake.killCommandMutex.Lock()
	fake.killCommandArgsForCall = append(fake.killCommandArgsForCall, struct {
		id     string
		signal string
	}{id, signal})
	fake.killCommandMutex.Unlock()
	if fake.KillCommandStub != nil {
		return fake.KillCommandStub(id, signal)
	} else {
		return fake.killCommandReturns.result1
	}
}

func (fake *FakeRuncBinary) KillCommandCallCount() int {
	fake.killCommandMutex.RLock()
	defer fake.killCommandMutex.RUnlock()
	return len(fake.killCommandArgsForCall)
}

func (fake *FakeRuncBinary) KillCommandArgsForCall(i int) (string, string) {
	fake.killCommandMutex.RLock()
	defer fake.killCommandMutex.RUnlock()
	return fake.killCommandArgsForCall[i].id, fake.killCommandArgsForCall[i].signal
}

func (fake *FakeRuncBinary) KillCommandReturns(result1 *exec.Cmd) {
	fake.KillCommandStub = nil
	fake.killCommandReturns = struct {
		result1 *exec.Cmd
	}{result1}
}

var _ runrunc.RuncBinary = new(FakeRuncBinary)
