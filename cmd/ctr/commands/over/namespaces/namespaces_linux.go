package namespaces

import (
	"demo/pkg/namespaces"
	"demo/pkg/runtime/opts"
	"github.com/urfave/cli"
)

func deleteOpts(context *cli.Context) []namespaces.DeleteOpts {
	var delOpts []namespaces.DeleteOpts
	if context.Bool("cgroup") {
		delOpts = append(delOpts, opts.WithNamespaceCgroupDeletion)
	}
	return delOpts
}
