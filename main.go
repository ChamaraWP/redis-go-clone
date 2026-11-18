package main

import (
	"fmt"
	"log"
	"net"
)

func main() {
	fmt.Println("Starting Redis Server")
	listener, err := net.Listen("tcp", ":6379")
	store := NewStore()

	if err != nil {
		log.Fatal(err)
		fmt.Println("Redis Server Terminating")
	}

	serve(listener, store)

}
