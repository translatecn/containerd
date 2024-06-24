package main

import (
	"context"
	_ "demo/over/plugins/shim/pause"
	"demo/over/plugins/shim/shim"
	_ "demo/over/plugins/shim/task"
	"demo/over/runtime/v2/runc/manager"
)

func main() {
	shim.RunManager(context.Background(), manager.NewShimManager("io.containerd.runc.v2"))
}
