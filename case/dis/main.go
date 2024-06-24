package main

import (
	"fmt"
	"io/ioutil"
	"log"
)

func main() {
	dirPath := "/Users/acejilam/Desktop/todo/containerd" // 需要查看的目录路径
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if file.IsDir() {
			fmt.Println("子目录：", file.Name())
		}
	}
}
