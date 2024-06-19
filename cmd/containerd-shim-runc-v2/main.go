package main

import (
	"context"
	"demo/over/runtime/v2/runc/manager"
	_ "demo/plugins/shim/pause"
	"demo/plugins/shim/shim"
	_ "demo/plugins/shim/task"
)

func main() {
	shim.RunManager(context.Background(), manager.NewShimManager("io.containerd.runc.v2"))
}
