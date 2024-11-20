package valkeystore

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/valkey-io/valkey-go"
)

func setupTestStore(t *testing.T) (*ValkeyStore, valkey.Client) {
	t.Helper()

	// Retrieve or set default DSN
	dsn := os.Getenv("SCS_VALKEY_TEST_DSN")
	if dsn == "" {
		dsn = "redis://localhost:6379/0" // Default DSN for local testing
	}

	// Parse DSN using MustParseURL
	client, err := valkey.NewClient(valkey.MustParseURL(dsn))
	if err != nil {
		t.Fatalf("failed to create Valkey client: %v", err)
	}

	// Create ValkeyStore instance
	store := New(client)

	// Flush database to ensure clean state
	cmd := client.B().Flushdb().Build()
	if err := client.Do(context.Background(), cmd).Error(); err != nil {
		t.Fatalf("failed to flush database: %v", err)
	}

	return store, client
}

func TestFind(t *testing.T) {
	store, client := setupTestStore(t)
	defer client.Close()

	ctx := context.Background()

	// Add test value
	setCmd := client.B().Set().Key(store.prefix + "session_token").Value("encoded_data").Build()
	if err := client.Do(ctx, setCmd).Error(); err != nil {
		t.Fatal(err)
	}

	// Test FindCtx
	b, found, err := store.FindCtx(ctx, "session_token")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatalf("expected session_token to be found")
	}
	if !bytes.Equal(b, []byte("encoded_data")) {
		t.Fatalf("expected %v, got %v", []byte("encoded_data"), b)
	}
}

func TestSaveNew(t *testing.T) {
	store, client := setupTestStore(t)
	defer client.Close()

	ctx := context.Background()

	// Test CommitCtx
	err := store.CommitCtx(ctx, "session_token", []byte("encoded_data"), time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	// Verify stored value
	cmd := client.B().Get().Key(store.prefix + "session_token").Build()
	resp := client.Do(ctx, cmd)
	data, err := resp.AsBytes()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(data, []byte("encoded_data")) {
		t.Fatalf("expected %v, got %v", []byte("encoded_data"), data)
	}
}

func TestExpiry(t *testing.T) {
	store, client := setupTestStore(t)
	defer client.Close()

	ctx := context.Background()

	// Commit a session token with a short expiry
	err := store.CommitCtx(ctx, "session_token", []byte("encoded_data"), time.Now().Add(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}

	// Verify existence before expiry
	_, found, _ := store.FindCtx(ctx, "session_token")
	if !found {
		t.Fatalf("expected session_token to exist")
	}

	// Wait for expiry and verify non-existence
	time.Sleep(200 * time.Millisecond)
	_, found, _ = store.FindCtx(ctx, "session_token")
	if found {
		t.Fatalf("expected session_token to be expired")
	}
}
