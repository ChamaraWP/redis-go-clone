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
	}
	defer listener.Close()

	store := NewStore()

	persister, err := NewFilePersister("appendonly.aof")
	if err != nil {
		log.Fatal(err)
	}
	defer persister.Close()

	if err != nil {
		log.Fatal(err)
		fmt.Println("Redis Server Terminating")
	}

	serve(listener, store, persister)

}
