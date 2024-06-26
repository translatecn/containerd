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
	internalapi "demo/over/api/cri"
	pb "demo/over/api/cri/v1"
	"errors"
	"fmt"
	"github.com/docker/go-units"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"strings"
)

type imageByRef []*pb.Image

func (a imageByRef) Len() int      { return len(a) }
func (a imageByRef) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a imageByRef) Less(i, j int) bool {
	if len(a[i].RepoTags) > 0 && len(a[j].RepoTags) > 0 {
		return a[i].RepoTags[0] < a[j].RepoTags[0]
	}
	if len(a[i].RepoDigests) > 0 && len(a[j].RepoDigests) > 0 {
		return a[i].RepoDigests[0] < a[j].RepoDigests[0]
	}
	return a[i].Id < a[j].Id
}

var imageStatusCommand = &cli.Command{
	Name:                   "inspecti",
	Usage:                  "Return the status of one or more images",
	ArgsUsage:              "IMAGE-ID [IMAGE-ID...]",
	UseShortOptionHandling: true,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output format, One of: json|yaml|go-template|table",
		},
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "Do not show verbose information",
		},
		&cli.StringFlag{
			Name:  "template",
			Usage: "The template string is only used when output is go-template; The Template format is golang template",
		},
	},
	Action: func(c *cli.Context) error {
		if c.NArg() == 0 {
			return cli.ShowSubcommandHelp(c)
		}
		imageClient, err := getImageService(c)
		if err != nil {
			return err
		}

		verbose := !(c.Bool("quiet"))
		output := c.String("output")
		if output == "" { // default to json output
			output = "json"
		}
		tmplStr := c.String("template")
		for i := 0; i < c.NArg(); i++ {
			id := c.Args().Get(i)

			r, err := ImageStatus(imageClient, id, verbose)
			if err != nil {
				return fmt.Errorf("image status for %q request: %w", id, err)
			}
			image := r.Image
			if image == nil {
				return fmt.Errorf("no such image %q present", id)
			}

			status, err := protobufObjectToJSON(r.Image)
			if err != nil {
				return fmt.Errorf("marshal status to json for %q: %w", id, err)
			}
			switch output {
			case "json", "yaml", "go-template":
				if err := outputStatusInfo(status, r.Info, output, tmplStr); err != nil {
					return fmt.Errorf("output status for %q: %w", id, err)
				}
				continue
			case "table": // table output is after this switch block
			default:
				return fmt.Errorf("output option cannot be %s", output)
			}

			// otherwise output in table format
			fmt.Printf("ID: %s\n", image.Id)
			for _, tag := range image.RepoTags {
				fmt.Printf("Tag: %s\n", tag)
			}
			for _, digest := range image.RepoDigests {
				fmt.Printf("Digest: %s\n", digest)
			}
			size := units.HumanSizeWithPrecision(float64(image.GetSize_()), 3)
			fmt.Printf("Size: %s\n", size)
			if verbose {
				fmt.Printf("Info: %v\n", r.GetInfo())
			}
		}

		return nil
	},
}

