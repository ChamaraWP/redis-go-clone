package main

import (
	"errors"
	"os"
	"sync"
)

type Persister interface {
	Append(command []string) error
}

type FilePersister struct {
	file  *os.File
	mutex sync.Mutex
}

func NewFilePersister(path string) (*FilePersister, error) {
	options := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	file, err := os.OpenFile(path, options, 0644)

	if err != nil {
		return nil, err
	}

	return &FilePersister{
		file: file,
	}, nil
}

func (p *FilePersister) Append(command []string) error {
	encodedCommand := encodeCommandRESP(command)
	if len(encodedCommand) == 0 {
		return errors.New("cannot persist empty command")
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	writtenByteLength, err := p.file.Write(encodedCommand)

	if err != nil {
		return err
	}

	if writtenByteLength != len(encodedCommand) {
		return errors.New("incomplete command write")
	}

	if err := p.Sync(); err != nil {
		return err
	}

	return nil

}

func (p *FilePersister) Close() error {
	return p.file.Close()
}

func (p *FilePersister) Sync() error {
	return p.file.Sync()
}
