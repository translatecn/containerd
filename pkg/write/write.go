package write

import (
	"encoding/json"
	"os"
)

func WriteFile(filename string, data interface{}) error {
	marshal, _ := json.MarshalIndent(data, "  ", "  ")
	return os.WriteFile(filename, marshal, os.ModePerm)
}
