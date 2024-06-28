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

package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	internalapi "demo/pkg/api/cri"
	pb "demo/pkg/api/cri/v1"
	"github.com/docker/go-units"
	"github.com/sirupsen/logrus"
)

type statsOptions struct {
	// all containers
	all bool
	// id of container
	id string
	// podID of container
	podID string
	// sample is the duration for sampling cpu usage.
	sample time.Duration
	// labels are selectors for the sandbox
	labels map[string]string
	// output format
	output string
	// live watch
	watch bool
}

type containerStatsByID []*pb.ContainerStats

func (c containerStatsByID) Len() int      { return len(c) }
func (c containerStatsByID) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c containerStatsByID) Less(i, j int) bool {
	return c[i].Attributes.Id < c[j].Attributes.Id
}

func getContainerStats(ctx context.Context, client internalapi.RuntimeService, request *pb.ListContainerStatsRequest) (*pb.ListContainerStatsResponse, error) {
	logrus.Debugf("ListContainerStatsRequest: %v", request)
	r, err := client.ListContainerStats(context.TODO(), request.Filter)
	logrus.Debugf("ListContainerResponse: %v", r)
	if err != nil {
		return nil, err
	}
	sort.Sort(containerStatsByID(r))
	return &pb.ListContainerStatsResponse{Stats: r}, nil
}

func displayStats(ctx context.Context, client internalapi.RuntimeService, request *pb.ListContainerStatsRequest, display *display, opts statsOptions) error {
	r, err := getContainerStats(ctx, client, request)
	if err != nil {
		return err
	}
	switch opts.output {
	case "json":
		return outputProtobufObjAsJSON(r)
	case "yaml":
		return outputProtobufObjAsYAML(r)
	}
	oldStats := make(map[string]*pb.ContainerStats)
	for _, s := range r.GetStats() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		oldStats[s.Attributes.Id] = s
	}

	time.Sleep(opts.sample)

	r, err = getContainerStats(ctx, client, request)
	if err != nil {
		return err
	}

	display.AddRow([]string{columnContainer, columnName, columnCPU, columnMemory, columnDisk, columnInodes})
	for _, s := range r.GetStats() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		id := getTruncatedID(s.Attributes.Id, "")
		name := s.GetAttributes().GetMetadata().GetName()
		cpu := s.GetCpu().GetUsageCoreNanoSeconds().GetValue()
		mem := s.GetMemory().GetWorkingSetBytes().GetValue()
		disk := s.GetWritableLayer().GetUsedBytes().GetValue()
		inodes := s.GetWritableLayer().GetInodesUsed().GetValue()
		if !opts.all && cpu == 0 && mem == 0 {
			// Skip non-running container
			continue
		}
		old, ok := oldStats[s.Attributes.Id]
		if !ok {
			// Skip new container
			continue
		}
		var cpuPerc float64
		if cpu != 0 {
			// Only generate cpuPerc for running container
			duration := s.GetCpu().GetTimestamp() - old.GetCpu().GetTimestamp()
			if duration == 0 {
				return fmt.Errorf("cpu stat is not updated during sample")
			}
			cpuPerc = float64(cpu-old.GetCpu().GetUsageCoreNanoSeconds().GetValue()) / float64(duration) * 100
		}
		display.AddRow([]string{id, name, fmt.Sprintf("%.2f", cpuPerc), units.HumanSize(float64(mem)),
			units.HumanSize(float64(disk)), fmt.Sprintf("%d", inodes)})

	}
	display.ClearScreen()
	display.Flush()

	return nil
}
