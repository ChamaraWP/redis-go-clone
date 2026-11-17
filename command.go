package main

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"time"
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

		entry := Entry{
			Value: value,
		}

		store.Set(key, entry)
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

		value, err := store.Increment(parts[1])

		if err != nil {
			return CommandResult{}, err
		}

		return CommandResult{
			Type:  ResponseInteger,
			Value: strconv.Itoa(value),
		}, nil

	}

	if command == "EXPIRE" {

		if (len(parts)) != 3 {
			return CommandResult{}, errors.New("invalid EXPIRE command")
		}

		seconds, err := strconv.Atoi(parts[2])

		if err != nil {
			return CommandResult{}, err
		}

		timeInSeconds := time.Duration(seconds) * time.Second

		value := store.Expire(parts[1], timeInSeconds)

		returnValue := "0"
		if value {
			returnValue = "1"
		}

		return CommandResult{
			Type:  ResponseInteger,
			Value: returnValue,
		}, nil

	}

	if command == "TTL" {
		if (len(parts)) != 2 {
			return CommandResult{}, errors.New("invalid TTL command")
		}

		key := parts[1]

		value, status := store.TTL(key)

		switch status {

		case TTLKeyMissing:
			return CommandResult{
				Type:  ResponseInteger,
				Value: "-2",
			}, nil

		case TTLNoExpiry:
			return CommandResult{
				Type:  ResponseInteger,
				Value: "-1",
			}, nil

		case TTLHasExpiry:
			seconds := int64(value / time.Second)

			return CommandResult{
				Type:  ResponseInteger,
				Value: strconv.FormatInt(seconds, 10),
			}, nil

		default:
			return CommandResult{}, errors.New("unknown TTL status")
		}

	}

	if command == "DEL" {
		if (len(parts)) < 2 {
			return CommandResult{}, errors.New("invalid DEL command")
		}

		key := parts[1:]
		count := strconv.Itoa(store.Delete(key...))

		return CommandResult{Type: ResponseInteger, Value: count}, nil
	}

	if command == "EXISTS" {
		if (len(parts)) < 2 {
			return CommandResult{}, errors.New("invalid EXISTS command")
		}

		key := parts[1:]
		count := strconv.Itoa(store.Exists(key...))

		return CommandResult{Type: ResponseInteger, Value: count}, nil
	}

	return CommandResult{}, errors.New("invalid Command Received")

}
