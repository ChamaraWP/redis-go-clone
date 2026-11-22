package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestTCPSetCommand(t *testing.T) {
	store := NewStore()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	persister, err := NewFilePersister("appendonlytest.aof")
	if err != nil {
		t.Fatal("file initialization failed", err)
	}
	defer persister.Close()

	go serve(listener, store, persister)

	conn, err := net.Dial("tcp", listener.Addr().String())

	if err != nil {
		t.Fatal(err)
	}

	defer conn.Close()

	command := "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"
	conn.Write([]byte(command))

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')

	if err != nil {
		t.Fatal(err)
	}

	if response != "+OK\r\n" {
		t.Fatalf("expected response to be +OK\\r\\n but got %q", response)
	}

}

func TestTCPSetAndGet(t *testing.T) {
	store := NewStore()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	persister, err := NewFilePersister("appendonlytest.aof")
	if err != nil {
		t.Fatal("file initialization failed", err)
	}
	defer persister.Close()

	go serve(listener, store, persister)

	conn, err := net.Dial("tcp", listener.Addr().String())

	if err != nil {
		t.Fatal(err)
	}

	defer conn.Close()

	commandSet := "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"
	_, err = conn.Write([]byte(commandSet))
	if err != nil {
		t.Fatal(err)
	}

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')

	if err != nil {
		t.Fatal(err)
	}

	if response != "+OK\r\n" {
		t.Fatalf("expected response to be +OK\\r\\n but got %q", response)
	}

	commandGet := "*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n"
	_, err = conn.Write([]byte(commandGet))
	if err != nil {
		t.Fatal(err)
	}

	header, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	value, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	responseGet := header + value

	if responseGet != "$3\r\nbar\r\n" {
		t.Fatalf("expected response to be $3\\r\\nbar\\r\\n but got %q", responseGet)
	}

}

func TestTCPSharedStateAcrossConnections(t *testing.T) {

	store := NewStore()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	persister, err := NewFilePersister("appendonlytest.aof")
	if err != nil {
		t.Fatal("file initialization failed", err)
	}
	defer persister.Close()

	go serve(listener, store, persister)

	connA, err := net.Dial("tcp", listener.Addr().String())

	if err != nil {
		t.Fatal(err)
	}

	defer connA.Close()

	connB, err := net.Dial("tcp", listener.Addr().String())

	if err != nil {
		t.Fatal(err)
	}

	defer connB.Close()

	commandSet := "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"
	_, err = connA.Write([]byte(commandSet))
	if err != nil {
		t.Fatal(err)
	}

	readerA := bufio.NewReader(connA)
	response, err := readerA.ReadString('\n')

	if err != nil {
		t.Fatal(err)
	}

	if response != "+OK\r\n" {
		t.Fatalf("expected response to be +OK\\r\\n but got %q", response)
	}

	commandGet := "*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n"
	_, err = connB.Write([]byte(commandGet))
	if err != nil {
		t.Fatal(err)
	}

	readerB := bufio.NewReader(connB)
	responseHeaderB, err := readerB.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	responseValueB, err := readerB.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	responseB := responseHeaderB + responseValueB

	if responseB != "$3\r\nbar\r\n" {
		t.Fatalf("expected response to be $3\\r\\nbar\\r\\n but got %q", responseB)
	}

}

