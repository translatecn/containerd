package platforms

import (
	"demo/over/log"
	"runtime"
	"sync"
)

// Present the ARM instruction set architecture, eg: v7, v8
// Don't use this value directly; call cpuVariant() instead.
var cpuVariantValue string

var cpuVariantOnce sync.Once

func cpuVariant() string {
	cpuVariantOnce.Do(func() {
		if isArmArch(runtime.GOARCH) {
			var err error
			cpuVariantValue, err = getCPUVariant()
			if err != nil {
				log.L.Errorf("Error getCPUVariant for OS %s: %v", runtime.GOOS, err)
			}
		}
	})
	return cpuVariantValue
}
