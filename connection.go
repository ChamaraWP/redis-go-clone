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
func handleConnection(conn net.Conn, store *Store) {
	defer conn.Close()

	log.Println("Connection stablish", conn.RemoteAddr().String())
	reader := bufio.NewReader(conn)

	for {
		values, err := parseCommand(reader)

		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Print(err)
				log.Println("connection terminated by the client", conn.RemoteAddr())
				return
			}
			log.Println("reading failed", err)
			return
		}

		results, err := executeCommand(values, store)

		if err != nil {
			response := encodeError(err)
			_, writerErr := conn.Write([]byte(response))
			if writerErr != nil {
				log.Println("command execution failed", writerErr)
				return
			}

			continue
		}

		response := encodeResponse(results)
		_, err = conn.Write([]byte(response))

		if err != nil {
			log.Println("writing to the connection failed", err)
			return
		}

		log.Println("parsed command using RESP", values)
	}

}
