package my_mk

import (
	"os"
)

func MkdirAll(path string, perm os.FileMode) error {
	//fmt.Println("MkdirAll", path)
	return os.MkdirAll(path, perm)
}

func Mkdir(name string, perm os.FileMode) error {
	//fmt.Println("Mkdir", name)
	return os.Mkdir(name, perm)
}
func MkdirTemp(dir, pattern string) (string, error) {
	x, err := os.MkdirTemp(dir, pattern)
	//fmt.Println("MkdirTemp", x)
	return x, err
}
