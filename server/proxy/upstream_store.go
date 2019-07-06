// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package proxy

import (
	"encoding/json"

	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/pkg/errors"
)

func (p proxy) MakeBasicUpstream(key, typ string) upstream.Basic {
	return upstream.NewBasic(key, typ, p.context.API, p.context.KVStore)
}

func (p proxy) LoadUpstream(key string) (upstream.Upstream, error) {
	up, err := p.loadUpstream(func() ([]byte, error) {
		return p.upstreamKV.Load(key)
	})
	if err != nil {
		return nil, err
	}
	return up, nil
}

func (p proxy) LoadCurrentUpstream() (upstream.Upstream, error) {
	up, err := p.loadUpstream(p.LoadCurrentUpstreamRaw)
	if err != nil {
		return nil, err
	}
	return up, nil
}

func (p proxy) loadUpstream(dataf func() ([]byte, error)) (upstream.Upstream, error) {
	data, err := dataf()
	if err != nil {
		return nil, err
	}

	// Unmarshal into any of the types just so that we can get the common data
	// Inherit the environment from the proxy, JSON will not overwrite it
	basic := upstream.NewBasic("", "", p.context.API, p.context.KVStore)
	err = json.Unmarshal(data, &basic)
	if err != nil {
		return nil, err
	}

	unmarshal := p.context.Unmarshallers[basic.Type()]
	if unmarshal == nil {
		return nil, errors.Errorf("upstream %q has unsupported type: %q", basic.Key(), basic.Type())
	}

	return unmarshal.UnmarshalUpstream(data, basic)
}

func (p proxy) LoadUpstreamRaw(key string) ([]byte, error) {
	return p.upstreamKV.Load(key)
}

func (p proxy) StoreUpstream(up upstream.Upstream) error {
	err := kvstore.StoreJSON(p.upstreamKV, up.Key(), up)
	if err != nil {
		return err
	}

	// Update known upstreams,
	known, err := p.LoadKnownUpstreams()
	if err != nil && err != kvstore.ErrNotFound {
		return err
	}
	if known == nil {
		known = map[string]string{}
	}
	known[up.Key()] = up.Type()
	err = p.StoreKnownUpstreams(known)
	if err != nil {
		return err
	}
	return nil
}

func (p proxy) DeleteUpstream(key string) (returnErr error) {
	// Delete the upstream.
	err := p.upstreamKV.Delete(key)
	if err != nil {
		return err
	}
	// Update known upstreams
	known, err := p.LoadKnownUpstreams()
	if err != nil {
		return err
	}
	delete(known, key)
	err = p.StoreKnownUpstreams(known)
	if err != nil {
		return err
	}

	// Remove the current upstream if it matches the deleted
	up, err := p.LoadCurrentUpstream()
	switch err {
	case nil:
		if up.Key() == key {
			err = p.context.KVStore.Delete(kvstore.KeyCurrentUpstream)
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

func (p proxy) StoreKnownUpstreams(known map[string]string) error {
	err := kvstore.StoreJSON(p.context.KVStore, kvstore.KeyKnownUpstreams, known)
	if err != nil {
		return err
	}
	return nil
}

func (p proxy) LoadKnownUpstreams() (map[string]string, error) {
	known := map[string]string{}
	err := kvstore.LoadJSON(p.context.KVStore, kvstore.KeyKnownUpstreams, &known)
	if err != nil {
		return nil, err
	}
	return known, nil
}

func (p proxy) StoreCurrentUpstream(up upstream.Upstream) error {
	return kvstore.StoreJSON(p.context.KVStore, kvstore.KeyCurrentUpstream, up)
}

func (p proxy) LoadCurrentUpstreamRaw() ([]byte, error) {
	return p.context.KVStore.Load(kvstore.KeyCurrentUpstream)
}
