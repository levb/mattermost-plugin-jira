// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package proxy

import (
	"crypto/rsa"
	"net/http"
	"text/template"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
)

type Config struct {
	API           plugin.API
	KVStore       kvstore.KVStore
	Templates     map[string]*template.Template
	Unmarshallers map[string]upstream.Unmarshaller
}

type Context struct {
	Config

	RSAPrivateKey   *rsa.PrivateKey
	AuthTokenSecret []byte
}

type Proxy interface {
	Context() *Context

	upstream.UpstreamStore

	RunHTTP(*plugin.Context, http.ResponseWriter, *http.Request) error
	RunCommand(*plugin.Context, *model.CommandArgs) (*model.CommandResponse, error)
}

type proxy struct {
	context        Context
	actionConfig   action.Config
	upstreamConfig upstream.Config
	upstreamKV     kvstore.KVStore
}

func MakeProxy(config Config, actionConfig action.Config) (Proxy, error) {
	kv := config.KVStore
	rsaPrivateKey, err := EnsureRSAPrivateKey(kv)
	if err != nil {
		return nil, err
	}
	authTokenSecret, err := EnsureAuthTokenSecret(kv)
	if err != nil {
		return nil, err
	}

	return &proxy{
		context: Context{
			Config:          config,
			RSAPrivateKey:   rsaPrivateKey,
			AuthTokenSecret: authTokenSecret,
		},
		actionConfig: actionConfig,
		upstreamKV:   kvstore.NewHashedKeyStore(kv, kvstore.KeyPrefixUpstream),
	}, nil
}

func (p proxy) Context() *Context {
	return &p.context
}

func (p proxy) RunHTTP(*plugin.Context, http.ResponseWriter, *http.Request) error {
	// TODO
	return nil
}

func (p proxy) RunCommand(*plugin.Context, *model.CommandArgs) (*model.CommandResponse, error) {
	// TODO
	return &model.CommandResponse{}, nil
}
