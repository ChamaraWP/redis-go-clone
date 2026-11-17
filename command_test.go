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

func TestCommandDeleteExistingLiveKey(t *testing.T) {
	store := NewStore()

	store.Set("existingLiveKey", Entry{Value: "existingLiveValue"})

	parts := []string{
		"DEL",
		"existingLiveKey",
	}

	value, err := executeCommand(parts, store)

	if err != nil {
		t.Fatal("expect to give correct count")
	}

	if value.Type != ResponseInteger {
		t.Fatalf("expected ResponseInteger, got %v", value.Type)
	}

	if value.Value != "1" {
		t.Fatalf("expected deleted entry count to be 1 but %s", value.Value)
	}

	_, ok := store.Get("existingLiveKey")

	if ok {
		t.Fatal("expected entry to no to exists after delete")
	}

}

func TestCommandDeleteExpiredKey(t *testing.T) {
	store := NewStore()

	store.Set("expiredKey", Entry{Value: "expiredValue", ExpiresAt: time.Now().Add(-1 * time.Second)})

	parts := []string{
		"DEL",
		"expiredKey",
	}

	value, err := executeCommand(parts, store)

	if err != nil {
		t.Fatal("expect to give correct count")
	}

	if value.Value != "0" {
		t.Fatalf("expected deleted entry count to be 0 but %s", value.Value)
	}

	if value.Type != ResponseInteger {
		t.Fatalf("expected ResponseInteger, got %v", value.Type)
	}

	_, ok := store.Get("expiredKey")

	if ok {
		t.Fatal("expected entry to no to exists after delete")
	}

}

func TestCommandDeleteMixedKeys(t *testing.T) {
	store := NewStore()

	store.Set("a", Entry{Value: "A", ExpiresAt: time.Now().Add(10 * time.Second)})
	store.Set("b", Entry{Value: "B", ExpiresAt: time.Now().Add(-10 * time.Second)})
	store.Set("c", Entry{Value: "C"})

	parts := []string{
		"DEL",
		"a",
		"b",
		"c",
		"d",
	}

	value, err := executeCommand(parts, store)

	if err != nil {
		t.Fatal("expect to give correct count")
	}

	if value.Type != ResponseInteger {
		t.Fatalf("expected ResponseInteger, got %v", value.Type)
	}

	if value.Value != "2" {
		t.Fatalf("expected deleted entry count to be 2 but %s", value.Value)
	}

	_, okA := store.Get("a")
	_, okB := store.Get("b")
	_, okC := store.Get("c")

	if okA || okB || okC {
		t.Fatal("expect the entries to not to exists")
	}

}

func TestCommandDeleteMissingArguments(t *testing.T) {
	store := NewStore()

	store.Set("wrongArgsCount", Entry{Value: "wrongArgsCountValue"})

	parts := []string{
		"DEL",
	}

	_, err := executeCommand(parts, store)

	if err == nil {
		t.Fatal("expected error for insufficient argument")
	}

}

func TestCommandExistsLiveKey(t *testing.T) {
	store := NewStore()

	store.Set("existingKey", Entry{Value: "existingValue"})

	parts := []string{
		"EXISTS",
		"existingKey",
	}

	value, err := executeCommand(parts, store)

	if err != nil {
		t.Fatal("expect to give correct count")
	}

	if value.Type != ResponseInteger {
		t.Fatalf("expected ResponseInteger, got %v", value.Type)
	}

	if value.Value != "1" {
		t.Fatalf("expected entry count to be 1 but %s", value.Value)
	}

	_, ok := store.Get("existingKey")

	if !ok {
		t.Fatal("expected entry to exists")
	}

}

func TestCommandExistsMissingKey(t *testing.T) {
	store := NewStore()

	parts := []string{
		"EXISTS",
		"missingKey",
	}

	value, err := executeCommand(parts, store)

	if err != nil {
		t.Fatal("expect to give correct count")
	}

	if value.Type != ResponseInteger {
		t.Fatalf("expected ResponseInteger, got %v", value.Type)
	}

	if value.Value != "0" {
		t.Fatalf("expected entry count to be 0 but %s", value.Value)
	}

	_, ok := store.Get("missingKey")

	if ok {
		t.Fatal("expected entry to not exists")
	}

}

func TestCommandExistsExpiredKey(t *testing.T) {
	store := NewStore()

	store.Set("expiredKey", Entry{Value: "expiredValue", ExpiresAt: time.Now().Add(-1 * time.Second)})

	parts := []string{
		"EXISTS",
		"expiredKey",
	}

	value, err := executeCommand(parts, store)

	if err != nil {
		t.Fatal("expect to give correct count")
	}

	if value.Value != "0" {
		t.Fatalf("expected deleted entry count to be 0 but %s", value.Value)
	}

	if value.Type != ResponseInteger {
		t.Fatalf("expected ResponseInteger, got %v", value.Type)
	}

	_, ok := store.Get("expiredKey")

	if ok {
		t.Fatal("expected entry to no to exists after expired")
	}

}

func TestCommandExistsMixedKeys(t *testing.T) {
	store := NewStore()

	store.Set("a", Entry{Value: "A", ExpiresAt: time.Now().Add(10 * time.Second)})
	store.Set("b", Entry{Value: "B", ExpiresAt: time.Now().Add(-10 * time.Second)})
	store.Set("c", Entry{Value: "C"})

	parts := []string{
		"EXISTS",
		"a",
		"b",
		"c",
		"d",
	}

	value, err := executeCommand(parts, store)

	if err != nil {
		t.Fatal("expect to give correct count")
	}

	if value.Type != ResponseInteger {
		t.Fatalf("expected ResponseInteger, got %v", value.Type)
	}

	if value.Value != "2" {
		t.Fatalf("expected existing entry count to be 2 but got %s", value.Value)
	}

	_, okA := store.Get("a")
	_, okB := store.Get("b")
	_, okC := store.Get("c")

	if !okA || !okC {
		t.Fatal("expected live entries a and c to still exist")
	}

	if okB {
		t.Fatal("expect the entries to not to exists")
	}

}

func TestCommandExistsMissingArguments(t *testing.T) {
	store := NewStore()

	store.Set("wrongArgsCount", Entry{Value: "wrongArgsCountValue"})

	parts := []string{
		"EXISTS",
	}

	_, err := executeCommand(parts, store)

	if err == nil {
		t.Fatal("expected error for insufficient argument")
	}

}
