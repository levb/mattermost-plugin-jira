// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package instance

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/url"
	"net/http"

	"github.com/andygrunwald/go-jira"
	ajwt "github.com/rbriski/atlassian-jwt"
	oauth2_jira "golang.org/x/oauth2/jira"
	"github.com/dgrijalva/jwt-go"
)

type jiraCloudInstance struct {
	*instance

	// Initially a new instance is created with an expiration time. The
	// admin is expected to upload it to the Jira instance, and we will
	// then receive a /installed callback that initializes the instance
	// and makes it permanent. No subsequent /installed will be accepted
	// for the instance.
	Installed bool

	// For cloud instances (atlassian-connect.json install and user auth)
	RawAtlassianSecurityContext string
	*AtlassianSecurityContext   `json:"none"`
}

var _ Instance = (*jiraCloudInstance)(nil)

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

func NewCloudInstance(key string, installed bool, rawASC string,
	asc *AtlassianSecurityContext) *jiraCloudInstance {

	return &jiraCloudInstance{
		instance:                    newInstance(InstanceTypeCloud, key),
		Installed:                   installed,
		RawAtlassianSecurityContext: rawASC,
		AtlassianSecurityContext:    asc,
	}
}

func (jci jiraCloudInstance) GetMattermostKey() string {
	return jci.AtlassianSecurityContext.Key
}

func (jci jiraCloudInstance) GetDisplayDetails() map[string]string {
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

func (jci jiraCloudInstance) GetUserConnectURL(conf Config, secretsStore SecretStore,
	mattermostUserId string) (string, error) {

	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	secret := fmt.Sprintf("%x", randomBytes)
	err = secretsStore.StoreOneTimeSecret(mattermostUserId, secret)
	if err != nil {
		return "", err
	}
	token, err := NewEncodedAuthToken(secretsStore, mattermostUserId, secret)
	if err != nil {
		return "", err
	}

	v := url.Values{}
	v.Add(argMMToken, token)
	return fmt.Sprintf("%v/login?dest-url=%v/plugins/servlet/ac/%s/%s?%v",
		jci.GetURL(), jci.GetURL(), jci.AtlassianSecurityContext.Key, userRedirectPageKey, v.Encode()), nil
}

func (jci jiraCloudInstance) GetURL() string {
	return jci.AtlassianSecurityContext.BaseURL
}

func (jci jiraCloudInstance) GetClient(conf Config, secretsStore SecretStore,
	jiraUser *JiraUser) (*jira.Client, error) {

	oauth2Conf := oauth2_jira.Config{
		BaseURL: jci.GetURL(),
		Subject: jiraUser.Name,
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
func (jci jiraCloudInstance) getClientForServer() (*jira.Client, error) {
	jwtConf := &ajwt.Config{
		Key:          jci.AtlassianSecurityContext.Key,
		ClientKey:    jci.AtlassianSecurityContext.ClientKey,
		SharedSecret: jci.AtlassianSecurityContext.SharedSecret,
		BaseURL:      jci.AtlassianSecurityContext.BaseURL,
	}

	return jira.NewClient(jwtConf.Client(), jwtConf.BaseURL)
}

func (jci jiraCloudInstance) JWTFromHTTP(r *http.Request) (
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
		return []byte(a.JiraCloudInstance.AtlassianSecurityContext.SharedSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, "", http.StatusUnauthorized, errors.WithMessage(err, "failed to validatte JWT")
	}

	a.Debugf("action: verified Jira JWT")
	return token, tokenString, http.StatusOK, nil
}
