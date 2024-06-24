package main

import (
	"crypto"
	"demo/over/hasher"
	"demo/over/seed"
	"fmt"
	"os"

	"demo/cmd/containerd/command"
	//nolint:staticcheck // Global math/rand seed is deprecated, but still used by external dependencies

	_ "demo/plugins/containerd"
)

func init() {
	//nolint:staticcheck // Global math/rand seed is deprecated, but still used by external dependencies
	seed.WithTimeAndRand()
	crypto.RegisterHash(crypto.SHA256, hasher.NewSHA256)
}

func main() {
	app := command.App()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "containerd: %s\n", err)
		os.Exit(1)
	}
}
