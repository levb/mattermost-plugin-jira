// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package instance

import (
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/store"
)

const disablePrefixForInstance = false

const (
	keyCurrentInstance = "current_jira_instance"
	keyKnownInstances  = "known_jira_instances"
	prefixInstance     = "jira_instance_"
)

type InstanceStore interface {
	Store(Instance) error
	Delete(string) error
	Load(string, interface{}) error
	LoadRaw(string) ([]byte, error)

	CurrentInstanceStore
}

type CurrentInstanceStore interface {
	StoreCurrentInstance(Instance) error
	LoadCurrentInstanceRaw() ([]byte, error)
}

type instanceStore struct {
	currentStore store.Store
	store        store.Store
}

var _ InstanceStore = (*instanceStore)(nil)

func NewInstanceStore(s store.Store) InstanceStore {
	return &instanceStore{
		currentStore: s,
		store:        store.NewHashedKeyStore(s, prefixInstance),
	}
}

func (s instanceStore) Load(key string, instanceRef interface{}) error {
	err := store.LoadJSON(s.store, key, instanceRef)
	if err != nil {
		return errors.WithMessagef(err, "failed to load instance %q", key)
	}
	return nil
}

func (s instanceStore) LoadRaw(key string) ([]byte, error) {
	return s.store.Load(key)
}

func (s instanceStore) Store(instance Instance) (returnErr error) {
	err := store.StoreJSON(s.store, instance.GetURL(), instance)
	if err != nil {
		return errors.WithMessagef(err, "failed to store instance %q", instance.GetURL())
	}

	// Update known instances
	known, err := s.LoadKnownInstances()
	if err != nil {
		return err
	}
	known[instance.GetURL()] = instance.GetType()
	err = s.StoreKnownInstances(known)
	if err != nil {
		return err
	}
	return nil
}

func (s instanceStore) Delete(key string) (returnErr error) {
	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessagef(returnErr, "failed to delete instance:%v", key)
		}
	}()

	// Delete the instance.
	err := s.store.Delete(key)
	if err != nil {
		return err
	}

	// Update known instances
	known, err := s.LoadKnownInstances()
	if err != nil {
		return err
	}
	delete(known, key)
	err = s.StoreKnownInstances(known)
	if err != nil {
		return err
	}

	basic := BasicInstance{}
	// Remove the current instance if it matches the deleted
	err = store.LoadJSON(s.store, keyCurrentInstance, &basic)
	if err != nil {
		return err
	}
	if basic.InstanceKey == key {
		err = s.store.Delete(keyCurrentInstance)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s instanceStore) StoreKnownInstances(known map[string]string) error {
	err := store.StoreJSON(s.store, keyKnownInstances, known)
	if err != nil {
		return errors.WithMessagef(err,
			"failed to store known Jira instances %+v", known)
	}
	return nil
}

func (s instanceStore) LoadKnownInstances() (map[string]string, error) {
	known := map[string]string{}
	err := store.LoadJSON(s.store, keyKnownInstances, &known)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load known Jira instances")
	}
	return known, nil
}

func (s instanceStore) StoreCurrentInstance(instance Instance) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
	}()
	err := store.StoreJSON(s.store, keyCurrentInstance, instance)
	if err != nil {
		return errors.WithMessagef(err, "failed to store current Jira instance %q", instance.GetURL())
	}
	return nil
}

func (s instanceStore) LoadCurrentInstanceRaw() ([]byte, error) {
	return s.store.Load(keyCurrentInstance)
}
