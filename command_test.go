package main

import (
	"strconv"
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

func TestCommandTTLExtraArgument(t *testing.T) {

	store := NewStore()

	parts := []string{
		"TTL",
		"TTLExtraArg",
		"abc",
	}

	_, err := executeCommand(parts, store)

	if err == nil {
		t.Fatal("expected error for extra argument")
	}

}

func TestCommandTTLMissingKey(t *testing.T) {

	store := NewStore()

	parts := []string{
		"TTL",
		"TTLKeyMissing",
	}

	value, err := executeCommand(parts, store)

	if err != nil {
		t.Fatal("expect to give correct status")
	}

	if value.Type != ResponseInteger {
		t.Fatalf("expected ResponseInteger, got %v", value.Type)
	}

	if value.Value != "-2" {
		t.Fatalf("expected value to be -2 but %s", value.Value)
	}

}

func TestCommandTTLNoExpiry(t *testing.T) {

	store := NewStore()

	store.Set("TTLNoExpiry", Entry{
		Value: "ValueTTLNoExpiry",
	})

	parts := []string{
		"TTL",
		"TTLNoExpiry",
	}

	value, err := executeCommand(parts, store)

	if err != nil {
		t.Fatal("expect to give correct status")
	}

	if value.Type != ResponseInteger {
		t.Fatalf("expected ResponseInteger, got %v", value.Type)
	}

	if value.Value != "-1" {
		t.Fatalf("expected value to be -1 but %s", value.Value)
	}
}

func TestCommandTTLHasExpiry(t *testing.T) {

	store := NewStore()

	store.Set("TTLHasExpiry", Entry{
		Value:     "ValueTTLHasExpiry",
		ExpiresAt: time.Now().Add(10 * time.Second),
	})

	parts := []string{
		"TTL",
		"TTLHasExpiry",
	}

	value, err := executeCommand(parts, store)

	if err != nil {
		t.Fatal("expect to give correct status")
	}

	if value.Type != ResponseInteger {
		t.Fatalf("expected ResponseInteger, got %v", value.Type)
	}

	remaining, err := strconv.Atoi(value.Value)

	if err != nil {
		t.Fatalf("expected TTL response to be an integer, got %q", value.Value)
	}

	if remaining <= 0 {
		t.Fatalf("expected value to be greater than 0 but %d", remaining)
	}

	if remaining > 10 {
		t.Fatalf("expected value to be smaller than 10 but %d", remaining)
	}
}

func TestCommandTTLExpiredKey(t *testing.T) {

	store := NewStore()

	store.Set("ActiveKeyButExpired", Entry{
		Value:     "ActiveKeyValueButExpired",
		ExpiresAt: time.Now().Add(-1 * time.Second),
	})

	parts := []string{
		"TTL",
		"ActiveKeyButExpired",
	}

	value, err := executeCommand(parts, store)

	if err != nil {
		t.Fatal("expect to give correct status")
	}

	if value.Type != ResponseInteger {
		t.Fatalf("expected ResponseInteger, got %v", value.Type)
	}

	if value.Value != "-2" {
		t.Fatalf("expected value to be -2 but %s", value.Value)
	}
}
