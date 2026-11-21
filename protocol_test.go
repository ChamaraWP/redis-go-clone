package main

import (
	"bufio"
	"strings"
	"testing"
)

func TestProtocolParsingCRLF(t *testing.T) {
	reader := bufio.NewReader(
		strings.NewReader("$3\r\nfooXX"),
	)

	_, err := readFullBytes(reader)

	if err == nil {
		t.Fatal("expected error for invalid CRLF")
	}
}

func TestReadFullBytesNonNumericLength(t *testing.T) {
	reader := bufio.NewReader(
		strings.NewReader("$abc\r\nfoo\r\n"),
	)

	_, err := readFullBytes(reader)

	if err == nil {
		t.Fatal("expected error for if the value after $ is not a number")
	}
}

func TestReadArrayLengthInvalidArrayPrefix(t *testing.T) {
	reader := bufio.NewReader(
		strings.NewReader("abc\r\n"),
	)

	_, err := readArrayLength(reader)

	if err == nil {
		t.Fatal("expected invalid array prefix to throw an error")
	}
}

func TestReadArrayLengthInvalidArrayLength(t *testing.T) {
	reader := bufio.NewReader(
		strings.NewReader("*x\r\n"),
	)

	_, err := readArrayLength(reader)

	if err == nil {
		t.Fatal("expected invalid array length to throw an error")
	}
}

func TestEncodeRespCommand(t *testing.T) {
	command := []string{"SET", "foo", "bar"}

	encodedCommand := string(encodeCommandRESP(command))
	expected := "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"

	if encodedCommand != expected {
		t.Fatalf("expected %q, got %q", expected, encodedCommand)
	}
}
