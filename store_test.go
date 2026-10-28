package main

import (
	"strconv"
	"sync"
	"testing"
)

func TestStoreSetAndGet(t *testing.T) {
	store := NewStore()

	store.Set("username", "alex")

	value, ok := store.Get("username")

	if !ok {
		t.Fatalf("expected value to be exist")
	}

	if value != "alex" {
		t.Errorf("expected alex %s", value)
	}

}

func TestStoreGetMissingKey(t *testing.T) {
	store := NewStore()

	value, ok := store.Get("userid")

	if ok {
		t.Fatalf("ok %t should be false", ok)
	}

	if value != "" {
		t.Errorf("expected value -1")
	}
}

func TestConcurrencyAccess(t *testing.T) {
	store := NewStore()

	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.Set("username", "alex")
		}()
	}

	wg.Wait()
}

func TestConcurrencyAccessMultipleKey(t *testing.T) {
	store := NewStore()

	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)

		go func(index int) {

			defer wg.Done()

			key := "key" + strconv.Itoa(index)
			value := "value" + strconv.Itoa(index)
			store.Set(key, value)
		}(i)
	}

	wg.Wait()

	for i := 0; i < 5; i++ {

		key := "key" + strconv.Itoa(i)
		expectedValue := "value" + strconv.Itoa(i)
		result, ok := store.Get(key)

		if !ok {
			t.Errorf("expected %s to exist", key)
			continue
		}

		if result != expectedValue {
			t.Errorf("expected %s, got %s", expectedValue, result)
		}

	}

}
