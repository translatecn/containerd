package drop

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

func DropEnv(environ []string) string {
	eM := make(map[string]string)
	for k, v := range eA2M(environ) {
		eM[k] = v
	}
	for k, _ := range eA2M(os.Environ()) {
		delete(eM, k)
	}
	var keys []string
	for k, _ := range eM {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var res []string
	for _, k := range keys {
		res = append(res, fmt.Sprintf("%s=%s", k, eM[k]))
	}
	marshal, _ := json.Marshal(res)
	return string(marshal)
}
func eA2M(environ []string) map[string]string {
	eM := make(map[string]string)
	for _, env := range environ {
		envParts := strings.SplitN(env, "=", 2)
		eM[envParts[0]] = envParts[1]
	}
	return eM
}
