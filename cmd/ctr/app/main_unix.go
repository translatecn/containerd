package app

import "demo/cmd/ctr/commands/shim"

func init() {
	extraCmds = append(extraCmds, shim.Command)
}
