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

	"github.com/andygrunwald/go-jira"
	"github.com/dgrijalva/jwt-go"
	ajwt "github.com/rbriski/atlassian-jwt"
	oauth2_jira "golang.org/x/oauth2/jira"

	"github.com/mattermost/mattermost-plugin-jira/server/instance"
	"github.com/mattermost/mattermost-plugin-jira/server/store"
)

const Type = "cloud"

type Instance struct {
	instance.BasicInstance

	// Initially a new instance is created with an expiration time. The
	// admin is expected to upload it to the Jira instance, and we will
	// then receive a /installed callback that initializes the instance
	// and makes it permanent. No subsequent /installed will be accepted
	// for the instance.
	Installed       bool
	authTokenSecret []byte `json:"none"`

	// For cloud instances (atlassian-connect.json install and user auth)
	*AtlassianSecurityContext   `json:"none"`
	RawAtlassianSecurityContext string
}

var _ instance.Instance = (*Instance)(nil)

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

func New(key string, installed bool, rawASC string,
	asc *AtlassianSecurityContext, authTokenSecret []byte) *Instance {

	return &Instance{
		BasicInstance: instance.BasicInstance{
			InstanceType: Type,
			InstanceKey:  key,
			InstanceURL:  asc.BaseURL,
		},
		Installed:                   installed,
		AtlassianSecurityContext:    asc,
		RawAtlassianSecurityContext: rawASC,
		authTokenSecret:             authTokenSecret,
	}
}

func FromJSON(data, authTokenSecret []byte) (*Instance, error) {
	inst := Instance{}
	err := json.Unmarshal(data, &inst)
	if err != nil {
		return nil, err
	}
	inst.BasicInstance.InstanceURL = inst.AtlassianSecurityContext.BaseURL
	inst.authTokenSecret = authTokenSecret
	return &inst, nil
}

func (jci Instance) GetMattermostKey() string {
	return jci.AtlassianSecurityContext.Key
}

func (jci Instance) GetDisplayDetails() map[string]string {
	if !jci.Installed {
		return map[string]string{
			"Setup": "In progress",
		}
	}

	return map[string]string{
		"Key":            jci.AtlassianSecurityContext.Key,
		"ClientKey":      jci.AtlassianSecurityContext.ClientKey,
		"ServerVersion":  jci.AtlassianSecurityContext.ServerVersion,
		"PluginsVersion": jci.AtlassianSecurityContext.PluginsVersion,
	}
}

func (cloudInstance Instance) GetUserConnectURL(otsStore store.OneTimeStore,
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
	token, err := cloudInstance.NewAuthToken(mattermostUserId, secret)
	if err != nil {
		return "", err
	}

	v := url.Values{}
	v.Add(ArgMMToken, token)
	return fmt.Sprintf("%v/login?dest-url=%v/plugins/servlet/ac/%s/%s?%v",
		cloudInstance.InstanceURL,
		cloudInstance.InstanceURL,
		cloudInstance.AtlassianSecurityContext.Key,
		UserLandingPageKey,
		v.Encode(),
	), nil
}

func (jci Instance) GetClient(pluginURL string, user *store.User) (*jira.Client, error) {
	oauth2Conf := oauth2_jira.Config{
		BaseURL: jci.InstanceURL,
		// TODO replace with ID
		Subject: user.Name,
	}

	oauth2Conf.Config.ClientID = jci.AtlassianSecurityContext.OAuthClientId
	oauth2Conf.Config.ClientSecret = jci.AtlassianSecurityContext.SharedSecret
	oauth2Conf.Config.Endpoint.AuthURL = "https://auth.atlassian.io"
	oauth2Conf.Config.Endpoint.TokenURL = "https://auth.atlassian.io/oauth2/token"

	httpClient := oauth2Conf.Client(context.Background())

	jiraClient, err := jira.NewClient(httpClient, oauth2Conf.BaseURL)
	return jiraClient, err
}

// Creates a "bot" client with a JWT
func (jci Instance) getClientForServer() (*jira.Client, error) {
	jwtConf := &ajwt.Config{
		Key:          jci.AtlassianSecurityContext.Key,
		ClientKey:    jci.AtlassianSecurityContext.ClientKey,
		SharedSecret: jci.AtlassianSecurityContext.SharedSecret,
		BaseURL:      jci.AtlassianSecurityContext.BaseURL,
	}

	return jira.NewClient(jwtConf.Client(), jwtConf.BaseURL)
}

func (jci Instance) JWTFromHTTPRequest(r *http.Request) (
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
		return []byte(jci.AtlassianSecurityContext.SharedSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, "", http.StatusUnauthorized, errors.WithMessage(err, "failed to validatte JWT")
	}

	return token, tokenString, http.StatusOK, nil
}