func TestTCPConcurrentIncrement(t *testing.T) {

	store := NewStore()
	var wg sync.WaitGroup

	listener, err := net.Listen("tcp", ":0")
	errCh := make(chan error, 100)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	persister, err := NewFilePersister("appendonlytest.aof")
	if err != nil {
		t.Fatal("file initialization failed", err)
	}
	defer persister.Close()

	go serve(listener, store, persister)
	conn, err := net.Dial("tcp", listener.Addr().String())

	if err != nil {
		t.Fatal(err)
	}

	defer conn.Close()

	commandSet := "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$2\r\n10\r\n"
	_, err = conn.Write([]byte(commandSet))
	if err != nil {
		t.Fatal(err)
	}

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')

	if err != nil {
		t.Fatal(err)
	}

	if response != "+OK\r\n" {
		t.Fatalf("expected response to be +OK\\r\\n but got %q", response)
	}

	for range 100 {
		wg.Go(func() {
			conn, err := net.Dial("tcp", listener.Addr().String())

			if err != nil {
				errCh <- err
				return
			}
			defer conn.Close()

			commandIncr := "*2\r\n$4\r\nINCR\r\n$3\r\nfoo\r\n"
			_, err = conn.Write([]byte(commandIncr))

			if err != nil {
				errCh <- err
				return
			}

			reader := bufio.NewReader(conn)
			response, err := reader.ReadString('\n')

			if err != nil {
				errCh <- err
				return
			}

			if !strings.HasPrefix(response, ":") {
				errCh <- fmt.Errorf("unexpected INCR response: %q", response)
				return
			}

			numberText := strings.TrimSuffix(
				strings.TrimPrefix(response, ":"),
				"\r\n",
			)

			_, err = strconv.Atoi(numberText)
			if err != nil {
				errCh <- fmt.Errorf("invalid INCR integer response: %q", response)
				return
			}

		})

	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}

	commandGet := "*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n"
	_, err = conn.Write([]byte(commandGet))
	if err != nil {
		t.Fatal(err)
	}

	header, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	value, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	responseGet := header + value

	if responseGet != "$3\r\n110\r\n" {
		t.Fatalf("expected response $3\\r\\n110\\r\\n, got %q", responseGet)
	}

}

func TestRespMalformedCommand(t *testing.T) {
	store := NewStore()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	persister, err := NewFilePersister("appendonlytest.aof")
	if err != nil {
		t.Fatal("file initialization failed", err)
	}
	defer persister.Close()

	go serve(listener, store, persister)

	conn, err := net.Dial("tcp", listener.Addr().String())

	if err != nil {
		t.Fatal(err)
	}

	defer conn.Close()

	malformedCommand := "*2\r\n$3\r\nGET\r\n$5\r\nfoo\r\n"

	_, err = conn.Write([]byte(malformedCommand))
	if err != nil {
		t.Fatal(err)
	}

	tcpConn, ok := conn.(*net.TCPConn)

	if !ok {
		t.Fatal("expected TCP connection")
	}

	if err := tcpConn.CloseWrite(); err != nil {
		t.Fatal(err)
	}

	reader := bufio.NewReader(conn)

	_, err = reader.ReadString('\n')

	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected server to close connection, got %v", err)
	}
}

func TestRespInvalidCommand(t *testing.T) {

	store := NewStore()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	persister, err := NewFilePersister("appendonlytest.aof")
	if err != nil {
		t.Fatal("file initialization failed", err)
	}
	defer persister.Close()
	go serve(listener, store, persister)

	conn, err := net.Dial("tcp", listener.Addr().String())

	if err != nil {
		t.Fatal(err)
	}

	defer conn.Close()

	invalidGetCommand := "*3\r\n$3\r\nGET\r\n$3\r\nfoo\r\n$5\r\nextra\r\n"
	_, err = conn.Write([]byte(invalidGetCommand))
	if err != nil {
		t.Fatal(err)
	}

	reader := bufio.NewReader(conn)
	errorResponse, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(errorResponse, "-ERR") {
		t.Fatalf("expected RESP error, got %q", errorResponse)
	}

	validSetCommand := "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"
	_, err = conn.Write([]byte(validSetCommand))
	if err != nil {
		t.Fatal(err)
	}

	setResponse, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	if setResponse != "+OK\r\n" {
		t.Fatalf("expected +OK\\r\\n, got %q", setResponse)
	}

	validGetCommand := "*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n"
	_, err = conn.Write([]byte(validGetCommand))
	if err != nil {
		t.Fatal(err)
	}

	header, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	value, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	response := header + value

	if response != "$3\r\nbar\r\n" {
		t.Fatalf("expected response to be $3\\r\\nbar\\r\\n but got %q", response)
	}

}
