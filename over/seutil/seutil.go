package seutil

import (
	"github.com/opencontainers/selinux/go-selinux"
)

func ChangeToKVM(l string) (string, error) {
	if l == "" || !selinux.GetEnabled() {
		return "", nil
	}
	proc, _ := selinux.KVMContainerLabels()
	selinux.ReleaseLabel(proc)

	current, err := selinux.NewContext(l)
	if err != nil {
		return "", err
	}
	next, err := selinux.NewContext(proc)
	if err != nil {
		return "", err
	}
	current["type"] = next["type"]
	return current.Get(), nil
}
