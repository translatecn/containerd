package write

import (
	"encoding/json"
	"fmt"
	"os"
)

func WriteFile(filename string, data interface{}) error {
	marshal, _ := json.MarshalIndent(data, "  ", "  ")
	return os.WriteFile(filename, marshal, os.ModePerm)
}

func AppendRunLog(flag string, info interface{}) {

	// 打开文件，如果文件不存在则创建
	file, err := os.OpenFile("/tmp/run.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()
	var marshal []byte
	switch v := info.(type) {
	case string:
		t := map[interface{}]interface{}{}
		err := json.Unmarshal([]byte(v), &t)
		if err != nil {
			marshal = []byte(v)
		} else {
			marshal, _ = json.MarshalIndent(t, "  ", "  ")
		}
	default:
		marshal, _ = json.MarshalIndent(info, "  ", "  ")
	}
	content := ``
	if len(marshal) != 0 {
		content = flag + "\n" + string(marshal) + "\n"
	} else {
		content = flag + "\n"
	}
	_, err = file.WriteString(content)
	if err != nil {
		fmt.Println(err)
	}
}
