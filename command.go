package main

import (
	"errors"
	"log"
	"strconv"
	"strings"
)

type ResponseType string

const (
	ResponseSimpleString ResponseType = "simple_string"
	ResponseBulkString   ResponseType = "bulk_string"
	ResponseNull         ResponseType = "null"
	ResponseInteger      ResponseType = "integer"
)

type CommandResult struct {
	Type  ResponseType
	Value string
}

func executeCommand(parts []string, store *Store) (CommandResult, error) {
	log.Printf("received payload %s", parts)

	if len(parts) == 0 {
		return CommandResult{}, errors.New("empty command")
	}

	command := strings.ToUpper(parts[0])

	if command == "SET" {
		if len(parts) != 3 {
			return CommandResult{}, errors.New("invalid Set command")
		}
		key := parts[1]
		value := parts[2]
		store.Set(key, value)
		return CommandResult{Type: ResponseSimpleString, Value: "OK"}, nil
	}

	if command == "GET" {
		if len(parts) != 2 {
			return CommandResult{}, errors.New("invalid Get command")
		}

		key := parts[1]
		value, ok := store.Get(key)

		if !ok {
			return CommandResult{Type: ResponseNull, Value: ""}, nil
		}

		return CommandResult{Type: ResponseBulkString, Value: value}, nil
	}

	if command == "INCR" {
		if (len(parts)) != 2 {
			return CommandResult{}, errors.New("invalid INCR command")
		}

		value, err := store.increment(parts[1])

		if err != nil {
			return CommandResult{}, err
		}

		return CommandResult{
			Type:  ResponseInteger,
			Value: strconv.Itoa(value),
		}, nil

	}

	return CommandResult{}, errors.New("invalid Command Received")

}
