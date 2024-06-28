package docker

import (
	"context"
	"demo/pkg/reference"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// RepositoryScope returns a repository scope string such as "repository:foo/bar:pull"
// for "host/foo/bar:baz".
// When push is true, both pull and push are added to the scope.
func RepositoryScope(refspec reference.Spec, push bool) (string, error) {
	u, err := url.Parse("dummy://" + refspec.Locator)
	if err != nil {
		return "", err
	}
	s := "repository:" + strings.TrimPrefix(u.Path, "/") + ":pull"
	if push {
		s += ",push"
	}
	return s, nil
}

// tokenScopesKey is used for the key for context.WithValue().
// value: []string (e.g. {"registry:foo/bar:pull"})
type tokenScopesKey struct{}

// ContextWithRepositoryScope returns a context with tokenScopesKey{} and the repository scope value.
func ContextWithRepositoryScope(ctx context.Context, refspec reference.Spec, push bool) (context.Context, error) {
	s, err := RepositoryScope(refspec, push)
	if err != nil {
		return nil, err
	}
	return WithScope(ctx, s), nil
}

// WithScope appends a custom registry auth scope to the context.
func WithScope(ctx context.Context, scope string) context.Context {
	var scopes []string
	if v := ctx.Value(tokenScopesKey{}); v != nil {
		scopes = v.([]string)
		scopes = append(scopes, scope)
	} else {
		scopes = []string{scope}
	}
	return context.WithValue(ctx, tokenScopesKey{}, scopes)
}

// ContextWithAppendPullRepositoryScope is used to append repository pull
// scope into existing scopes indexed by the tokenScopesKey{}.
func ContextWithAppendPullRepositoryScope(ctx context.Context, repo string) context.Context {
	return WithScope(ctx, fmt.Sprintf("repository:%s:pull", repo))
}

// GetTokenScopes returns deduplicated and sorted scopes from ctx.Value(tokenScopesKey{}) and common scopes.
func GetTokenScopes(ctx context.Context, common []string) []string {
	scopes := []string{}
	if x := ctx.Value(tokenScopesKey{}); x != nil {
		scopes = append(scopes, x.([]string)...)
	}

	scopes = append(scopes, common...)
	sort.Strings(scopes)

	if len(scopes) == 0 {
		return scopes
	}

	l := 0
	for idx := 1; idx < len(scopes); idx++ {
		// Note: this comparison is unaware of the scope grammar (https://docs.docker.com/registry/spec/auth/scope/)
		// So, "repository:foo/bar:pull,push" != "repository:foo/bar:push,pull", although semantically they are equal.
		if scopes[l] == scopes[idx] {
			continue
		}

		l++
		scopes[l] = scopes[idx]
	}
	return scopes[:l+1]
}
