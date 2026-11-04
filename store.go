package main

import (
	"errors"
	"strconv"
	"sync"
	"time"
)

type Entry struct {
	Value     string
	ExpiresAt time.Time
}

type Store struct {
	data  map[string]Entry
	mutex sync.RWMutex
}

func NewStore() *Store {
	return &Store{
		data: make(map[string]Entry),
	}
}

func (s *Store) Set(key string, value Entry) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.data[key] = value

}

func (s *Store) Get(key string) (string, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	entry, ok := s.data[key]

	if !ok {
		return "", false
	}

	if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {

		delete(s.data, key)

		return "", false

	}

	return entry.Value, ok
}

func (s *Store) increment(key string) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	entry, ok := s.data[key]

	if !ok || (!entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt)) {
		s.data[key] = Entry{
			Value: "1",
		}
		return 1, nil
	}

	integerValue, err := strconv.Atoi(entry.Value)

	if err != nil {
		return 0, errors.New("value is not a number")
	}

	integerValue++
	s.data[key] = Entry{
		Value:     strconv.Itoa(integerValue),
		ExpiresAt: entry.ExpiresAt,
	}

	return integerValue, nil

}
