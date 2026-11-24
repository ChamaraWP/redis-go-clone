package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	fmt.Println("Starting Redis Server")

	path := "appendonly.aof"
	store := NewStore()

	err := replayAOF(path, store)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatal(err)
	}

	persister, err := NewFilePersister(path)
	if err != nil {
		log.Fatal(err)
	}
	defer persister.Close()

	listener, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	serve(listener, store, persister)
}
