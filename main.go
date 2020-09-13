// main
package main

import (
	"flag"
	"fmt"
	"runtime"
)

func main() {
	confPath := flag.String("c", "conf.json", "config json file")
	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())

	err, conf := NewConfig(*confPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	service := NewHTTP(conf)
	service.Start()
}
