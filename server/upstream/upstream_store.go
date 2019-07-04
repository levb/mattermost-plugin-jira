// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package upstream

import (
	"crypto/rsa"
	"encoding/json"

	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

type StoreConfig struct {
	RSAPrivateKey   *rsa.PrivateKey `json:"-"`
	AuthTokenSecret []byte          `json:"-"`
	PluginKey       string          `json:"-"`
}

type Store interface {
	Config() *StoreConfig
	MakeBasicUpstream(conf UpstreamConfig) BasicUpstream
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

type Unmarshaller interface {
	UnmarshalUpstream([]byte, BasicUpstream) (Upstream, error)
	UnmarshalUser([]byte, string) (User, error)
}

type upstreamStore struct {
	config        StoreConfig
	api           plugin.API
	kv            kvstore.KVStore
	upstreamKV    kvstore.KVStore
	unmarshallers map[string]Unmarshaller
}

func NewStore(api plugin.API, conf StoreConfig, kv kvstore.KVStore, unmarshallers map[string]Unmarshaller) Store {
	return &upstreamStore{
		api:           api,
		kv:            kv,
		upstreamKV:    kvstore.NewHashedKeyStore(kv, kvstore.KeyPrefixUpstream),
		config:        conf,
		unmarshallers: unmarshallers,
	}
}

func (s upstreamStore) MakeBasicUpstream(conf UpstreamConfig) BasicUpstream {
	if conf.Key == "" {
		conf.Key = conf.URL
	}
	up := BasicUpstream{
		UpstreamConfig: conf,
		kv:             s.kv,
		unmarshaller:   s.unmarshallers[conf.Type],
	}
	up.UpstreamConfig.StoreConfig = s.config
	return up
}

func (s upstreamStore) Load(key string) (Upstream, error) {
	up, err := s.load(func() ([]byte, error) {
		return s.upstreamKV.Load(key)
	})
	if err != nil {
		return nil, err
	}
	return up, nil
}

func (s upstreamStore) LoadCurrent() (Upstream, error) {
	up, err := s.load(s.LoadCurrentRaw)
	if err != nil {
		return nil, err
	}
	return up, nil
}

func (s upstreamStore) load(dataf func() ([]byte, error)) (Upstream, error) {
	data, err := dataf()
	if err != nil {
		return nil, err
	}

	// Unmarshal into any of the types just so that we can get the common data
	up := s.MakeBasicUpstream(UpstreamConfig{})
	err = json.Unmarshal(data, &up)
	if err != nil {
		return nil, err
	}
	up.kv = s.kv

	upconf := up.Config()
	unmarshal := s.unmarshallers[upconf.Type]
	if unmarshal == nil {
		return nil,
			errors.Errorf("upstream %q has unsupported type: %q", upconf.Key, upconf.Type)
	}
	up.unmarshaller = unmarshal

	newUp, err := unmarshal.UnmarshalUpstream(data, up)
	return newUp, nil
}

func (s upstreamStore) LoadRaw(key string) ([]byte, error) {
	return s.upstreamKV.Load(key)
}

func (s upstreamStore) Store(up Upstream) (returnErr error) {
	upconf := up.Config()
	err := kvstore.StoreJSON(s.upstreamKV, upconf.Key, up)
	if err != nil {
		return err
	}

	// Update known upstreams,
	known, err := s.LoadKnown()
	if err != nil && err != kvstore.ErrNotFound {
		return err
	}
	if known == nil {
		known = map[string]string{}
	}
	known[upconf.Key] = upconf.Type
	err = s.StoreKnown(known)
	if err != nil {
		return err
	}
	return nil
}

func (s upstreamStore) Delete(key string) (returnErr error) {
	// Delete the upstream.
	err := s.upstreamKV.Delete(key)
	if err != nil {
		return err
	}
	// Update known upstreams
	known, err := s.LoadKnown()
	if err != nil {
		return err
	}
	delete(known, key)
	err = s.StoreKnown(known)
	if err != nil {
		return err
	}

	// Remove the current upstream if it matches the deleted
	up, err := s.LoadCurrent()
	switch err {
	case nil:
		if up.Config().Key == key {
			err = s.kv.Delete(kvstore.KeyCurrentUpstream)
			if err != nil {
				return err
			}
		}

	case kvstore.ErrNotFound:

	default:
		return err
	}

	return nil
}

func (s upstreamStore) StoreKnown(known map[string]string) error {
	err := kvstore.StoreJSON(s.kv, kvstore.KeyKnownUpstreams, known)
	if err != nil {
		return err
	}
	return nil
}

func (s upstreamStore) LoadKnown() (map[string]string, error) {
	known := map[string]string{}
	err := kvstore.LoadJSON(s.kv, kvstore.KeyKnownUpstreams, &known)
	if err != nil {
		return nil, err
	}
	return known, nil
}

func (s upstreamStore) StoreCurrent(up Upstream) error {
	return kvstore.StoreJSON(s.kv, kvstore.KeyCurrentUpstream, up)
}

func (s upstreamStore) LoadCurrentRaw() ([]byte, error) {
	return s.kv.Load(kvstore.KeyCurrentUpstream)
}

func (s *upstreamStore) Config() *StoreConfig {
	return &s.config
}
