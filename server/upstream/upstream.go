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
)

var ErrWrongUpstreamType = errors.New("wrong upstream type")

type UpstreamConfig struct {
	StoreConfig `json:"-"`
	Key         string
	URL         string
	Type        string
}

type Upstream interface {
	Config() *UpstreamConfig
	StoreUser(User) error
	DeleteUser(User) error
	LoadUser(mattermostUserId string) (User, error)
	LoadMattermostUserId(upstreamUserId string) (string, error)

	DisplayDetails() map[string]string
	GetClient(string, User) (*jira.Client, error)
	GetUserConnectURL(ots kvstore.OneTimeStore, pluginURL string, mattermostUserId string) (string, error)
}

type BasicUpstream struct {
	UpstreamConfig
	kv           kvstore.KVStore
	unmarshaller Unmarshaller
}

func (up *BasicUpstream) Config() *UpstreamConfig {
	return &up.UpstreamConfig
}

func (up BasicUpstream) DisplayDetails() map[string]string {
	return map[string]string{}
}

func (up BasicUpstream) GetClient(string, User) (*jira.Client, error) {
	return nil, errors.New("API not available")
}

func (up BasicUpstream) GetUserConnectURL(ots kvstore.OneTimeStore, pluginURL string, mattermostUserId string) (string, error) {
	return "", nil
}

func NormalizeUpstreamURL(upURL string) (string, error) {
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