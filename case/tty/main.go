package main

import (
	"fmt"
	"k8s.io/kubectl/pkg/util/term"
	"os"
)

func main() {
	t := term.TTY{
		In:  os.Stdin,
		Out: os.Stdout,
		Raw: true,
	}
	fmt.Println(t.GetSize())
}
