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
	"fmt"
	"os"
	"sync"
	"time"

	"demo/third_party/k8s.io/kubernetes/pkg/kubelet/cri/remote"
	"github.com/pborman/uuid"
	"gopkg.in/yaml.v3"

	. "github.com/onsi/ginkgo/v2"
)

var (
	//lock for uuid
	uuidLock sync.Mutex

	// lastUUID record last generated uuid from NewUUID()
	lastUUID uuid.UUID

	// the callbacks to run during BeforeSuite
	beforeSuiteCallbacks []func()

	// DefaultPodLabels are labels used by default in pods
	DefaultPodLabels map[string]string

	// DefaultContainerCommand is the default command used for containers
	DefaultContainerCommand []string

	// DefaultPauseCommand is the default command used for containers
	DefaultPauseCommand []string

	// DefaultLinuxPodLabels default pod labels for Linux
	DefaultLinuxPodLabels = map[string]string{}

	// DefaultLinuxContainerCommand default container command for Linux
	DefaultLinuxContainerCommand = []string{"top"}

	// DefaultLinuxPauseCommand default container command for Linux pause
	DefaultLinuxPauseCommand = []string{"sh", "-c", "top"}
)

const (
	// DefaultUIDPrefix is a default UID prefix of PodSandbox
	DefaultUIDPrefix string = "cri-test-uid"

	// DefaultNamespacePrefix is a default namespace prefix of PodSandbox
	DefaultNamespacePrefix string = "cri-test-namespace"

	// DefaultAttempt is a default attempt prefix of PodSandbox or container
	DefaultAttempt uint32 = 2
)

// LoadCRIClient creates a InternalAPIClient.
func LoadCRIClient() (*InternalAPIClient, error) {
	rService, err := remote.NewRemoteRuntimeService(
		TestContext.RuntimeServiceAddr,
		TestContext.RuntimeServiceTimeout,
		nil,
	)
	if err != nil {
		return nil, err
	}

	var imageServiceAddr = TestContext.ImageServiceAddr
	if imageServiceAddr == "" {
		// Fallback to runtime service endpoint
		imageServiceAddr = TestContext.RuntimeServiceAddr
	}
	iService, err := remote.NewRemoteImageService(imageServiceAddr, TestContext.ImageServiceTimeout, nil)
	if err != nil {
		return nil, err
	}

	return &InternalAPIClient{
		CRIRuntimeClient: rService,
		CRIImageClient:   iService,
	}, nil
}

func nowStamp() string {
	return time.Now().Format(time.StampMilli)
}

func log(level string, format string, args ...interface{}) {
	fmt.Fprintf(GinkgoWriter, nowStamp()+": "+level+": "+format+"\n", args...)
}

// Logf prints a info message.
func Logf(format string, args ...interface{}) {
	log("INFO", format, args...)
}

// LoadYamlFile attempts to load the given YAML file into the given struct.
func LoadYamlFile(filepath string, obj interface{}) error {
	Logf("Attempting to load YAML file %q into %+v", filepath, obj)
	fileContent, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("error reading %q file contents: %v", filepath, err)
	}

	err = yaml.Unmarshal(fileContent, obj)
	if err != nil {
		return fmt.Errorf("error unmarshalling %q YAML file: %v", filepath, err)
	}

	Logf("Successfully loaded YAML file %q into %+v", filepath, obj)
	return nil
}
