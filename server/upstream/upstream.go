// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package upstream

import (
	"net/url"
	"path"
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/store"
)

var ErrWrongUpstreamType = errors.New("wrong upstream type")

type Config struct {
	StoreConfig

	// Instance-level
	Key  string
	URL  string
	Type string
}

type Upstream interface {
	Config() *Config
	UserStore

	DisplayDetails() map[string]string
	GetClient(string, User) (*jira.Client, error)
	GetUserConnectURL(ots store.OneTimeStore, pluginURL string, mattermostUserId string) (string, error)
}

type upstream struct {
	config       Config
	store        store.Store
	loadUserFunc LoadUserFunc
}

func NewUpstream(conf Config, store store.Store, loadUserFunc LoadUserFunc) Upstream {
	return &upstream{
		config:       conf,
		store:        store,
		loadUserFunc: loadUserFunc,
	}
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
