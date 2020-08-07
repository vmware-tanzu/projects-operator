// Copyright 2020-Present VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

// Code generated by counterfeiter. DO NOT EDIT.
package webhookfakes

import (
	"sync"

	"github.com/pivotal/projects-operator/api/v1alpha1"
	"github.com/pivotal/projects-operator/pkg/webhook"
	v1 "k8s.io/api/authentication/v1"
)

type FakeProjectFilterer struct {
	FilterProjectsStub        func([]v1alpha1.Project, v1.UserInfo) []string
	filterProjectsMutex       sync.RWMutex
	filterProjectsArgsForCall []struct {
		arg1 []v1alpha1.Project
		arg2 v1.UserInfo
	}
	filterProjectsReturns struct {
		result1 []string
	}
	filterProjectsReturnsOnCall map[int]struct {
		result1 []string
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeProjectFilterer) FilterProjects(arg1 []v1alpha1.Project, arg2 v1.UserInfo) []string {
	var arg1Copy []v1alpha1.Project
	if arg1 != nil {
		arg1Copy = make([]v1alpha1.Project, len(arg1))
		copy(arg1Copy, arg1)
	}
	fake.filterProjectsMutex.Lock()
	ret, specificReturn := fake.filterProjectsReturnsOnCall[len(fake.filterProjectsArgsForCall)]
	fake.filterProjectsArgsForCall = append(fake.filterProjectsArgsForCall, struct {
		arg1 []v1alpha1.Project
		arg2 v1.UserInfo
	}{arg1Copy, arg2})
	fake.recordInvocation("FilterProjects", []interface{}{arg1Copy, arg2})
	fake.filterProjectsMutex.Unlock()
	if fake.FilterProjectsStub != nil {
		return fake.FilterProjectsStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.filterProjectsReturns
	return fakeReturns.result1
}

func (fake *FakeProjectFilterer) FilterProjectsCallCount() int {
	fake.filterProjectsMutex.RLock()
	defer fake.filterProjectsMutex.RUnlock()
	return len(fake.filterProjectsArgsForCall)
}

func (fake *FakeProjectFilterer) FilterProjectsCalls(stub func([]v1alpha1.Project, v1.UserInfo) []string) {
	fake.filterProjectsMutex.Lock()
	defer fake.filterProjectsMutex.Unlock()
	fake.FilterProjectsStub = stub
}

func (fake *FakeProjectFilterer) FilterProjectsArgsForCall(i int) ([]v1alpha1.Project, v1.UserInfo) {
	fake.filterProjectsMutex.RLock()
	defer fake.filterProjectsMutex.RUnlock()
	argsForCall := fake.filterProjectsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeProjectFilterer) FilterProjectsReturns(result1 []string) {
	fake.filterProjectsMutex.Lock()
	defer fake.filterProjectsMutex.Unlock()
	fake.FilterProjectsStub = nil
	fake.filterProjectsReturns = struct {
		result1 []string
	}{result1}
}

func (fake *FakeProjectFilterer) FilterProjectsReturnsOnCall(i int, result1 []string) {
	fake.filterProjectsMutex.Lock()
	defer fake.filterProjectsMutex.Unlock()
	fake.FilterProjectsStub = nil
	if fake.filterProjectsReturnsOnCall == nil {
		fake.filterProjectsReturnsOnCall = make(map[int]struct {
			result1 []string
		})
	}
	fake.filterProjectsReturnsOnCall[i] = struct {
		result1 []string
	}{result1}
}

func (fake *FakeProjectFilterer) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.filterProjectsMutex.RLock()
	defer fake.filterProjectsMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeProjectFilterer) recordInvocation(key string, args []interface{}) {
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

var _ webhook.ProjectFilterer = new(FakeProjectFilterer)
