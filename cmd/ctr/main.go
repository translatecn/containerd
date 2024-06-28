package main

import (
	"crypto"
	"demo/pkg/hasher"
	"demo/pkg/seed"
	"fmt"
	"os"

	"demo/cmd/ctr/app"
	//nolint:staticcheck // Global math/rand seed is deprecated, but still used by external dependencies
	"github.com/urfave/cli"
)

var pluginCmds = []cli.Command{}

func init() {
	//nolint:staticcheck // Global math/rand seed is deprecated, but still used by external dependencies
	seed.WithTimeAndRand()
	crypto.RegisterHash(crypto.SHA256, hasher.NewSHA256)
}

func main() {
	app := app.New()
	app.Commands = append(app.Commands, pluginCmds...)
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "ctr: %s\n", err)
		os.Exit(1)
	}
}
