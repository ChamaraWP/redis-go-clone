package main

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestStoreSetAndGet(t *testing.T) {
	store := NewStore()

	store.Set("username", Entry{Value: "alex", ExpiresAt: time.Now().Add(10 * time.Second)})

	value, ok := store.Get("username")

	if !ok {
		t.Fatalf("expected value to be exist")
	}

	if value != "alex" {
		t.Errorf("expected alex %s", value)
	}

}

func TestStoreSetAndGetWithTtl(t *testing.T) {
	store := NewStore()

	store.Set("username", Entry{Value: "alex", ExpiresAt: time.Now().Add(2 * time.Second)})

	value, ok := store.Get("username")

	if !ok {
		t.Fatalf("expected value to be exist")
	}

	if value != "alex" && value != "" {
		t.Errorf("expected alex %s and should not be empty", value)
	}

}

func TestStoreTTLExpiration(t *testing.T) {
	store := NewStore()

	store.Set("username", Entry{Value: "alex", ExpiresAt: time.Now().Add(-1 * time.Second)})

	value, ok := store.Get("username")

	if ok {
		t.Fatal("expected key to be expired")
	}

	if value != "" {
		t.Errorf("expected empty value, got %s", value)
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
			store.Set("username", Entry{Value: "alex"})
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
			store.Set(key, Entry{Value: value})
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

	store.Set("incr", Entry{Value: "11"})

	persister, err := NewFilePersister("appendonlytest.aof")
	if err != nil {
		t.Fatal("file initialization failed", err)
	}
	defer persister.Close()

	value, err := store.Increment("incr", func() error {
		return persister.Append([]string{"INCR", "incr"})
	})

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

	store.Set("counter", Entry{Value: "hello"})

	persister, err := NewFilePersister("appendonlytest.aof")
	if err != nil {
		t.Fatal("file initialization failed", err)
	}
	defer persister.Close()

	_, err = store.Increment("counter", func() error {
		return persister.Append([]string{"INCR", "counter"})
	})

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

	persister, err := NewFilePersister("appendonlytest.aof")
	if err != nil {
		t.Fatal("file initialization failed", err)
	}
	defer persister.Close()

	value, err := store.Increment("incr", func() error {
		return persister.Append([]string{"INCR", "incr"})
	})

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

	persister, err := NewFilePersister("appendonlytest.aof")
	if err != nil {
		t.Fatal("file initialization failed", err)
	}
	defer persister.Close()

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, err := store.Increment("counter", func() error {
				return persister.Append([]string{"INCR", "counter"})
			})

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

func TestStoreIncrementWithNoExpiredTtl(t *testing.T) {
	store := NewStore()

	expiresAt := time.Now().Add(5 * time.Second)

	store.Set("incr", Entry{Value: "11", ExpiresAt: expiresAt})

	persister, err := NewFilePersister("appendonlytest.aof")
	if err != nil {
		t.Fatal("file initialization failed", err)
	}
	defer persister.Close()

	value, err := store.Increment("incr", func() error {
		return persister.Append([]string{"INCR", "incr"})
	})

	if err != nil {
		t.Fatalf("increment failed")
	}

	if value != 12 {
		t.Errorf("expected value 12 but received %d", value)
	}

	store.mutex.RLock()
	entry := store.data["incr"]
	store.mutex.RUnlock()

	if entry.Value != "12" {
		t.Fatal("incremented value should exists")
	}

	if !entry.ExpiresAt.Equal(expiresAt) {
		t.Errorf("expected expiry to remain unchanged")
	}
}

func TestStoreIncrementWithExpiredTtl(t *testing.T) {
	store := NewStore()

	expiresAt := time.Now().Add(-1 * time.Second)

	store.Set("incr", Entry{Value: "11", ExpiresAt: expiresAt})

	persister, err := NewFilePersister("appendonlytest.aof")
	if err != nil {
		t.Fatal("file initialization failed", err)
	}
	defer persister.Close()

	value, err := store.Increment("incr", func() error {
		return persister.Append([]string{"INCR", "incr"})
	})

	if err != nil {
		t.Fatalf("increment failed")
	}

	if value != 1 {
		t.Errorf("expected value 1 but received %d", value)
	}

	store.mutex.RLock()
	entry, ok := store.data["incr"]
	store.mutex.RUnlock()

	if !ok {
		t.Fatal("expected incr key to exist")
	}

	if entry.Value != "1" {
		t.Errorf("expected stored value 1, got %s", entry.Value)
	}

	if !entry.ExpiresAt.IsZero() {
		t.Errorf("expected expiry to be zero")
	}
}

func TestStoreIncrementWithNonNumericWithTtl(t *testing.T) {
	store := NewStore()

	expiresAt := time.Now().Add(5 * time.Second)

	store.Set("incr", Entry{Value: "hello", ExpiresAt: expiresAt})

	persister, err := NewFilePersister("appendonlytest.aof")
	if err != nil {
		t.Fatal("file initialization failed", err)
	}
	defer persister.Close()

	_, err = store.Increment("incr", func() error {
		return persister.Append([]string{"INCR", "incr"})
	})

	if err == nil {
		t.Fatal("expected increment to return an error for non-numeric value")
	}

	store.mutex.RLock()
	entry, ok := store.data["incr"]
	store.mutex.RUnlock()

	if !ok {
		t.Fatal("expected incr key to exist")
	}

	if entry.Value != "hello" {
		t.Errorf("expected stored value hello, got %s", entry.Value)
	}

	if !entry.ExpiresAt.Equal(expiresAt) {
		t.Errorf("expected expiry to remain unchanged")
	}
}

func TestExistingKeyPositiveDuration(t *testing.T) {

	store := NewStore()

	store.Set("session", Entry{Value: "testingExp"})

	ok := store.Expire("session", 20*time.Second)

	if !ok {
		t.Fatal("expected EXPIRE to succeed")
	}

	store.mutex.RLock()
	entry, ok := store.data["session"]
	store.mutex.RUnlock()

	if !ok {
		t.Fatal("expected key to exists")
	}

	if entry.ExpiresAt.IsZero() {
		t.Fatal("expected Expire At to exists")
	}

	if !entry.ExpiresAt.After(time.Now()) {
		t.Fatal("expected expiry to be in the future")
	}

}

func TestStoreExpireWithMissingKey(t *testing.T) {
	store := NewStore()

	ok := store.Expire("noKey", 20*time.Second)

	if ok {
		t.Fatal("expected EXPIRE to return false for missing key")
	}

	_, exists := store.Get("noKey")

	if exists {
		t.Fatal("expected missing key to remain missing")
	}
}

func TestStoreAlreadyExpiredKey(t *testing.T) {
	store := NewStore()

	store.Set("expiredSession", Entry{
		Value:     "ExpiredSessionValue",
		ExpiresAt: time.Now().Add(-1 * time.Second),
	})

	ok := store.Expire("expiredSession", 10*time.Second)

	if ok {
		t.Fatal("expected EXPIRE to return false expired entry")
	}

	_, ok = store.Get("expiredSession")

	if ok {
		t.Fatal("expected expired key to be removed")
	}
}

func TestStoreWithExistingKeyWithZeroOrMinusTtl(t *testing.T) {

	store := NewStore()

	store.Set("session", Entry{
		Value: "SessionValue",
	})

	ok := store.Expire("session", 0)

	if !ok {
		t.Fatal("expected EXPIRE to succeed for existing key")
	}

	_, exists := store.Get("session")

	if exists {
		t.Fatal("expected key to be removed after EXPIRE with zero TTL")
	}

}

func TestStoreTtlWithMissingKey(t *testing.T) {
	store := NewStore()

	remaining, status := store.TTL("missingKey")

	if remaining != 0 || status != TTLKeyMissing {
		t.Fatalf(
			"expected remaining=0 and status=TTLKeyMissing, got remaining=%v status=%v",
			remaining,
			status,
		)
	}

}

func TestStoreTtlWithNoExpiry(t *testing.T) {
	store := NewStore()

	store.Set("KeyWithNoExpiry", Entry{
		Value: "ValueWithNoExpiry",
	})

	remaining, status := store.TTL("KeyWithNoExpiry")

	if remaining != 0 || status != TTLNoExpiry {
		t.Fatalf(
			"expected remaining=0 and status=TTLNoExpiry, got remaining=%v status=%v",
			remaining,
			status,
		)
	}
}

func TestStoreTtlWithActiveExpiry(t *testing.T) {
	store := NewStore()

	store.Set("KeyWithActiveExpiry", Entry{
		Value:     "ValueWithActiveExpiry",
		ExpiresAt: time.Now().Add(10 * time.Second),
	})

	remaining, status := store.TTL("KeyWithActiveExpiry")

	if status != TTLHasExpiry {
		t.Fatalf("expected TTLHasExpiry, got %v", status)
	}

	if remaining <= 0 {
		t.Fatalf("expected positive remaining TTL, got %v", remaining)
	}

	if remaining > 10*time.Second {
		t.Fatalf("expected remaining TTL <= 10s, got %v", remaining)
	}

}

func TestStoreTtlWithHasExpireValueButExpired(t *testing.T) {

	store := NewStore()

	store.Set("KeyWithExpiryButExpired", Entry{
		Value:     "ValueKeyWithExpiryButExpired",
		ExpiresAt: time.Now().Add(-1 * time.Second),
	})

	remaining, status := store.TTL("KeyWithExpiryButExpired")

	if remaining != 0 || status != TTLKeyMissing {
		t.Fatalf(
			"expected remaining=0 and status=TTLKeyMissing, got remaining=%v status=%v",
			remaining,
			status,
		)
	}

	_, exists := store.Get("KeyWithExpiryButExpired")

	if exists {
		t.Fatal("expected expired key to be removed")
	}
}

func TestStoreDeleteExistingLiveKey(t *testing.T) {
	store := NewStore()

	store.Set("existingLiveKey", Entry{Value: "existingLiveValue"})

	count := store.Delete("existingLiveKey")

	if count != 1 {
		t.Fatalf("expected deleted entry count to be 1 but %d", count)
	}

	_, ok := store.Get("existingLiveKey")

	if ok {
		t.Fatal("expected entry to no to exists after delete")
	}

}

func TestStoreDeleteMissingKey(t *testing.T) {
	store := NewStore()

	count := store.Delete("missingKey")

	if count != 0 {
		t.Fatalf("expected count be 0 but %d", count)
	}

}

func TestStoreDeleteExpiredKey(t *testing.T) {
	store := NewStore()

	store.Set("expiredKey", Entry{Value: "expiredKeyValue", ExpiresAt: time.Now().Add(-1 * time.Second)})

	count := store.Delete("expiredKey")

	if count != 0 {
		t.Fatalf("expected count be 0 but %d", count)
	}

	_, ok := store.Get("expiredKey")

	if ok {
		t.Fatal("expect the entry to not to exists")
	}

}

func TestStoreDeleteMixedKeys(t *testing.T) {
	store := NewStore()

	store.Set("a", Entry{Value: "A", ExpiresAt: time.Now().Add(10 * time.Second)})
	store.Set("b", Entry{Value: "B", ExpiresAt: time.Now().Add(-10 * time.Second)})
	store.Set("c", Entry{Value: "c"})

	count := store.Delete("a", "b", "c", "d")

	if count != 2 {
		t.Fatalf("expected count be 2 but %d", count)
	}

	_, okA := store.Get("a")
	_, okB := store.Get("b")
	_, okC := store.Get("c")

	if okA || okB || okC {
		t.Fatal("expect the entries to not to exists")
	}

}
func TestStoreExistsLiveKey(t *testing.T) {
	store := NewStore()

	store.Set("existsLiveKey", Entry{
		Value: "existsLiveValue",
	})

	count := store.Exists("existsLiveKey")

	if count != 1 {
		t.Fatalf("expected exists entry count to be 1 but %d", count)
	}

}

func TestStoreExistsMissingKey(t *testing.T) {
	store := NewStore()

	count := store.Exists("missingKey")

	if count != 0 {
		t.Fatalf("expected exists entry count to be 0 but %d", count)
	}

}

func TestStoreExistsExpiredKey(t *testing.T) {
	store := NewStore()

	store.Set("expiredKey", Entry{
		Value:     "expiredValue",
		ExpiresAt: time.Now().Add(-1 * time.Second),
	})

	count := store.Exists("expiredKey")

	if count != 0 {
		t.Fatalf("expected expired entry count to be 0 but %d", count)
	}

	_, ok := store.Get("expiredKey")

	if ok {
		t.Fatal("expect entry to not to exists")
	}

}

func TestStoreExistsMixedKeys(t *testing.T) {
	store := NewStore()

	store.Set("a", Entry{Value: "A", ExpiresAt: time.Now().Add(10 * time.Second)})
	store.Set("b", Entry{Value: "B", ExpiresAt: time.Now().Add(-10 * time.Second)})
	store.Set("c", Entry{Value: "c"})

	count := store.Exists("a", "b", "c", "d")

	if count != 2 {
		t.Fatalf("expected count be 2 but %d", count)
	}

	_, okA := store.Get("a")
	_, okB := store.Get("b")
	_, okC := store.Get("c")

	if !okA || !okC {
		t.Fatal("expect the entries to exists")
	}

	if okB {
		t.Fatal("expect the entry to no to exists")
	}

}
