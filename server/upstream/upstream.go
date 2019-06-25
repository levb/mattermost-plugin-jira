// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package upstream

import (
	//"crypto/rsa"
	"github.com/pkg/errors"

	"github.com/andygrunwald/go-jira"

	"github.com/mattermost/mattermost-plugin-jira/server/store"
)

var ErrWrongUpstreamType = errors.New("wrong upstream type")

type Config struct {
	StoreConfig

	// Instance-level
	Key string
	URL string
	Type string

	LoadUser LoadUserFunc `json:"none"`
}

type Upstream interface {
	Config() *Config
	UserStore
	
	DisplayDetails() map[string]string
	GetClient(string, User) (*jira.Client, error)
	GetUserConnectURL(ots store.OneTimeStore, pluginURL string, mattermostUserId string) (string, error)
}

type upstream struct {
	config Config
	store store.Store
	loadUserFunc LoadUserFunc
}

func (up upstream) Config() *Config {
	return &up.config
}

func (up upstream) DisplayDetails() map[string]string {
	return map[string]string{}
}

func (up upstream) GetClient(string, User) (*jira.Client, error) {
	return nil, errors.New("API not available")
}

func (up upstream) GetUserConnectURL(ots store.OneTimeStore, pluginURL string, mattermostUserId string) (string, error) {
	return "", nil
}
