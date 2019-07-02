// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira_server

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"

	gojira "github.com/andygrunwald/go-jira"
	"github.com/dghubble/oauth1"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
)

const Type = "server"

type Upstream struct {
	upstream.BasicUpstream
	mattermostKey string
}

func newUpstream(upstore upstream.Store, jiraURL, mattermostKey string) upstream.Upstream {
	conf := upstream.UpstreamConfig{
		StoreConfig: *(upstore.Config()),
		Key:         jiraURL,
		URL:         jiraURL,
		Type:        Type,
	}

	up := &Upstream{
		BasicUpstream: upstore.MakeBasicUpstream(conf),
		mattermostKey: mattermostKey,
	}

	return up
}

func (up Upstream) GetDisplayDetails() map[string]string {
	return map[string]string{
		"MattermostKey": up.mattermostKey,
	}
}

func (up Upstream) GetUserConnectURL(otsStore kvstore.OneTimeStore,
	pluginURL, mattermostUserId string) (returnURL string, returnErr error) {

	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessage(returnErr, "failed to get a connect link")
		}
	}()

	oauth1Config, err := up.getOAuth1Config(pluginURL)
	if err != nil {
		return "", err
	}

	token, secret, err := oauth1Config.RequestToken()
	if err != nil {
		return "", err
	}

	err = otsStore.StoreOauth1aTemporaryCredentials(mattermostUserId,
		&kvstore.OAuth1aTemporaryCredentials{Token: token, Secret: secret})
	if err != nil {
		return "", err
	}

	authURL, err := oauth1Config.AuthorizationURL(token)
	if err != nil {
		return "", err
	}

	return authURL.String(), nil
}

func (up Upstream) GetClient(pluginURL string,
	u upstream.User) (returnClient *gojira.Client, returnErr error) {
	defer func() {
		if returnErr != nil {
			returnErr = errors.WithMessagef(returnErr,
				"failed to get a Jira client for %q", u.UpstreamDisplayName())
		}
	}()

	user, ok := u.(*user)
	if !ok {
		return nil, errors.Errorf("expected Jira Server user, got %T", u)
	}
	if user.Oauth1AccessToken == "" || user.Oauth1AccessSecret == "" {
		return nil, errors.New("no access token, please use /jira connect")
	}

	oauth1Config, err := up.getOAuth1Config(pluginURL)
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

func (up Upstream) getOAuth1Config(pluginURL string) (*oauth1.Config, error) {
	return &oauth1.Config{
		ConsumerKey:    up.mattermostKey,
		ConsumerSecret: "dontcare",
		CallbackURL:    pluginURL + "/" + routeOAuth1Complete,
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: up.Config().URL + "/plugins/servlet/oauth/request-token",
			AuthorizeURL:    up.Config().URL + "/plugins/servlet/oauth/authorize",
			AccessTokenURL:  up.Config().URL + "/plugins/servlet/oauth/access-token",
		},
		Signer: &oauth1.RSASigner{PrivateKey: up.Config().RSAPrivateKey},
	}, nil
}

func (up Upstream) PublicKeyString() ([]byte, error) {
	b, err := x509.MarshalPKIXPublicKey(&up.Config().RSAPrivateKey.PublicKey)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to encode public key")
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: b,
	}), nil
}

func RequireUpstream(a action.Action) error {
	err := proxy.RequireUpstream(a)
	if err != nil {
		return err
	}
	up, ok := a.Context().Upstream.(*Upstream)
	if !ok {
		return a.RespondError(http.StatusInternalServerError, errors.Errorf(
			"Jira Server upstream required, got %T", a.Context().Upstream))
	}
	a.Debugf("action: verified Jira Server instance %+v", up)
	return nil
}

type unmarshaller struct{}

var Unmarshaller upstream.Unmarshaller = unmarshaller{}

func (_ unmarshaller) UnmarshalUser(data []byte, defaultId string) (upstream.User, error) {
	u := user{}
	err := json.Unmarshal(data, &u)
	if err != nil {
		return nil, err
	}
	if u.BasicUser.MUserId == "" {
		u.BasicUser.MUserId = defaultId
	}
	if u.BasicUser.UUserId == "" {
		u.BasicUser.UUserId = u.JiraUser.Name
	}
	return &u, nil
}

func (_ unmarshaller) UnmarshalUpstream(data []byte, basicUp upstream.BasicUpstream) (upstream.Upstream, error) {
	up := Upstream{}
	err := json.Unmarshal(data, &up)
	if err != nil {
		return nil, err
	}
	up.BasicUpstream = basicUp
	return &up, nil
}
