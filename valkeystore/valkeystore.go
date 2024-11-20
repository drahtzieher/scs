package valkeystore

import (
	"context"
	"time"

	"github.com/valkey-io/valkey-go"
)

// ValkeyStore represents the session store using Valkey.
type ValkeyStore struct {
	client valkey.Client
	prefix string
}

// New returns a new ValkeyStore instance.
func New(client valkey.Client) *ValkeyStore {
	return NewWithPrefix(client, "scs:session:")
}

// NewWithPrefix returns a new ValkeyStore instance with a custom prefix.
func NewWithPrefix(client valkey.Client, prefix string) *ValkeyStore {
	return &ValkeyStore{
		client: client,
		prefix: prefix,
	}
}

// FindCtx retrieves the data for a given session token.
// If the session token is not found or expired, `exists` will be false.
func (v *ValkeyStore) FindCtx(ctx context.Context, token string) (b []byte, exists bool, err error) {
	cmd := v.client.B().
		Get().
		Key(v.prefix + token).
		Build()

	result := v.client.Do(ctx, cmd)

	// Handle non-Valkey errors
	if err := result.NonValkeyError(); err != nil {
		return nil, false, err
	}

	// Retrieve the value
	b, err = result.AsBytes()
	if err != nil {
		// Missing key case
		if b == nil {
			return nil, false, nil
		}
		return nil, false, err
	}

	return b, true, nil
}

// CommitCtx adds a session token and data to the ValkeyStore instance with the
// given expiry time.
func (v *ValkeyStore) CommitCtx(ctx context.Context, token string, b []byte, expiry time.Time) error {
	cmds := make(valkey.Commands, 0, 2)

	cmds = append(cmds, v.client.B().Set().
		Key(v.prefix+token).
		Value(string(b)).
		Build())

	cmds = append(cmds, v.client.B().Expire().
		Key(v.prefix+token).
		Seconds(int64(time.Until(expiry).Seconds())).
		Build())

	for _, resp := range v.client.DoMulti(ctx, cmds...) {
		if err := resp.Error(); err != nil {
			return err
		}
	}

	return nil
}

// DeleteCtx removes a session token and its associated data.
func (v *ValkeyStore) DeleteCtx(ctx context.Context, token string) error {
	cmd := v.client.B().
		Del().
		Key(v.prefix + token).
		Build()

	return v.client.Do(ctx, cmd).Error()
}

// AllCtx retrieves all active sessions as a map of token to data.
// Uses Scan for efficient pagination of large datasets.
func (v *ValkeyStore) AllCtx(ctx context.Context) (map[string][]byte, error) {
	sessions := make(map[string][]byte)
	var cursor uint64

	for {
		// Build scan command with cursor
		cmd := v.client.B().
			Scan().
			Cursor(cursor).
			Match(v.prefix + "*").
			Count(100). // Adjust batch size as needed
			Build()

		resp := v.client.Do(ctx, cmd)
		if err := resp.Error(); err != nil {
			return nil, err
		}

		scanResult, ok := resp.AsScanEntry()
		if ok == nil {
			// If we can't get a scan result, return empty map
			return sessions, nil
		}

		// Process this batch of keys
		for _, key := range scanResult.Elements {
			token := key[len(v.prefix):]
			data, exists, err := v.FindCtx(ctx, token)
			if err != nil {
				return nil, err
			}
			if exists {
				sessions[token] = data
			}
		}

		// Update cursor for next iteration
		cursor = scanResult.Cursor

		// Exit if we've processed all keys

		cursor = scanResult.Cursor
		if cursor == 0 {
			break
		}

	}

	return sessions, nil
}

// Plain Store methods to avoid compile-time errors. These methods panic as they
// require a context argument for proper operation.
func (v *ValkeyStore) Find(token string) ([]byte, bool, error) {
	panic("missing context arg")
}

func (v *ValkeyStore) Commit(token string, b []byte, expiry time.Time) error {
	panic("missing context arg")
}

func (v *ValkeyStore) Delete(token string) error {
	panic("missing context arg")
}
