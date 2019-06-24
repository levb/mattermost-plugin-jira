// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jiraserver

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"

	"github.com/andygrunwald/go-jira"
	"github.com/dghubble/oauth1"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/instance"
	"github.com/mattermost/mattermost-plugin-jira/server/store"
)

const Type = "server"

const keyRSAKey = "rsa_key"

const RouteOAuth1Complete = "/oauth1/complete.html"

type Instance struct {
	instance.BasicInstance

	// The SiteURL may change as we go, so we store the PluginKey when as it was installed
	MattermostKey string

	oauth1Config  *oauth1.Config
	rsaPrivateKey *rsa.PrivateKey
}

var _ instance.Instance = (*Instance)(nil)

func New(jiraURL, mattermostKey string, rsaPrivateKey *rsa.PrivateKey) *Instance {
	return &Instance{
		BasicInstance: instance.BasicInstance{
			InstanceType: Type,
			InstanceKey:  jiraURL,
			InstanceURL:  jiraURL,
		},
		MattermostKey: mattermostKey,
		rsaPrivateKey: rsaPrivateKey,
	}
}

func FromJSON(data []byte, rsaPrivateKey *rsa.PrivateKey) (*Instance, error) {
	inst := Instance{}
	err := json.Unmarshal(data, &inst)
	if err != nil {
		return nil, err
	}

	inst.rsaPrivateKey = rsaPrivateKey
	return &inst, nil
}

func (serverInstance Instance) GetMattermostKey() string {
	return serverInstance.MattermostKey
}

func (serverInstance Instance) GetDisplayDetails() map[string]string {
	return map[string]string{
		"MattermostKey": serverInstance.MattermostKey,
	}
}

func (serverInstance Instance) GetUserConnectURL(otsStore store.OneTimeStore,
	pluginURL, mattermostUserId string) (returnURL string, returnErr error) {

	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessage(returnErr, "failed to get a connect link")
		}
	}()

	oauth1Config, err := serverInstance.GetOAuth1Config(pluginURL)
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

func (serverInstance Instance) GetClient(pluginURL string,
	user *store.User) (returnClient *jira.Client, returnErr error) {

	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessagef(returnErr,
				"failed to get a Jira client for %q", user.DisplayName)
		}
	}()

	if user.Oauth1AccessToken == "" || user.Oauth1AccessSecret == "" {
		return nil, errors.New("No access token, please use /jira connect")
	}

	oauth1Config, err := serverInstance.GetOAuth1Config(pluginURL)
	if err != nil {
		return nil, err
	}

	token := oauth1.NewToken(user.Oauth1AccessToken, user.Oauth1AccessSecret)
	httpClient := oauth1Config.Client(oauth1.NoContext, token)
	jiraClient, err := jira.NewClient(httpClient, serverInstance.GetURL())
	if err != nil {
		return nil, err
	}

	return jiraClient, nil
}

func (serverInstance *Instance) GetOAuth1Config(pluginURL string) (*oauth1.Config, error) {
	return &oauth1.Config{
		ConsumerKey:    serverInstance.MattermostKey,
		ConsumerSecret: "dontcare",
		CallbackURL:    pluginURL + "/" + RouteOAuth1Complete,
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: serverInstance.GetURL() + "/plugins/servlet/oauth/request-token",
			AuthorizeURL:    serverInstance.GetURL() + "/plugins/servlet/oauth/authorize",
			AccessTokenURL:  serverInstance.GetURL() + "/plugins/servlet/oauth/access-token",
		},
		Signer: &oauth1.RSASigner{PrivateKey: serverInstance.rsaPrivateKey},
	}, nil
}

func (serverInstance Instance) PublicKeyString() ([]byte, error) {
	b, err := x509.MarshalPKIXPublicKey(&serverInstance.rsaPrivateKey.PublicKey)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to encode public key")
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: b,
	}), nil
}
