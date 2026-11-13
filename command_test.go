package main

import (
	"testing"
	"time"
)

func TestExecuteCommandExpireExistingKey(t *testing.T) {
	store := NewStore()

	store.Set("session", Entry{
		Value: "abc",
	})

	parts := []string{
		"EXPIRE",
		"session",
		"10",
	}

	result, err := executeCommand(parts, store)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Type != ResponseInteger {
		t.Fatalf("expected integer response, got %s", result.Type)
	}

	if result.Value != "1" {
		t.Fatalf("expected response value 1, got %s", result.Value)
	}
}

func TestExecuteCommandExpireExpiredKey(t *testing.T) {
	store := NewStore()

	store.Set("expiredKey", Entry{
		Value:     "expiredKeyCommand",
		ExpiresAt: time.Now().Add(-1 * time.Second),
	})

	parts := []string{
		"EXPIRE",
		"expiredKey",
		"10",
	}

	result, err := executeCommand(parts, store)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, exists := store.Get("expiredKey")

	if exists {
		t.Fatal("expected expired key to be removed")
	}

	if result.Type != ResponseInteger {
		t.Fatalf("expected integer response, got %s", result.Type)
	}

	if result.Value != "0" {
		t.Fatalf("expected response value 0, got %s", result.Value)
	}

}

func TestExecuteCommandInvalidDuration(t *testing.T) {

	store := NewStore()

	store.Set("invalidKey", Entry{
		Value: "invalidKeyValue",
	})

	parts := []string{
		"EXPIRE",
		"invalidKey",
		"abc",
	}

	_, err := executeCommand(parts, store)

	if err == nil {
		t.Fatal("expected error for invalid duration")
	}

	value, exists := store.Get("invalidKey")

	if !exists {
		t.Fatal("expected key to remain unchanged")
	}

	if value != "invalidKeyValue" {
		t.Fatalf("expected value invalidKeyValue, got %s", value)
	}

}

func TestExecuteCommandWrongArgumentCount(t *testing.T) {
	store := NewStore()

	store.Set("wrongArgument", Entry{
		Value: "wrongArgumentValue",
	})

	parts := []string{
		"EXPIRE",
		"wrongArgument",
		"10",
		"extra",
	}

	_, err := executeCommand(parts, store)

	if err == nil {
		t.Fatal("expected error for extra argument")
	}

	value, exists := store.Get("wrongArgument")

	if !exists {
		t.Fatal("expected key to remain unchanged")
	}

	if value != "wrongArgumentValue" {
		t.Fatalf("expected value wrongArgumentValue, got %s", value)
	}
}
