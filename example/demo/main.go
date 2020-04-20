package main

import (
	"flag"
	"github.com/pxLi-io/libra"
)

func main() {
	flag.Parse()

	l, err := libra.New()
	if err != nil {
		println(err)
		return
	}
	println(l.Address())
	l.Serve()
}
