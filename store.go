package main

import (
	"errors"
	"strconv"
	"sync"
)

type Store struct {
	data  map[string]string
	mutex sync.RWMutex
}

func NewStore() *Store {
	return &Store{
		data: make(map[string]string),
	}
}

func (s *Store) Set(key string, value string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.data[key] = value

}

func (s *Store) Get(key string) (string, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	value, ok := s.data[key]
	return value, ok
}

func (s *Store) increment(key string) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	value, ok := s.data[key]

	if !ok {
		s.data[key] = "1"
		return 1, nil
	}

	integerValue, err := strconv.Atoi(value)

	if err != nil {
		return 0, errors.New("value is not a number")
	}

	integerValue++
	s.data[key] = strconv.Itoa(integerValue)

	return integerValue, nil

}
