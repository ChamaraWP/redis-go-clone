package main

import (
	"bufio"
	"errors"
	"io"
	"strconv"
)

func parseCommand(reader *bufio.Reader) ([]string, error) {
	count, err := readArrayLength(reader)

	if err != nil {
		return nil, err
	}

	commands := make([]string, count)

	for i := 0; i < count; i++ {
		values, err := readFullBytes(reader)
		if err != nil {
			return nil, err
		}
		commands[i] = values
	}

	return commands, nil

}

func readArrayLength(reader *bufio.Reader) (int, error) {

	line, err := reader.ReadString('\n')

	if err != nil {
		return 0, err
	}

	if len(line) < 4 || line[0] != '*' {
		return 0, errors.New("invalid RESP array header")
	}

	countText := line[1 : len(line)-2]

	count, err := strconv.Atoi(countText)

	if err != nil {
		return 0, err
	}

	return count, nil

}

func readFullBytes(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')

	if err != nil {
		return "", err
	}

	if len(line) < 4 || line[0] != '$' {
		return "", errors.New("resp bulk parsing  failed")
	}

	lineText := line[1 : len(line)-2]

	length, err := strconv.Atoi(lineText)

	if err != nil {
		return "", err
	}

	buffer := make([]byte, length)

	_, err = io.ReadFull(reader, buffer)

	if err != nil {
		return "", err
	}

	crlf := make([]byte, 2)

	_, err = io.ReadFull(reader, crlf)

	if err != nil {
		return "", err
	}

	return string(buffer), nil

}

func encodeResponse(results CommandResult) string {
	switch results.Type {
	case ResponseSimpleString:
		return "+" + results.Value + "\r\n"
	case ResponseBulkString:
		return "$" + strconv.Itoa((len(results.Value))) + "\r\n" + results.Value + "\r\n"
	case ResponseNull:
		return "$-1\r\n"
	case ResponseInteger:
		return ":" + results.Value + "\r\n"

	}
	return ""
}
