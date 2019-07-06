// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package upstream

import (
	"net/url"
	"path"
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-server/plugin"
)

var ErrWrongUpstreamType = errors.New("wrong upstream type")

type Upstream interface {
	UserStore

	Key() string
	Type() string

	DisplayFields() map[string]interface{}
	GetClient(string, User) (*jira.Client, error)
	GetUserConnectURL(ots kvstore.KVStore, pluginURL string, mattermostUserId string) (string, error)
}

type Unmarshaller interface {
	UnmarshalUpstream([]byte, Basic) (Upstream, error)
}

type UpstreamStore interface {
	MakeBasicUpstream(key, typ string) Basic

	LoadUpstream(string) (Upstream, error)
	LoadCurrentUpstream() (Upstream, error)
	LoadCurrentUpstreamRaw() ([]byte, error)
	LoadKnownUpstreams() (map[string]string, error)
	LoadUpstreamRaw(string) ([]byte, error)
	StoreUpstream(Upstream) error
	StoreCurrentUpstream(Upstream) error
	StoreKnownUpstreams(map[string]string) error
	DeleteUpstream(string) error

	DeleteUpstreamNotify(string) error
	StoreCurrentUpstreamNotify(Upstream) error
}

type Basic struct {
	UpstreamKey  string `json:"Key"`
	UpstreamType string `json:"Type"`
	kv           kvstore.KVStore
	api          plugin.API
}

var _ Upstream = (*Basic)(nil)

func NewBasic(key, typ string, api plugin.API, kv kvstore.KVStore) Basic {
	return Basic{
		UpstreamKey:  key,
		UpstreamType: typ,
		api:          api,
		kv:           kv,
	}
}

func (up Basic) Key() string {
	return up.UpstreamKey
}

func (up Basic) Type() string {
	return up.UpstreamType
}

func (up Basic) DisplayFields() map[string]interface{} {
	return map[string]interface{}{}
}

func (up Basic) GetClient(string, User) (*jira.Client, error) {
	return nil, errors.New("API not available")
}

func (up Basic) GetUserConnectURL(ots kvstore.KVStore, pluginURL string, mattermostUserId string) (string, error) {
	return "", nil
}

func NormalizeURL(upURL string) (string, error) {
	u, err := url.Parse(upURL)
	if err != nil {
		return "", err
	}
	if u.Host == "" {
		ss := strings.Split(u.Path, "/")
		if len(ss) > 0 && ss[0] != "" {
			u.Host = ss[0]
			u.Path = path.Join(ss[1:]...)
		}
		u, err = url.Parse(u.String())
		if err != nil {
			return "", err
		}
	}
	if u.Host == "" {
		return "", errors.Errorf("Invalid URL, no hostname: %q", upURL)
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	return strings.TrimSuffix(u.String(), "/"), nil
}
