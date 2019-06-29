// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jiraserver

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"

	gojira "github.com/andygrunwald/go-jira"
	"github.com/dghubble/oauth1"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/store"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
)

const Type = "server"

const RouteOAuth1Complete = "/oauth1/complete.html"

type jiraServerUpstream struct {
	upstream.Upstream
	mattermostKey string
}

type jiraServerUser struct {
	jira.User
	Oauth1AccessToken  string `json:",omitempty"`
	Oauth1AccessSecret string `json:",omitempty"`
}

func NewUpstream(store store.Store, jiraURL, mattermostKey string, rsaPrivateKey *rsa.PrivateKey) upstream.Upstream {
	conf := upstream.Config{
		StoreConfig: upstream.StoreConfig{
			RSAPrivateKey: rsaPrivateKey,
		},
		Key:  jiraURL,
		URL:  jiraURL,
		Type: Type,
	}

	up := &jiraServerUpstream{
		Upstream:      upstream.NewUpstream(conf, store, jira.UnmarshalUser),
		mattermostKey: mattermostKey,
	}

	return up
}

func UnmarshalUpstream(data []byte, config upstream.Config) (upstream.Upstream, error) {
	up := jiraServerUpstream{}
	*(up.Config()) = config
	err := json.Unmarshal(data, &up)
	if err != nil {
		return nil, err
	}
	return &up, nil
}

// func (up jiraServerUpstream) GetMattermostKey() string {
// 	return up.mattermostKey
// }

func (up jiraServerUpstream) GetDisplayDetails() map[string]string {
	return map[string]string{
		"MattermostKey": up.mattermostKey,
	}
}

func (up jiraServerUpstream) GetUserConnectURL(otsStore store.OneTimeStore,
	pluginURL, mattermostUserId string) (returnURL string, returnErr error) {

	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessage(returnErr, "failed to get a connect link")
		}
	}()

	oauth1Config, err := up.GetOAuth1Config(pluginURL)
	if err != nil {
		return "", err
	}

	token, secret, err := oauth1Config.RequestToken()
	if err != nil {
		return "", err
	}

	err = otsStore.StoreOauth1aTemporaryCredentials(mattermostUserId,
		&store.OAuth1aTemporaryCredentials{Token: token, Secret: secret})
	if err != nil {
		return "", err
	}

	authURL, err := oauth1Config.AuthorizationURL(token)
	if err != nil {
		return "", err
	}

	return authURL.String(), nil
}

func (up jiraServerUpstream) GetClient(pluginURL string,
	u upstream.User) (returnClient *gojira.Client, returnErr error) {
	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessagef(returnErr,
				"failed to get a Jira client for %q", u.UpstreamDisplayName())
		}
	}()

	user, ok := u.(*jiraServerUser)
	if !ok {
		return nil, errors.Errorf("expected Jira Server user, got %T", u)
	}
	if user.Oauth1AccessToken == "" || user.Oauth1AccessSecret == "" {
		return nil, errors.New("no access token, please use /jira connect")
	}

	oauth1Config, err := up.GetOAuth1Config(pluginURL)
	if err != nil {
		return nil, err
	}

	token := oauth1.NewToken(user.Oauth1AccessToken, user.Oauth1AccessSecret)
	httpClient := oauth1Config.Client(oauth1.NoContext, token)
	jiraClient, err := gojira.NewClient(httpClient, up.Config().URL)
	if err != nil {
		return nil, err
	}

	return jiraClient, nil
}

func (up jiraServerUpstream) GetOAuth1Config(pluginURL string) (*oauth1.Config, error) {
	return &oauth1.Config{
		ConsumerKey:    up.mattermostKey,
		ConsumerSecret: "dontcare",
		CallbackURL:    pluginURL + "/" + RouteOAuth1Complete,
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: up.Config().URL + "/plugins/servlet/oauth/request-token",
			AuthorizeURL:    up.Config().URL + "/plugins/servlet/oauth/authorize",
			AccessTokenURL:  up.Config().URL + "/plugins/servlet/oauth/access-token",
		},
		Signer: &oauth1.RSASigner{PrivateKey: up.Config().RSAPrivateKey},
	}, nil
}

func (up jiraServerUpstream) PublicKeyString() ([]byte, error) {
	b, err := x509.MarshalPKIXPublicKey(&up.Config().RSAPrivateKey.PublicKey)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to encode public key")
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: b,
	}), nil
}
