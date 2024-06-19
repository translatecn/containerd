package drop

import (
	"encoding/json"
	"os"
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
	marshal, _ := json.Marshal(eM)
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
