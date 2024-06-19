package cni

import (
	"fmt"

	types100 "demo/others/cni/pkg/types/100"
)

func validateInterfaceConfig(ipConf *types100.IPConfig, ifs int) error {
	if ipConf == nil {
		return fmt.Errorf("invalid IP configuration (nil)")
	}
	if ipConf.Interface != nil && *ipConf.Interface > ifs {
		return fmt.Errorf("invalid IP configuration (interface number %d is > number of interfaces %d)", *ipConf.Interface, ifs)
	}
	return nil
}

func getIfName(prefix string, i int) string {
	return fmt.Sprintf("%s%d", prefix, i)
}

func defaultInterface(prefix string) string {
	return getIfName(prefix, 0)
}
