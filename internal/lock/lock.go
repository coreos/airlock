package lock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"go.etcd.io/etcd/clientv3"
)

const (
	keyTemplate  = "com.coreos.airlock/groups/%s/v1/semaphore"
	defaultGroup = "default"
)

var (
	// ErrNilManager is returned on nil manager
	ErrNilManager = errors.New("nil Manager")
)

// Manager takes care of locking for clients
type Manager struct {
	client  *clientv3.Client
	keyPath string
}

// NewManager returns a new lock manager, ensuring the underlying semaphore is initialized.
func NewManager(ctx context.Context, etcdURLs []string, group string, slots uint64) (*Manager, error) {
	// TODO(lucab): move to clientv3.New(clientv3.Config)
	client, err := clientv3.NewFromURL(etcdURLs[0])
	if err != nil {
		return nil, err
	}

	keyPath := fmt.Sprintf(keyTemplate, url.QueryEscape(group))
	manager := Manager{client, keyPath}

	if err := manager.ensureInit(ctx, slots); err != nil {
		return nil, err
	}

	return &manager, nil
}

// RecursiveLock adds this lock `id` as a holder of the semaphore
//
// It will return an error if there is a problem getting or setting the
// semaphore, or if the maximum number of holders has been reached.
func (m *Manager) RecursiveLock(ctx context.Context, id string) error {
	sem, version, err := m.get(ctx)
	if err != nil {
		return err
	}

	held, err := sem.RecursiveLock(id)
	if err != nil {
		return err
	}
	if held {
		return nil
	}

	if err := m.set(ctx, sem, version); err != nil {
		return err
	}

	return nil
}

// UnlockIfHeld removes this lock `id` as a holder of the semaphore
//
// It returns an error if there is a problem getting or setting the semaphore.
func (m *Manager) UnlockIfHeld(ctx context.Context, id string) error {
	sem, version, err := m.get(ctx)
	if err != nil {
		return err
	}

	if err := sem.UnlockIfHeld(id); err != nil {
		return err
	}

	if err := m.set(ctx, sem, version); err != nil {
		return err
	}

	return nil
}

// Close reaps all running goroutines
func (m *Manager) Close() {
	if m == nil {
		return
	}

	m.client.Close()
}

// ensureInit initialize the semaphore in etcd, if it does not exist yet
func (m *Manager) ensureInit(ctx context.Context, slots uint64) error {
	if m == nil {
		return ErrNilManager
	}

	sem := NewSemaphore(slots)
	semValue, err := sem.String()
	if err != nil {
		return err
	}

	_, err = m.client.Txn(ctx).If(
		// version=0 means that the key does not exist.
		clientv3.Compare(clientv3.Version(m.keyPath), "=", 0),
	).Then(
		clientv3.OpPut(m.keyPath, semValue),
	).Commit()

	if err != nil {
		return err
	}
	return nil
}

// get returns the current semaphore value and version, or an error
func (m *Manager) get(ctx context.Context) (*Semaphore, int64, error) {
	resp, err := m.client.Get(ctx, m.keyPath)
	if err != nil {
		return nil, 0, err
	}
	if resp.Count != 1 {
		return nil, 0, fmt.Errorf("unexpected number of results: %d", resp.Count)
	}

	var data []byte
	var version int64
	for _, kv := range resp.Kvs {
		data = kv.Value
		version = kv.Version
		break
	}
	if version == 0 {
		return nil, 0, errors.New("key at version 0")
	}
	if len(data) == 0 {
		return nil, 0, errors.New("empty semaphore value")
	}

	sem := &Semaphore{}
	err = json.Unmarshal(data, sem)
	if err != nil {
		return nil, 0, err
	}

	return sem, version, nil
}

// set updates the semaphore in etcd, if `version` matches the one previously observed
func (m *Manager) set(ctx context.Context, sem *Semaphore, version int64) error {
	if m == nil {
		return ErrNilManager
	}
	if sem == nil {
		return ErrNilSemaphore
	}

	data, err := json.Marshal(sem)
	if err != nil {
		return err
	}

	// Conditionally Put if version in etcd is still the same we observed.
	// If the condition is not met, the transaction will return as "not succeeding".
	resp, err := m.client.Txn(ctx).If(
		clientv3.Compare(clientv3.Version(m.keyPath), "=", version),
	).Then(
		clientv3.OpPut(m.keyPath, string(data)),
	).Commit()

	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return errors.New("conflict on semaphore detected, aborting")
	}

	return nil
}
