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

func executeCommand(parts []string, store *Store, persister Persister) (CommandResult, error) {
	return runCommand(
		parts,
		store,
		func(command []string) error {
			return persister.Append(command)
		},
	)

}

func replayCommand(parts []string, store *Store) (CommandResult, error) {
	return runCommand(
		parts,
		store,
		func(command []string) error {
			return nil
		},
	)

}

func runCommand(
	parts []string,
	store *Store,
	beforeMutation func([]string) error,
) (CommandResult, error) {
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

		if err := beforeMutation(parts); err != nil {
			return CommandResult{}, err
		}

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

		value, err := store.Increment(parts[1], func() error {
			return beforeMutation(parts)
		})

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
		absoluteTime := time.Now().Add(timeInSeconds)

		persistedCommand := []string{
			"EXPIREAT",
			parts[1],
			strconv.FormatInt(absoluteTime.Unix(), 10),
		}

		if err := beforeMutation(persistedCommand); err != nil {
			return CommandResult{}, err
		}

		value := store.ExpireAt(parts[1], absoluteTime)

		returnValue := "0"
		if value {
			returnValue = "1"
		}

		return CommandResult{
			Type:  ResponseInteger,
			Value: returnValue,
		}, nil

	}

	if command == "EXPIREAT" {
		if len(parts) != 3 {
			return CommandResult{}, errors.New("invalid EXPIREAT command")
		}

		timestamp, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return CommandResult{}, err
		}

		expiresAt := time.Unix(timestamp, 0)

		value := store.ExpireAt(parts[1], expiresAt)

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

		if err := beforeMutation(parts); err != nil {
			return CommandResult{}, err
		}

		keys := parts[1:]
		count := strconv.Itoa(store.Delete(keys...))

		return CommandResult{Type: ResponseInteger, Value: count}, nil
	}

	if command == "EXISTS" {
		if (len(parts)) < 2 {
			return CommandResult{}, errors.New("invalid EXISTS command")
		}

		keys := parts[1:]
		count := strconv.Itoa(store.Exists(keys...))

		return CommandResult{Type: ResponseInteger, Value: count}, nil
	}

	return CommandResult{}, errors.New("invalid Command Received")

}
