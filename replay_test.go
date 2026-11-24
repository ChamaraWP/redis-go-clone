package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestReplayAOFRebuildsStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "appendonlytest.aof")

	persister, err := NewFilePersister(path)
	if err != nil {
		t.Fatal(err)
	}

	commands := [][]string{
		{"SET", "foo", "10"},
		{"INCR", "foo"},
		{"SET", "bar", "hello"},
		{"DEL", "bar"},
	}

	for _, command := range commands {
		if err := persister.Append(command); err != nil {
			t.Fatal(err)
		}
	}

	if err := persister.Close(); err != nil {
		t.Fatal(err)
	}

	store := NewStore()

	if err := replayAOF(path, store); err != nil {
		t.Fatal(err)
	}

	foo, ok := store.Get("foo")
	if !ok {
		t.Fatal("expected foo to exist after replay")
	}

	if foo != "11" {
		t.Fatalf("expected foo to be %q, got %q", "11", foo)
	}

	_, ok = store.Get("bar")
	if ok {
		t.Fatal("expected bar to be deleted after replay")
	}
}

func TestReplayAOFRestoresExpireAt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "appendonlytest.aof")

	persister, err := NewFilePersister(path)
	if err != nil {
		t.Fatal(err)
	}

	expiresAt := time.Now().Add(30 * time.Second).Unix()

	commands := [][]string{
		{"SET", "temp", "value"},
		{"EXPIREAT", "temp", strconv.FormatInt(expiresAt, 10)},
	}

	for _, command := range commands {
		if err := persister.Append(command); err != nil {
			t.Fatal(err)
		}
	}

	if err := persister.Close(); err != nil {
		t.Fatal(err)
	}

	store := NewStore()

	if err := replayAOF(path, store); err != nil {
		t.Fatal(err)
	}

	value, ok := store.Get("temp")
	if !ok {
		t.Fatal("expected temp to exist after replay")
	}

	if value != "value" {
		t.Fatalf("expected value %q, got %q", "value", value)
	}

	remaining, status := store.TTL("temp")

	if status != TTLHasExpiry {
		t.Fatalf("expected temp to have an expiry, got status %v", status)
	}

	if remaining <= 0 {
		t.Fatalf("expected positive TTL after replay, got %v", remaining)
	}
}

func TestConcurrentPersistenceCanBeReplayed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "appendonly.aof")

	persister, err := NewFilePersister(path)
	if err != nil {
		t.Fatal(err)
	}

	const workers = 100

	var wg sync.WaitGroup
	errCh := make(chan error, workers)

	for i := range workers {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			key := fmt.Sprintf("key-%d", i)
			value := fmt.Sprintf("value-%d", i)

			command := []string{
				"SET",
				key,
				value,
			}

			if err := persister.Append(command); err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatal(err)
	}

	if err := persister.Close(); err != nil {
		t.Fatal(err)
	}

	store := NewStore()

	if err := replayAOF(path, store); err != nil {
		t.Fatalf("replay failed: %v", err)
	}

	for i := range workers {
		key := fmt.Sprintf("key-%d", i)
		expected := fmt.Sprintf("value-%d", i)

		value, ok := store.Get(key)
		if !ok {
			t.Fatalf("expected key %q to exist after replay", key)
		}

		if value != expected {
			t.Fatalf(
				"expected key %q to contain %q, got %q",
				key,
				expected,
				value,
			)
		}
	}
}

func TestReplayAOFRejectsTruncatedCommand(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "appendonly.aof")

	content := "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$5\r\nba"

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	store := NewStore()

	err := replayAOF(path, store)

	if err == nil {
		t.Fatal("expected replay to fail for truncated AOF")
	}
}

func TestReplayAOFStopsOnCorruptionAfterValidCommand(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "appendonly.aof")

	validCommand := "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"
	truncatedCommand := "*3\r\n$3\r\nSET\r\n$3\r\nbaz\r\n$5\r\nqu"

	content := validCommand + truncatedCommand

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	store := NewStore()

	err := replayAOF(path, store)

	if err == nil {
		t.Fatal("expected replay to fail because AOF contains a truncated command")
	}

	value, ok := store.Get("foo")
	if !ok {
		t.Fatal("expected valid command before corruption to be replayed")
	}

	if value != "bar" {
		t.Fatalf("expected foo to be %q, got %q", "bar", value)
	}

	_, ok = store.Get("baz")
	if ok {
		t.Fatal("expected truncated command not to be applied")
	}
}
