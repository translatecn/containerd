/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	internalapi "demo/over/api/cri"

	. "github.com/onsi/gomega"
)

// Framework will keep a client for you.
type Framework struct {
	// CRI client
	CRIClient *InternalAPIClient
}

// InternalAPIClient is the CRI client.
type InternalAPIClient struct {
	CRIRuntimeClient internalapi.RuntimeService
	CRIImageClient   internalapi.ImageManagerService
}

// BeforeEach gets a client
func (f *Framework) BeforeEach() {
	if f.CRIClient == nil {
		c, err := LoadCRIClient()
		Expect(err).NotTo(HaveOccurred())
		f.CRIClient = c
	}
}

// AfterEach clean resources
func (f *Framework) AfterEach() {
	f.CRIClient = nil
}
