// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira_cloud

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	gojira "github.com/andygrunwald/go-jira"
	"github.com/dgrijalva/jwt-go"
	ajwt "github.com/rbriski/atlassian-jwt"
	oauth2_jira "golang.org/x/oauth2/jira"

	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/store"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
)

const Type = "cloud"

type jiraCloudUpstream struct {
	upstream.Upstream

	// Initially a new instance is created with an expiration time. The
	// admin is expected to upload it to the Jira instance, and we will
	// then receive a /installed callback that initializes the instance
	// and makes it permanent. No subsequent /installed will be accepted
	// for the instance.
	Installed bool

	// For cloud instances (atlassian-connect.json install and user auth)
	RawAtlassianSecurityContext string

	// Runtime data, not marshalled to JSON, not saved to the Store
	atlassianSecurityContext *AtlassianSecurityContext
}

const UserLandingPageKey = "user-redirect"

type AtlassianSecurityContext struct {
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

func NewUpstream(store store.Store, installed bool, rawASC string,
	asc *AtlassianSecurityContext, authTokenSecret []byte) upstream.Upstream {

	conf := upstream.Config{
		StoreConfig: upstream.StoreConfig{
			AuthTokenSecret: authTokenSecret,
		},
		Key:  asc.BaseURL,
		URL:  asc.BaseURL,
		Type: Type,
	}

	up := &jiraCloudUpstream{
		Upstream:                    upstream.NewUpstream(conf, store, jira.UnmarshalUser),
		Installed:                   installed,
		RawAtlassianSecurityContext: rawASC,
		atlassianSecurityContext:    asc,
	}

	return up
}

func UnmarshalUpstream(data []byte, config upstream.Config) (upstream.Upstream, error) {
	up := jiraCloudUpstream{}
	*(up.Config()) = config
	err := json.Unmarshal(data, &up)
	if err != nil {
		return nil, err
	}
	up.Config().Key = up.atlassianSecurityContext.BaseURL
	up.Config().URL = up.atlassianSecurityContext.BaseURL
	return &up, nil
}

func (up jiraCloudUpstream) GetDisplayDetails() map[string]string {
	if !up.Installed {
		return map[string]string{
			"Setup": "In progress",
		}
	}

	return map[string]string{
		"Key":            up.atlassianSecurityContext.Key,
		"ClientKey":      up.atlassianSecurityContext.ClientKey,
		"ServerVersion":  up.atlassianSecurityContext.ServerVersion,
		"PluginsVersion": up.atlassianSecurityContext.PluginsVersion,
	}
}

func (up jiraCloudUpstream) GetUserConnectURL(otsStore store.OneTimeStore,
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
	token, err := up.NewAuthToken(mattermostUserId, secret)
	if err != nil {
		return "", err
	}

	v := url.Values{}
	v.Add(ArgMMToken, token)
	return fmt.Sprintf("%v/login?dest-url=%v/plugins/servlet/ac/%s/%s?%v",
		up.Config().URL,
		up.Config().URL,
		up.atlassianSecurityContext.Key,
		UserLandingPageKey,
		v.Encode(),
	), nil
}

func (up jiraCloudUpstream) GetClient(pluginURL string, user upstream.User) (*gojira.Client, error) {

	oauth2Conf := oauth2_jira.Config{
		BaseURL: up.Config().URL,
		Subject: user.UpstreamId(),
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
func (up jiraCloudUpstream) getClientForServer() (*gojira.Client, error) {
	jwtConf := &ajwt.Config{
		Key:          up.atlassianSecurityContext.Key,
		ClientKey:    up.atlassianSecurityContext.ClientKey,
		SharedSecret: up.atlassianSecurityContext.SharedSecret,
		BaseURL:      up.atlassianSecurityContext.BaseURL,
	}

	return gojira.NewClient(jwtConf.Client(), jwtConf.BaseURL)
}

func (up jiraCloudUpstream) JWTFromHTTPRequest(r *http.Request) (
	token *jwt.Token, rawToken string, status int, err error) {

	tokenString := r.FormValue("jwt")
	if tokenString == "" {
		return nil, "", http.StatusBadRequest, errors.New("no jwt found in the HTTP request")
	}

	token, err = jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.Errorf(
				"unsupported signing method: %v", token.Header["alg"])
		}
		// HMAC secret is a []byte
		return []byte(up.atlassianSecurityContext.SharedSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, "", http.StatusUnauthorized, errors.WithMessage(err, "failed to validatte JWT")
	}

	return token, tokenString, http.StatusOK, nil
}
