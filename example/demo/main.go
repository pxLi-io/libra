package main

import (
	"flag"
	"fmt"
	"github.com/pxli-io/libra"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"
)

var addr = flag.String("addr", "127.0.0.1:5005", "pprof address, format <host>:<port>")

func main() {
	flag.Parse()

	go func() {
		log.Println(http.ListenAndServe(*addr, nil))
	}()

	l, err := libra.New()
	if err != nil {
		println(err.Error())
		return
	}
	println(l.Address())
	go l.Serve()
	defer l.Shutdown()

	var arr []string
	for i := 0; i < 12800; i++ {
		arr = append(arr, fmt.Sprintf("cluster_%d", i))
	}

	go func() {
		count := 0
		for {
			time.Sleep(10 * time.Millisecond)
			a := l.Get(arr...)
			if a != nil {
				//fmt.Println("---", len(a.Get(l.LocalID())))
				count++
				if count % 100 == 0 {
					println("###", count)
				}
				a.Free()
			} else {
				println("$$$", count)
			}
			//fmt.Println(count)
		}
	}()

	times, i := 0, 2
	for {
		time.Sleep(10 * time.Second)
		times++
		fmt.Printf("fire update %d\n", times)
		err = l.UpdateLoad(i)
		if err != nil {
			println(err.Error())
		}
		i = i + 4
	}
}