var removeImageCommand = &cli.Command{
	Name:                   "rmi",
	Usage:                  "Remove one or more images",
	ArgsUsage:              "IMAGE-ID [IMAGE-ID...]",
	UseShortOptionHandling: true,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "all",
			Aliases: []string{"a"},
			Usage:   "Remove all images",
		},
		&cli.BoolFlag{
			Name:    "prune",
			Aliases: []string{"q"},
			Usage:   "Remove all unused images",
		},
	},
	Action: func(cliCtx *cli.Context) error {
		imageClient, err := getImageService(cliCtx)
		if err != nil {
			return err
		}

		ids := map[string]bool{}
		for _, id := range cliCtx.Args().Slice() {
			logrus.Debugf("User specified image to be removed: %v", id)
			ids[id] = true
		}

		all := cliCtx.Bool("all")
		prune := cliCtx.Bool("prune")

		// Add all available images to the ID selector
		if all || prune {
			r, err := imageClient.ListImages(context.TODO(), nil)
			if err != nil {
				return err
			}
			for _, img := range r {
				logrus.Debugf("Adding image to be removed: %v", img.GetId())
				ids[img.GetId()] = true
			}
		}

		// On prune, remove images which are in use from the ID selector
		if prune {
			runtimeClient, err := getRuntimeService(cliCtx, 0)
			if err != nil {
				return err
			}

			// Container images
			containers, err := runtimeClient.ListContainers(context.TODO(), nil)
			if err != nil {
				return err
			}
			for _, container := range containers {
				img := container.GetImage().Image
				imageStatus, err := ImageStatus(imageClient, img, false)
				if err != nil {
					logrus.Errorf(
						"image status request for %q failed: %v",
						img, err,
					)
					continue
				}
				id := imageStatus.GetImage().GetId()
				logrus.Debugf("Excluding in use container image: %v", id)
				ids[id] = false
			}
		}

		if len(ids) == 0 {
			logrus.Info("No images to remove")
			return nil
		}

		errored := false
		for id, remove := range ids {
			if !remove {
				continue
			}
			status, err := ImageStatus(imageClient, id, false)
			if err != nil {
				logrus.Errorf("image status request for %q failed: %v", id, err)
				errored = true
				continue
			}
			if status.Image == nil {
				logrus.Errorf("no such image %s", id)
				errored = true
				continue
			}

			if err := RemoveImage(imageClient, id); err != nil {
				// We ignore further errors on prune because there might be
				// races
				if !prune {
					logrus.Errorf("error of removing image %q: %v", id, err)
					errored = true
				}
				continue
			}
			if len(status.Image.RepoTags) == 0 {
				// RepoTags is nil when pulling image by repoDigest,
				// so print deleted using that instead.
				for _, repoDigest := range status.Image.RepoDigests {
					fmt.Printf("Deleted: %s\n", repoDigest)
				}
				continue
			}
			for _, repoTag := range status.Image.RepoTags {
				fmt.Printf("Deleted: %s\n", repoTag)
			}
		}

		if errored {
			return fmt.Errorf("unable to remove the image(s)")
		}

		return nil
	},
}

var imageFsInfoCommand = &cli.Command{
	Name:                   "imagefsinfo",
	Usage:                  "Return image filesystem info",
	UseShortOptionHandling: true,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Output format, One of: json|yaml|go-template|table",
		},
		&cli.StringFlag{
			Name:  "template",
			Usage: "The template string is only used when output is go-template; The Template format is golang template",
		},
	},
	Action: func(c *cli.Context) error {
		imageClient, err := getImageService(c)
		if err != nil {
			return err
		}

		output := c.String("output")
		if output == "" { // default to json output
			output = "json"
		}
		tmplStr := c.String("template")

		r, err := ImageFsInfo(imageClient)
		if err != nil {
			return fmt.Errorf("image filesystem info request: %w", err)
		}
		for _, info := range r.ImageFilesystems {
			status, err := protobufObjectToJSON(info)
			if err != nil {
				return fmt.Errorf("marshal image filesystem info to json: %w", err)
			}

			switch output {
			case "json", "yaml", "go-template":
				if err := outputStatusInfo(status, nil, output, tmplStr); err != nil {
					return fmt.Errorf("output image filesystem info: %w", err)
				}
				continue
			case "table": // table output is after this switch block
			default:
				return fmt.Errorf("output option cannot be %s", output)
			}

			// otherwise output in table format
			fmt.Printf("TimeStamp: %d\n", info.Timestamp)
			fmt.Printf("UsedBytes: %s\n", info.UsedBytes)
			fmt.Printf("Mountpoint: %s\n", info.FsId.Mountpoint)
		}

		return nil

	},
}

// Ideally repo tag should always be image:tag.
// The repoTags is nil when pulling image by repoDigest,Then we will show image name instead.
func normalizeRepoTagPair(repoTags []string, imageName string) (repoTagPairs [][]string) {
	const none = "<none>"
	if len(repoTags) == 0 {
		repoTagPairs = append(repoTagPairs, []string{imageName, none})
		return
	}
	for _, repoTag := range repoTags {
		idx := strings.LastIndex(repoTag, ":")
		if idx == -1 {
			repoTagPairs = append(repoTagPairs, []string{"errorRepoTag", "errorRepoTag"})
			continue
		}
		name := repoTag[:idx]
		if name == none {
			name = imageName
		}
		repoTagPairs = append(repoTagPairs, []string{name, repoTag[idx+1:]})
	}
	return
}

