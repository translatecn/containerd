package main

import (
	"fmt"
	"net/url"
	"path"
)

func main() {

	u := url.URL{
		Scheme: "http",
		Host:   "127.0.0.1:0",
	}
	fmt.Println(
		u.ResolveReference(&url.URL{
			Path: path.Join("portforward", "token"),
		}).String(),
	)

}
