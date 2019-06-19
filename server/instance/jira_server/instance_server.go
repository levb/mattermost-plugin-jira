// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira_server

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/andygrunwald/go-jira"
	"github.com/dghubble/oauth1"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/instance"
	"github.com/mattermost/mattermost-plugin-jira/server/store"
)

const BackendType = "server"

const keyRSAKey = "rsa_key"

const RouteOAuth1Complete = "/oauth1/complete.html"

type JiraServerInstance struct {
	instance.BasicInstance

	// The JSON name is v2.0 compatible
	BackendURL string `json:"JIRAServerURL,omitempty"`

	// The SiteURL may change as we go, so we store the PluginKey when as it was installed
	MattermostKey string

	oauth1Config  *oauth1.Config
	rsaPrivateKey *rsa.PrivateKey
}

var _ instance.Instance = (*JiraServerInstance)(nil)

func NewServerInstance(jiraURL, mattermostKey string, rsaPrivateKey *rsa.PrivateKey) *JiraServerInstance {
	return &JiraServerInstance{
		BasicInstance: instance.BasicInstance{
			InstanceType: BackendType,
			InstanceKey:  jiraURL,
		},
		MattermostKey: mattermostKey,
		BackendURL:    jiraURL,
		rsaPrivateKey: rsaPrivateKey,
	}
}

// ensureRSAKey () (*rsaPrivateKey, error) {
// 	data, err := ensuredStore.Ensure(keyRSAKey, makeRSAKey)
// 	if err != nil {
// 		return nil, errors.WithMessage(err, "failed to create an OAuth1 config")
// 	}

// }
// func makeRSAKey() ([]byte, error) {
// 	newRSAKey, err := rsa.GenerateKey(rand.Reader, 1024)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return json.Marshal(newRSAKey)
// }

func (serverInstance JiraServerInstance) GetURL() string {
	return serverInstance.BackendURL
}

func (serverInstance JiraServerInstance) GetMattermostKey() string {
	return serverInstance.MattermostKey
}

func (serverInstance JiraServerInstance) GetDisplayDetails() map[string]string {
	return map[string]string{
		"MattermostKey": serverInstance.MattermostKey,
	}
}

func (serverInstance JiraServerInstance) GetUserConnectURL(otsStore store.OneTimeStore,
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

func (serverInstance JiraServerInstance) GetClient(pluginURL string,
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

func (serverInstance *JiraServerInstance) GetOAuth1Config(pluginURL string) (*oauth1.Config, error) {
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

func (serverInstance JiraServerInstance) PublicKeyString() ([]byte, error) {
	b, err := x509.MarshalPKIXPublicKey(&serverInstance.rsaPrivateKey.PublicKey)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to encode public key")
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: b,
	}), nil
}
