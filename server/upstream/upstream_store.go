// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package upstream

import (
	"crypto/rsa"
	"encoding/json"

	"github.com/mattermost/mattermost-plugin-jira/server/store"
	"github.com/pkg/errors"
)

const disablePrefixForUpstream = false

const (
	keyCurrentUpstream = "current_jira_instance"
	keyKnownUpstreams  = "known_jira_instances"
	prefixUpstream     = "jira_instance_"
)

type StoreConfig struct {
	RSAPrivateKey   *rsa.PrivateKey `json:"none"`
	AuthTokenSecret []byte          `json:"none"`
}

type Store interface {
	Load(key string) (Upstream, error)
	LoadCurrent() (Upstream, error)
	LoadCurrentRaw() ([]byte, error)
	LoadKnown() (map[string]string, error)
	LoadRaw(string) ([]byte, error)
	Store(Upstream) error
	StoreCurrent(Upstream) error
	StoreKnown(map[string]string) error
	Delete(string) error
}

type LoadUpstreamFunc func(data []byte) (Upstream, error)

type upstreamStore struct {
	conf          StoreConfig
	store         store.Store
	prefixedStore store.Store
	loadUserFunc  LoadUserFunc
	loaders       map[string]LoadUpstreamFunc
}

func NewStore(conf StoreConfig, s store.Store, loadUserFunc LoadUserFunc) Store {
	return &upstreamStore{
		store:         s,
		prefixedStore: store.NewHashedKeyStore(s, prefixUpstream),
		conf:          conf,
		loadUserFunc:  loadUserFunc,
	}
}

func (s upstreamStore) MakeUpstream(key, url, typ string, loadUserFunc LoadUserFunc) Upstream {
	if key == "" {
		key = url
	}
	return &upstream{
		config: Config{
			StoreConfig: s.conf,
			Key:         key,
			URL:         url,
			Type:        typ,
		},
		store:        s.store,
		loadUserFunc: s.loadUserFunc,
	}
}

func (s *upstreamStore) Register(typ string, loaderf LoadUpstreamFunc) {
	s.loaders[typ] = loaderf
}

func (s upstreamStore) Load(key string) (Upstream, error) {
	up, err := s.load(func() ([]byte, error) {
		return s.LoadRaw(key)
	})
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load upstream")
	}
	return up, nil
}

func (s upstreamStore) LoadCurrent() (Upstream, error) {
	up, err := s.load(s.LoadCurrentRaw)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load current upstream")
	}
	return up, nil
}

func (s upstreamStore) load(dataf func() ([]byte, error)) (Upstream, error) {
	data, err := dataf()
	if err != nil {
		return nil, err
	}

	// Unmarshal into any of the types just so that we can get the common data
	up := upstream{}
	err = json.Unmarshal(data, &up)
	if err != nil {
		return nil, err
	}

	upconf := up.Config()
	loaderf := s.loaders[upconf.Type]
	if loaderf == nil {
		return nil,
			errors.Errorf("upstream %q has unsupported type: %q", upconf.Key, upconf.Type)
	}

	newUp, err := loaderf(data)
	newUp.Config().StoreConfig = s.conf
	return newUp, nil
}

func (s upstreamStore) LoadRaw(key string) ([]byte, error) {
	return s.store.Load(key)
}

func (s upstreamStore) Store(up Upstream) (returnErr error) {
	upc := up.Config()
	err := store.StoreJSON(s.store, upc.Key, up)
	if err != nil {
		return errors.WithMessagef(err, "failed to store upstream %q", upc.Key)
	}

	// Update known upstreams
	known, err := s.LoadKnown()
	if err != nil {
		return err
	}
	known[upc.Key] = upc.Type
	err = s.StoreKnown(known)
	if err != nil {
		return err
	}
	return nil
}

func (s upstreamStore) Delete(key string) (returnErr error) {
	// Delete the upstream.
	err := s.store.Delete(key)
	if err != nil {
		return err
	}

	// Update known upstreams
	known, err := s.LoadKnown()
	if err != nil {
		return errors.WithMessage(err, "failed to load known upstreams")
	}
	delete(known, key)
	err = s.StoreKnown(known)
	if err != nil {
		return errors.WithMessage(err, "failed to store known upstreams")
	}

	// Remove the current upstream if it matches the deleted
	up := upstream{}
	err = store.LoadJSON(s.store, keyCurrentUpstream, &up)
	if err != nil {
		return errors.WithMessage(err, "failed to load current upstream")
	}
	if up.Config().Key == key {
		err = s.store.Delete(keyCurrentUpstream)
		if err != nil {
			return errors.WithMessage(err, "failed to delete current upstream")
		}
	}

	return nil
}

func (s upstreamStore) StoreKnown(known map[string]string) error {
	err := store.StoreJSON(s.store, keyKnownUpstreams, known)
	if err != nil {
		return errors.WithMessagef(err,
			"failed to store known upstreams %+v", known)
	}
	return nil
}

func (s upstreamStore) LoadKnown() (map[string]string, error) {
	known := map[string]string{}
	err := store.LoadJSON(s.store, keyKnownUpstreams, &known)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load known upstreams")
	}
	return known, nil
}

func (s upstreamStore) StoreCurrent(up Upstream) (returnErr error) {
	err := store.StoreJSON(s.store, keyCurrentUpstream, up)
	if err != nil {
		return errors.WithMessagef(err, "failed to store current upstream %q", up.Config().Key)
	}
	return nil
}

func (s upstreamStore) LoadCurrentRaw() ([]byte, error) {
	data, err := s.store.Load(keyCurrentUpstream)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load current upstream")
	}
	return data, nil
}
