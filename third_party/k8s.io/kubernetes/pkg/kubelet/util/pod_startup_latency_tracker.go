/*
Copyright 2022 The Kubernetes Authors.

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

package util

import (
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// PodStartupLatencyTracker records key moments for startup latency calculation,
// e.g. image pulling or pod observed running on watch.
type PodStartupLatencyTracker interface {
	ObservedPodOnWatch(pod *v1.Pod, when time.Time)
	RecordImageStartedPulling(podUID types.UID)
	RecordImageFinishedPulling(podUID types.UID)
	RecordStatusUpdated(pod *v1.Pod)
	DeletePodStartupState(podUID types.UID)
}

type perPodState struct {
	firstStartedPulling time.Time
	lastFinishedPulling time.Time
	// first time, when pod status changed into Running
	observedRunningTime time.Time
	// log, if pod latency was already Observed
	metricRecorded bool
}

// NewPodStartupLatencyTracker creates an instance of PodStartupLatencyTracker

// hasPodStartedSLO, check if for given pod, each container has been started at least once
//
// This should reflect "Pod startup latency SLI" definition
// ref: https://github.com/kubernetes/community/blob/master/sig-scalability/slos/pod_startup_latency.md
func hasPodStartedSLO(pod *v1.Pod) bool {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Running == nil || cs.State.Running.StartedAt.IsZero() {
			return false
		}
	}

	return true
}
