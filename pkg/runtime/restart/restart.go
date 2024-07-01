// Package restart enables containers to have labels added and monitored to
// keep the container's task running if it is killed.
//
// Setting the StatusLabel on a container instructs the restart monitor to keep
// that container's task in a specific status.
// Setting the LogPathLabel on a container will setup the task's IO to be redirected
// to a log file when running a task within the restart manager.
//
// The restart labels can be cleared off of a container using the WithNoRestarts Opt.
//
// The restart monitor has one option in the containerd config under the [plugins.restart]
// section.  `interval = "10s" sets the reconcile interval that the restart monitor checks
// for task state and reconciles the desired status for that task.
package restart

import (
	"fmt"
	"strconv"
	"strings"

	"demo/pkg/containerd"
	"github.com/sirupsen/logrus"
)

const (
	// StatusLabel sets the restart status label for a container
	StatusLabel = "containerd.io/restart.status"
	// LogURILabel sets the restart log uri label for a container
	LogURILabel = "containerd.io/restart.loguri"

	// PolicyLabel sets the restart policy label for a container
	PolicyLabel = "containerd.io/restart.policy"
	// CountLabel sets the restart count label for a container
	CountLabel = "containerd.io/restart.count"
	// ExplicitlyStoppedLabel sets the restart explicitly stopped label for a container
	ExplicitlyStoppedLabel = "containerd.io/restart.explicitly-stopped"

	// LogPathLabel sets the restart log path label for a container
	//
	// Deprecated(in release 1.5): use LogURILabel
	LogPathLabel = "containerd.io/restart.logpath"
)

// Policy represents the restart policies of a container.
type Policy struct {
	name              string
	maximumRetryCount int
}

// NewPolicy creates a restart policy with the specified name.
// supports the following restart policies:
// - no, Do not restart the container.
// - always, Always restart the container regardless of the exit status.
// - on-failure[:max-retries], Restart only if the container exits with a non-zero exit status.
// - unless-stopped, Always restart the container unless it is stopped.
func NewPolicy(policy string) (*Policy, error) {
	policySlice := strings.Split(policy, ":")
	var (
		err        error
		retryCount int
	)
	switch policySlice[0] {
	case "", "no", "always", "unless-stopped":
		policy = policySlice[0]
		if policy == "" {
			policy = "always"
		}
		if len(policySlice) > 1 {
			return nil, fmt.Errorf("restart policy %q not support max retry count", policySlice[0])
		}
	case "on-failure":
		policy = policySlice[0]
		if len(policySlice) > 1 {
			retryCount, err = strconv.Atoi(policySlice[1])
			if err != nil {
				return nil, fmt.Errorf("invalid max retry count: %s", policySlice[1])
			}
		}
	default:
		return nil, fmt.Errorf("restart policy %q not supported", policy)
	}
	return &Policy{
		name:              policy,
		maximumRetryCount: retryCount,
	}, nil
}

func (rp *Policy) String() string {
	if rp.maximumRetryCount > 0 {
		return fmt.Sprintf("%s:%d", rp.name, rp.maximumRetryCount)
	}
	return rp.name
}

func (rp *Policy) Name() string {
	return rp.name
}

func (rp *Policy) MaximumRetryCount() int {
	return rp.maximumRetryCount
}

// Reconcile reconciles the restart policy of a container.
func Reconcile(status containerd.Status, labels map[string]string) bool {
	rp, err := NewPolicy(labels[PolicyLabel])
	if err != nil {
		logrus.WithError(err).Error("policy reconcile")
		return false
	}
	switch rp.Name() {
	case "", "always":
		return true
	case "on-failure":
		restartCount, err := strconv.Atoi(labels[CountLabel])
		if err != nil && labels[CountLabel] != "" {
			logrus.WithError(err).Error("policy reconcile")
			return false
		}
		if status.ExitStatus != 0 && (rp.maximumRetryCount == 0 || restartCount < rp.maximumRetryCount) {
			return true
		}
	case "unless-stopped":
		explicitlyStopped, _ := strconv.ParseBool(labels[ExplicitlyStoppedLabel])
		if !explicitlyStopped {
			return true
		}
	}
	return false
}
