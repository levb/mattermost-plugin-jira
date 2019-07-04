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
	ajwt "github.com/rbriski/atlassian-jwt"
	oauth2_jira "golang.org/x/oauth2/jira"

	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
)

const Type = "cloud"

type Upstream struct {
	upstream.BasicUpstream

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

func newUpstream(upStore upstream.Store, rawASC string,
	asc *atlassianSecurityContext) upstream.Upstream {

	conf := upstream.UpstreamConfig{
		StoreConfig: *(upStore.Config()),
		Key:         asc.BaseURL,
		URL:         asc.BaseURL,
		Type:        Type,
	}

	return &Upstream{
		BasicUpstream:               upStore.MakeBasicUpstream(conf),
		RawAtlassianSecurityContext: rawASC,
		atlassianSecurityContext:    asc,
	}
}

func (up Upstream) GetDisplayDetails() map[string]string {
	return map[string]string{
		"Key":            up.atlassianSecurityContext.Key,
		"ClientKey":      up.atlassianSecurityContext.ClientKey,
		"ServerVersion":  up.atlassianSecurityContext.ServerVersion,
		"PluginsVersion": up.atlassianSecurityContext.PluginsVersion,
	}
}

func (up Upstream) GetUserConnectURL(otsStore kvstore.OneTimeStore,
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
		up.Config().URL,
		up.Config().URL,
		up.atlassianSecurityContext.Key,
		userLandingPageKey,
		v.Encode(),
	), nil
}

func (up Upstream) GetClient(pluginURL string, user upstream.User) (*gojira.Client, error) {

	oauth2Conf := oauth2_jira.Config{
		BaseURL: up.Config().URL,
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
func (up Upstream) getClientForServer() (*gojira.Client, error) {
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

func (_ unmarshaller) UnmarshalUpstream(data []byte, basicUp upstream.BasicUpstream) (upstream.Upstream, error) {
	up := Upstream{}
	err := json.Unmarshal(data, &up)
	if err != nil {
		return nil, err
	}
	up.BasicUpstream = basicUp

	asc := atlassianSecurityContext{}
	err = json.Unmarshal([]byte(up.RawAtlassianSecurityContext), &asc)
	up.atlassianSecurityContext = &asc

	up.Config().Key = up.atlassianSecurityContext.BaseURL
	up.Config().URL = up.atlassianSecurityContext.BaseURL
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
