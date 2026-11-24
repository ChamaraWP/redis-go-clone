package main

import (
	"bufio"
	"errors"
	"io"
	"os"
)

func replayAOF(path string, store *Store) error {
	file, err := os.Open(path)

	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		parts, err := parseCommand(reader)

		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return err
		}

		_, err = replayCommand(parts, store)
		if err != nil {
			return err
		}

	}

}
