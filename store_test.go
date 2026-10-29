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

func TestStoreIncrement(t *testing.T) {
	store := NewStore()

	store.Set("incr", "11")

	value, err := store.increment("incr")

	if err != nil {
		t.Fatalf("increment failed")
	}

	if value != 12 {
		t.Errorf("expected value 12 but received %d", value)
	}

	storedValue, ok := store.Get("incr")

	if !ok {
		t.Fatal("expected key to exist")
	}

	if storedValue != "12" {
		t.Errorf("expected stored value 12, got %s", storedValue)
	}

}

func TestStoreIncrementNonNumeric(t *testing.T) {
	store := NewStore()

	store.Set("counter", "hello")

	_, err := store.increment("counter")

	if err == nil {
		t.Fatal("expected increment to return an error for non-numeric value")
	}

	storedValue, ok := store.Get("counter")

	if !ok {
		t.Fatal("expected key to exist")
	}

	if storedValue != "hello" {
		t.Errorf("expected stored value remain hello, got %s", storedValue)
	}

}

func TestAutoIncrementMissingKey(t *testing.T) {

	store := NewStore()

	value, err := store.increment("incr")

	if err != nil {
		t.Fatal("increment failed for missing key")
	}

	if value != 1 {
		t.Errorf("expected stored value to be 1, got %d", value)
	}

	storedValue, ok := store.Get("incr")

	if !ok {
		t.Fatal("expected incr key to be created")
	}

	if storedValue != "1" {
		t.Errorf("expected stored value 1, got %s", storedValue)
	}

}

func TestStoreConcurrentIncrement(t *testing.T) {

	store := NewStore()

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, err := store.increment("counter")

			if err != nil {
				t.Errorf("increment failed: %v", err)
			}

		}()

	}

	wg.Wait()

	value, ok := store.Get("counter")

	if !ok {
		t.Fatal("expected counter to exist")
	}

	if value != "100" {
		t.Errorf("expected counter to be 100, got %s", value)
	}

}