func normalizeRepoDigest(repoDigests []string) (string, string) {
	if len(repoDigests) == 0 {
		return "<none>", "<none>"
	}
	repoDigestPair := strings.Split(repoDigests[0], "@")
	if len(repoDigestPair) != 2 {
		return "errorName", "errorRepoDigest"
	}
	return repoDigestPair[0], repoDigestPair[1]
}

// ImageStatus sends an ImageStatusRequest to the server, and parses
// the returned ImageStatusResponse.
func ImageStatus(client internalapi.ImageManagerService, image string, verbose bool) (*pb.ImageStatusResponse, error) {
	request := &pb.ImageStatusRequest{
		Image:   &pb.ImageSpec{Image: image},
		Verbose: verbose,
	}
	logrus.Debugf("ImageStatusRequest: %v", request)
	res, err := client.ImageStatus(context.TODO(), request.Image, request.Verbose)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("ImageStatusResponse: %v", res)
	return res, nil
}

// RemoveImage sends a RemoveImageRequest to the server, and parses
// the returned RemoveImageResponse.
func RemoveImage(client internalapi.ImageManagerService, image string) error {
	if image == "" {
		return fmt.Errorf("ImageID cannot be empty")
	}
	request := &pb.RemoveImageRequest{Image: &pb.ImageSpec{Image: image}}
	logrus.Debugf("RemoveImageRequest: %v", request)
	if err := client.RemoveImage(context.TODO(), request.Image); err != nil {
		return err
	}
	return nil
}

// ImageFsInfo sends an ImageStatusRequest to the server, and parses
// the returned ImageFsInfoResponse.
func ImageFsInfo(client internalapi.ImageManagerService) (*pb.ImageFsInfoResponse, error) {
	res, err := client.ImageFsInfo(context.TODO())
	if err != nil {
		return nil, err
	}
	resp := &pb.ImageFsInfoResponse{ImageFilesystems: res}
	logrus.Debugf("ImageFsInfoResponse: %v", resp)
	return resp, nil
}

// ListImages sends a ListImagesRequest to the server, and parses
// the returned ListImagesResponse.
func ListImages(client internalapi.ImageManagerService, image string) (*pb.ListImagesResponse, error) {
	request := &pb.ListImagesRequest{Filter: &pb.ImageFilter{Image: &pb.ImageSpec{Image: image}}}
	logrus.Debugf("ListImagesRequest: %v", request)
	res, err := client.ListImages(context.TODO(), request.Filter)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListImagesResponse{Images: res}
	logrus.Debugf("ListImagesResponse: %v", resp)
	return resp, nil
}

func parseCreds(creds string) (string, string, error) {
	if creds == "" {
		return "", "", errors.New("credentials can't be empty")
	}
	up := strings.SplitN(creds, ":", 2)
	if len(up) == 1 {
		return up[0], "", nil
	}
	if up[0] == "" {
		return "", "", errors.New("username can't be empty")
	}
	return up[0], up[1], nil
}

// PullImageWithSandbox sends a PullImageRequest to the server, and parses
// the returned PullImageResponse.
func PullImageWithSandbox(client internalapi.ImageManagerService, image string, auth *pb.AuthConfig, sandbox *pb.PodSandboxConfig, ann map[string]string) (*pb.PullImageResponse, error) {
	request := &pb.PullImageRequest{
		Image: &pb.ImageSpec{
			Image:       image,
			Annotations: ann,
		},
	}
	if auth != nil {
		request.Auth = auth
	}
	if sandbox != nil {
		request.SandboxConfig = sandbox
	}
	logrus.Debugf("PullImageRequest: %v", request)
	res, err := client.PullImage(context.TODO(), request.Image, request.Auth, request.SandboxConfig)
	if err != nil {
		return nil, err
	}
	resp := &pb.PullImageResponse{ImageRef: res}
	logrus.Debugf("PullImageResponse: %v", resp)
	return resp, nil
}
