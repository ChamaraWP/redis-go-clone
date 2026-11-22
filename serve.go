package main

import (
	"log"
	"net"
)

func serve(listener net.Listener, store *Store, persister Persister) {
	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Println(err)
			continue
		}

		go handleConnection(conn, store, persister)
	}
}
