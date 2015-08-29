// go_server project server.go
package main

import (
	"flag"
	"go_net"
	"runtime"
	"server"
)

//var numThread = flag.Int("nt", runtime.NumCPU(), "usage:-nt 100(number of thread)")
var address = flag.String("address", "", "usage:-address ip:port")
var filename = flag.String("filename", "", "usage:-filename test.dat")

func main() {
	flag.Parse()

	if *address == "" || *filename == "" {
		flag.Usage()
		return
	}

	udpConn, ok := go_net.GoListen(*address)
	defer udpConn.Close()

	if true != ok {
		return
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	server.Run(filename, udpConn)
}
