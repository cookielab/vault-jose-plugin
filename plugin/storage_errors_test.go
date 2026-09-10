package josejwt

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
)

type failingWriteStorage struct {
	logical.InmemStorage
	err error
}

func (s *failingWriteStorage) Put(ctx context.Context, entry *logical.StorageEntry) error {
	if s.err != nil {
		return s.err
	}
	return s.InmemStorage.Put(ctx, entry)
}

func TestSavePreservesStorageError(t *testing.T) {
	for _, path := range []string{"roles/test", "jwks/test", "jwks/test/key"} {
		t.Run(path, func(t *testing.T) {
			storageErr := errors.New("storage write failed")
			storage := &failingWriteStorage{err: storageErr}
			config := logical.TestBackendConfig()
			config.StorageView = storage
			backend, err := Factory(context.Background(), config)
			if err != nil {
				t.Fatal(err)
			}
			response, err := backend.HandleRequest(context.Background(), &logical.Request{
				Operation: logical.UpdateOperation,
				Path:      path,
				Storage:   storage,
				Data: map[string]interface{}{
					"type": "jwt", "key_set": "test", "alg": "RS256", "use": "sig",
				},
			})
			if !errors.Is(err, storageErr) {
				t.Fatalf("storage error was lost: %v", err)
			}
			if response != nil {
				t.Fatal("storage failure must not be masked by a logical error response")
			}
		})
	}
}

func TestDeleteKeyPreservesStorageError(t *testing.T) {
	ctx := context.Background()
	storage := &failingWriteStorage{}
	config := logical.TestBackendConfig()
	config.StorageView = storage
	backend, err := Factory(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	response, err := backend.HandleRequest(ctx, &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "jwks/test/key",
		Storage:   storage,
		Data:      map[string]interface{}{"alg": "RS256", "use": "sig"},
	})
	if err != nil || response == nil || response.IsError() {
		t.Fatalf("could not create test key: %v", err)
	}
	storage.err = errors.New("storage write failed")
	response, err = backend.HandleRequest(ctx, &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      "jwks/test/key",
		Storage:   storage,
	})
	if !errors.Is(err, storage.err) || response != nil {
		t.Fatalf("delete masked storage error: %v", err)
	}
	storage.err = nil
	response, err = backend.HandleRequest(ctx, &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "jwks/test/key",
		Storage:   storage,
	})
	if err != nil || response == nil || response.IsError() {
		t.Fatalf("failed delete changed the stored key: %v", err)
	}
}
