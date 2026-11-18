package main

import (
	"log"
	"net"
)

func serve(listener net.Listener, store *Store) {
	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Println(err)
			continue
		}

		go handleConnection(conn, store)
	}
}
