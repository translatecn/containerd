package main

import (
	"context"
	_ "demo/pkg/plugins/shim/pause"
	"demo/pkg/plugins/shim/shim"
	_ "demo/pkg/plugins/shim/task"
	"demo/pkg/runtime/v2/runc/manager"
)

func main() {
	shim.RunManager(context.Background(), manager.NewShimManager("io.containerd.runc.v2"))
}
