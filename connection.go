package main

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
)

// this function will accept an connection from that created from
// TCP listener and check whether it is EOF or has any other error
// if not i will keep accepting the data and if the function return it will
// automatically clean it up
func handleConnection(conn net.Conn) {
	defer conn.Close()

	log.Println("Connection stablish", conn.RemoteAddr().String())
	reader := bufio.NewReader(conn)

	for {
		values, err := parseCommand(reader)

		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Println("Connection terminated by the client", conn.RemoteAddr())
				return
			}
			log.Println("Reading failed", err)
			return
		}

		log.Println("parsed command from resp", values)
	}

}
