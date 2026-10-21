package main

import (
	"fmt"
	"log"
	"net"
)

func main() {
	fmt.Println("Starting Redis Server")
	listener, err := net.Listen("tcp", ":6379")

	if err != nil {
		log.Fatal(err)
		fmt.Println("Redis Server Terminating")
	}

	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Println(err)
			continue
		}

		log.Println("Connection stablish", conn.RemoteAddr().String())
		buff := make([]byte, 1024)
		n, err := conn.Read(buff)
		if err != nil {
			log.Println("Reading failed")
		}
		log.Println("Number of byte read", len(buff[:n]))
	}

}
