package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPersistenceWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "appendonly.aof")
	persister, err := NewFilePersister(path)

	if err != nil {
		t.Fatal(err)
	}

	defer persister.Close()

	command := []string{"SET", "foo", "bar"}

	err = persister.Append(command)

	if err != nil {
		t.Fatal(" error occurred during the append", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	expected := "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"
	contentString := string(content)

	if contentString != expected {
		t.Fatalf("expected %q, got %q", expected, contentString)
	}

}
