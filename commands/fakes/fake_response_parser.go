// This file was generated by counterfeiter
package fakes

import (
	"sync"

	"github.com/cloudfoundry-incubator/diego-enabler/commands"
	"github.com/cloudfoundry-incubator/diego-enabler/models"
)

type FakeResponseParser struct {
	ParseStub        func([]byte) (models.Applications, error)
	parseMutex       sync.RWMutex
	parseArgsForCall []struct {
		arg1 []byte
	}
	parseReturns struct {
		result1 models.Applications
		result2 error
	}
}

func (fake *FakeResponseParser) Parse(arg1 []byte) (models.Applications, error) {
	fake.parseMutex.Lock()
	fake.parseArgsForCall = append(fake.parseArgsForCall, struct {
		arg1 []byte
	}{arg1})
	fake.parseMutex.Unlock()
	if fake.ParseStub != nil {
		return fake.ParseStub(arg1)
	} else {
		return fake.parseReturns.result1, fake.parseReturns.result2
	}
}

func (fake *FakeResponseParser) ParseCallCount() int {
	fake.parseMutex.RLock()
	defer fake.parseMutex.RUnlock()
	return len(fake.parseArgsForCall)
}

func (fake *FakeResponseParser) ParseArgsForCall(i int) []byte {
	fake.parseMutex.RLock()
	defer fake.parseMutex.RUnlock()
	return fake.parseArgsForCall[i].arg1
}

func (fake *FakeResponseParser) ParseReturns(result1 models.Applications, result2 error) {
	fake.ParseStub = nil
	fake.parseReturns = struct {
		result1 models.Applications
		result2 error
	}{result1, result2}
}

var _ commands.ResponseParser = new(FakeResponseParser)
