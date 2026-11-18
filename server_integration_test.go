package main

import (
	"bufio"
	"net"
	"testing"
)

func TestTCPSetCommand(t *testing.T) {
	store := NewStore()

	listener, err := net.Listen("tcp", ":0")

	if err != nil {
		t.Fatal(err)
	}

	defer listener.Close()

	go serve(listener, store)

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

	go serve(listener, store)

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

	go serve(listener, store)

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
