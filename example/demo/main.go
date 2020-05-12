package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	"github.com/pxli-io/libra"
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
	for i := 0; i < 12800000; i++ {
		arr = append(arr, fmt.Sprintf("cluster_%d", i))
	}

	go func() {
		count := 0
		cur := time.Now()
		for {
			//time.Sleep(1 * time.Millisecond)
			a := l.Get(arr...)
			if a != nil {
				count++
				if count%100 == 0 {
					println("###", count)
					fmt.Println("---", len(a.Get(l.LocalID())))
					println(time.Since(cur).String())
					println(runtime.NumGoroutine())
					cur = time.Now()
				}
				a.Free()
			} else {
				println("$$$", count)
			}
			//fmt.Println(count)
		}
	}()

	go func() {
		count := 0
		cur := time.Now()
		for {
			//time.Sleep(1 * time.Millisecond)
			a := l.Get(arr...)
			if a != nil {
				count++
				if count%100 == 0 {
					println("2###", count)
					fmt.Println("2---", len(a.Get(l.LocalID())))
					println(2,time.Since(cur).String())
					println(2,runtime.NumGoroutine())
					cur = time.Now()
				}
				a.Free()
			} else {
				println("2$$$", count)
			}
			//fmt.Println(count)
		}
	}()

	times, i := 0, 2
	for {
		time.Sleep(10 * time.Second)
		times++
		fmt.Printf("fire update %d\n", times)
		err = l.UpdateWeight(i)
		if err != nil {
			println(err.Error())
		}
		i = i + 4
	}
}
