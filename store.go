package main

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type TTLStatus int

const (
	TTLKeyMissing TTLStatus = iota
	TTLNoExpiry
	TTLHasExpiry
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
		println("expired key found %s", key)
		delete(s.data, key)
		return "", false
	}

	return entry.Value, ok
}

func (s *Store) Increment(key string) (int, error) {
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

func (s *Store) Expire(key string, duration time.Duration) bool {

	s.mutex.Lock()

	defer s.mutex.Unlock()

	entry, ok := s.data[key]

	if !ok {
		return false
	}

	now := time.Now()

	if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {
		delete(s.data, key)
		return false
	}

	if duration <= 0 {
		delete(s.data, key)
		return true
	}

	entry.ExpiresAt = now.Add(duration)
	fmt.Printf("expiry set to %+v\n", entry)

	s.data[key] = entry
	return true

}

func (s *Store) TTL(key string) (time.Duration, TTLStatus) {
	s.mutex.Lock()

	defer s.mutex.Unlock()

	entry, ok := s.data[key]

	if !ok {
		return 0, TTLKeyMissing
	}

	if entry.ExpiresAt.IsZero() {
		return 0, TTLNoExpiry
	}

	now := time.Now()

	if now.After(entry.ExpiresAt) {
		delete(s.data, key)
		return 0, TTLKeyMissing
	}

	remaining := time.Until(entry.ExpiresAt)

	return remaining, TTLHasExpiry
}

func (s *Store) Delete(keys ...string) int {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var count = 0
	now := time.Now()

	for _, key := range keys {

		entry, ok := s.data[key]

		if !ok {
			continue
		}

		if !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt) {
			delete(s.data, key)
			continue
		}

		delete(s.data, key)
		count++

	}

	return count
}
