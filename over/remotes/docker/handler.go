package docker

import (
	"context"
	labels2 "demo/over/labels"
	"demo/over/log"
	"demo/over/reference"
	"fmt"
	"net/url"
	"strings"

	"demo/over/content"
	"demo/over/images"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// AppendDistributionSourceLabel updates the label of blob with distribution source.
func AppendDistributionSourceLabel(manager content.Manager, ref string) (images.HandlerFunc, error) {
	refspec, err := reference.Parse(ref)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse("dummy://" + refspec.Locator)
	if err != nil {
		return nil, err
	}

	source, repo := u.Hostname(), strings.TrimPrefix(u.Path, "/")
	return func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		info, err := manager.Info(ctx, desc.Digest)
		if err != nil {
			return nil, err
		}

		key := distributionSourceLabelKey(source)

		originLabel := ""
		if info.Labels != nil {
			originLabel = info.Labels[key]
		}
		value := appendDistributionSourceLabel(originLabel, repo)

		// The repo name has been limited under 256 and the distribution
		// label might hit the limitation of label size, when blob data
		// is used as the very, very common layer.
		if err := labels2.Validate(key, value); err != nil {
			log.G(ctx).Warnf("skip to append distribution label: %s", err)
			return nil, nil
		}

		info = content.Info{
			Digest: desc.Digest,
			Labels: map[string]string{
				key: value,
			},
		}
		_, err = manager.Update(ctx, info, fmt.Sprintf("labels.%s", key))
		return nil, err
	}, nil
}

func appendDistributionSourceLabel(originLabel, repo string) string {
	repos := []string{}
	if originLabel != "" {
		repos = strings.Split(originLabel, ",")
	}
	repos = append(repos, repo)

	// use empty string to present duplicate items
	for i := 1; i < len(repos); i++ {
		tmp, j := repos[i], i-1
		for ; j >= 0 && repos[j] >= tmp; j-- {
			if repos[j] == tmp {
				tmp = ""
			}
			repos[j+1] = repos[j]
		}
		repos[j+1] = tmp
	}

	i := 0
	for ; i < len(repos) && repos[i] == ""; i++ {
	}

	return strings.Join(repos[i:], ",")
}

func distributionSourceLabelKey(source string) string {
	return fmt.Sprintf("%s.%s", labels2.LabelDistributionSource, source)
}

// selectRepositoryMountCandidate will select the repo which has longest
// common prefix components as the candidate.
func selectRepositoryMountCandidate(refspec reference.Spec, sources map[string]string) string {
	u, err := url.Parse("dummy://" + refspec.Locator)
	if err != nil {
		// NOTE: basically, it won't be error here
		return ""
	}

	source, target := u.Hostname(), strings.TrimPrefix(u.Path, "/")
	repoLabel, ok := sources[distributionSourceLabelKey(source)]
	if !ok || repoLabel == "" {
		return ""
	}

	n, match := 0, ""
	components := strings.Split(target, "/")
	for _, repo := range strings.Split(repoLabel, ",") {
		// the target repo is not a candidate
		if repo == target {
			continue
		}

		if l := commonPrefixComponents(components, repo); l >= n {
			n, match = l, repo
		}
	}
	return match
}

func commonPrefixComponents(components []string, target string) int {
	targetComponents := strings.Split(target, "/")

	i := 0
	for ; i < len(components) && i < len(targetComponents); i++ {
		if components[i] != targetComponents[i] {
			break
		}
	}
	return i
}
