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

type serverUpstream struct {
	upstream.Basic
}

func newUpstream(upstore upstream.UpstreamStore, jiraURL string) upstream.Upstream {
	return &serverUpstream{
		Basic: upstore.MakeBasicUpstream(jiraURL, Type),
	}
}

func (up serverUpstream) LoadUser(mattermostUserId string) (upstream.User, error) {
	data, err := up.LoadUserRaw(mattermostUserId)
	if err != nil {
		return nil, err
	}

	u := user{}
	err = json.Unmarshal(data, &u)
	if err != nil {
		return nil, err
	}
	if u.BasicUser.MUserId == "" {
		u.BasicUser.MUserId = mattermostUserId
	} else if u.BasicUser.MUserId != mattermostUserId {
		return nil, errors.Errorf(
			"stored user id %q did not match the current user id: %q", u.BasicUser.MUserId, mattermostUserId)
	}

	if u.BasicUser.UUserId == "" {
		u.BasicUser.UUserId = u.JiraUser.Name
	}
	return &u, nil
}

func (up serverUpstream) GetDisplayDetails() map[string]string {
	return map[string]string{
		"pluginKey": up.Context().PluginKey,
	}
}

func (up serverUpstream) GetUserConnectURL(ots kvstore.OneTimeStore,
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

	err = storeTempCredentials(ots, mattermostUserId,
		&oauth1aTempCredentials{Token: token, Secret: secret})
	if err != nil {
		return "", err
	}

	authURL, err := oauth1Config.AuthorizationURL(token)
	if err != nil {
		return "", err
	}

	return authURL.String(), nil
}

func (up serverUpstream) GetClient(pluginURL string,
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
	jiraClient, err := gojira.NewClient(httpClient, up.URL())
	if err != nil {
		return nil, err
	}

	return jiraClient, nil
}

func (up serverUpstream) getOAuth1Config(pluginURL string) (*oauth1.Config, error) {
	return &oauth1.Config{
		ConsumerKey:    up.Context().PluginKey,
		ConsumerSecret: "dontcare",
		CallbackURL:    pluginURL + "/" + routeOAuth1Complete,
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: up.URL() + "/plugins/servlet/oauth/request-token",
			AuthorizeURL:    up.URL() + "/plugins/servlet/oauth/authorize",
			AccessTokenURL:  up.URL() + "/plugins/servlet/oauth/access-token",
		},
		Signer: &oauth1.RSASigner{PrivateKey: up.Context().ProxyRSAPrivateKey},
	}, nil
}

func publicKeyString(up upstream.Upstream) ([]byte, error) {
	b, err := x509.MarshalPKIXPublicKey(&up.Context().ProxyRSAPrivateKey.PublicKey)
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
	up, ok := a.Context().Upstream.(*serverUpstream)
	if !ok {
		return a.RespondError(http.StatusInternalServerError, errors.Errorf(
			"Jira Server upstream required, got %T", a.Context().Upstream))
	}
	a.Debugf("action: verified Jira Server instance %+v", up)
	return nil
}

type unmarshaller struct{}

var Unmarshaller upstream.Unmarshaller = unmarshaller{}

func (_ unmarshaller) UnmarshalUpstream(data []byte, basicUp upstream.Basic) (upstream.Upstream, error) {
	up := serverUpstream{}
	err := json.Unmarshal(data, &up)
	if err != nil {
		return nil, err
	}
	up.Basic = basicUp
	return &up, nil
}
