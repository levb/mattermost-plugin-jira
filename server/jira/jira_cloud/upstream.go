// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira_cloud

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/url"

	gojira "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"
	ajwt "github.com/rbriski/atlassian-jwt"
	oauth2_jira "golang.org/x/oauth2/jira"

	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
)

const Type = "cloud"

type cloudUpstream struct {
	upstream.Basic

	// For cloud instances (atlassian-connect.json install and user auth)
	RawAtlassianSecurityContext string

	// Runtime data, not marshalled to JSON, not saved to the Store
	atlassianSecurityContext *atlassianSecurityContext
}

const userLandingPageKey = "user-redirect"

type atlassianSecurityContext struct {
	Key            string `json:"key"`
	ClientKey      string `json:"clientKey"`
	SharedSecret   string `json:"sharedSecret"`
	ServerVersion  string `json:"serverVersion"`
	PluginsVersion string `json:"pluginsVersion"`
	BaseURL        string `json:"baseUrl"`
	ProductType    string `json:"productType"`
	Description    string `json:"description"`
	EventType      string `json:"eventType"`
	OAuthClientId  string `json:"oauthClientId"`
}

func newUpstream(upstore upstream.UpstreamStore, rawASC string,
	asc *atlassianSecurityContext) upstream.Upstream {

	return &cloudUpstream{
		Basic:                       upstore.MakeBasicUpstream(asc.BaseURL, Type),
		RawAtlassianSecurityContext: rawASC,
		atlassianSecurityContext:    asc,
	}
}

func (up cloudUpstream) URL() string {
	return up.atlassianSecurityContext.BaseURL
}

func (up cloudUpstream) LoadUser(mattermostUserId string) (upstream.User, error) {
	data, err := up.LoadUserRaw(mattermostUserId)
	if err != nil {
		return nil, err
	}

	u := jira.User{}
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

func (up cloudUpstream) GetDisplayDetails() map[string]string {
	return map[string]string{
		"Key":            up.atlassianSecurityContext.Key,
		"ClientKey":      up.atlassianSecurityContext.ClientKey,
		"ServerVersion":  up.atlassianSecurityContext.ServerVersion,
		"PluginsVersion": up.atlassianSecurityContext.PluginsVersion,
	}
}

func (up cloudUpstream) GetUserConnectURL(otsStore kvstore.OneTimeStore,
	pluginURL, mattermostUserId string) (string, error) {

	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	secret := fmt.Sprintf("%x", randomBytes)
	err = otsStore.Store(mattermostUserId, []byte(secret))
	if err != nil {
		return "", err
	}
	token, err := up.newAuthToken(mattermostUserId, secret)
	if err != nil {
		return "", err
	}

	v := url.Values{}
	v.Add(argMMToken, token)
	return fmt.Sprintf("%v/login?dest-url=%v/plugins/servlet/ac/%s/%s?%v",
		up.URL(),
		up.URL(),
		up.atlassianSecurityContext.Key,
		userLandingPageKey,
		v.Encode(),
	), nil
}

func (up cloudUpstream) GetClient(pluginURL string, user upstream.User) (*gojira.Client, error) {

	oauth2Conf := oauth2_jira.Config{
		BaseURL: up.atlassianSecurityContext.BaseURL,
		Subject: user.UpstreamUserId(),
	}

	oauth2Conf.Config.ClientID = up.atlassianSecurityContext.OAuthClientId
	oauth2Conf.Config.ClientSecret = up.atlassianSecurityContext.SharedSecret
	oauth2Conf.Config.Endpoint.AuthURL = "https://auth.atlassian.io"
	oauth2Conf.Config.Endpoint.TokenURL = "https://auth.atlassian.io/oauth2/token"

	httpClient := oauth2Conf.Client(context.Background())

	jiraClient, err := gojira.NewClient(httpClient, oauth2Conf.BaseURL)
	return jiraClient, err
}

// Creates a "bot" client with a JWT
func (up cloudUpstream) getClientForServer() (*gojira.Client, error) {
	jwtConf := &ajwt.Config{
		Key:          up.atlassianSecurityContext.Key,
		ClientKey:    up.atlassianSecurityContext.ClientKey,
		SharedSecret: up.atlassianSecurityContext.SharedSecret,
		BaseURL:      up.atlassianSecurityContext.BaseURL,
	}

	return gojira.NewClient(jwtConf.Client(), jwtConf.BaseURL)
}

type unmarshaller struct{}

// Unmarshaller unmarshals Jira Cloud entities from JSON
var Unmarshaller upstream.Unmarshaller = unmarshaller{}

func (_ unmarshaller) UnmarshalUpstream(data []byte, basicUp upstream.Basic) (upstream.Upstream, error) {
	up := cloudUpstream{}
	err := json.Unmarshal(data, &up)
	if err != nil {
		return nil, err
	}
	up.Basic = basicUp

	asc := atlassianSecurityContext{}
	err = json.Unmarshal([]byte(up.RawAtlassianSecurityContext), &asc)
	up.atlassianSecurityContext = &asc

	if up.Basic.UpstreamType == "" {
		up.Basic.UpstreamType = Type
	} else if up.Basic.UpstreamType != Type {
		return nil, errors.Errorf(
			"attempted to load upstream type %q as a %q", up.Basic.UpstreamType, Type)
	}
	return &up, nil
}

func (_ unmarshaller) UnmarshalUser(data []byte, mattermostUserId string) (upstream.User, error) {
	return jira.UnmarshalUser(data, mattermostUserId)
}

func storeUnconfirmedUpstream(ots kvstore.OneTimeStore, jiraURL string) error {
	return kvstore.NewHashedKeyStore(ots, kvstore.KeyPrefixUnconfirmedUpstream).Store(
		jiraURL, []byte("doesn't matter"))
}

func loadUnconfirmedUpstream(ots kvstore.OneTimeStore, jiraURL string) error {
	_, err := kvstore.NewHashedKeyStore(ots, kvstore.KeyPrefixUnconfirmedUpstream).Load(jiraURL)
	return err
}
