// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira_cloud

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/lib"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/plugin"
)

const authTokenTTL = 15 * time.Minute

const ArgMMToken = "mm_token"

type AuthToken struct {
	MattermostUserID string    `json:"mattermost_user_id,omitempty"`
	Secret           string    `json:"secret,omitempty"`
	Expires          time.Time `json:"expires,omitempty"`
}

func (up JiraCloudUpstream) NewAuthToken(mattermostUserID,
	secret string) (returnToken string, returnErr error) {

	t := AuthToken{
		MattermostUserID: mattermostUserID,
		Secret:           secret,
		Expires:          time.Now().Add(authTokenTTL),
	}

	jsonBytes, err := json.Marshal(t)
	if err != nil {
		return "", errors.WithMessage(err, "NewAuthToken failed")
	}

	encrypted, err := encrypt(jsonBytes, up.Config().AuthTokenSecret)
	if err != nil {
		return "", errors.WithMessage(err, "NewAuthToken failed")
	}

	return encode(encrypted)
}

func ProcessUserConnected(api plugin.API, cloudup upstream.Upstream, ots kvstore.OneTimeStore, 
	tokenUser upstream.User, tokenSecret, mattermostUserId string) (int, error) {
	up := cloudup.(*JiraCloudUpstream)

	storedSecret, err := ots.Load(mattermostUserId)
	if err != nil {
		return http.StatusUnauthorized, errors.WithMessage(err, "failed to confirm the link")
	}
	if len(storedSecret) == 0 || string(storedSecret) != tokenSecret {
		return http.StatusUnauthorized, errors.New("this link has already been used")
	}

	err = lib.StoreUserNotify(api, up, tokenUser)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func ProcessUserDisconnected(api plugin.API, cloudup upstream.Upstream, user upstream.User) (int, error) {
	up := cloudup.(*JiraCloudUpstream)

	err := lib.DeleteUserNotify(api, up, user)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func ParseTokens(cloudup upstream.Upstream, upstreamJWT *jwt.Token,
	mmtoken, mattermostUserId string) (upstream.User, string, int, error) {
	up := cloudup.(*JiraCloudUpstream)

	claims, ok := upstreamJWT.Claims.(jwt.MapClaims)
	if !ok {
		return nil, "", http.StatusBadRequest, errors.New("invalid JWT claims")
	}
	contextClaim, ok := claims["context"].(map[string]interface{})
	if !ok {
		return nil, "", http.StatusBadRequest, errors.New("invalid JWT claim context")
	}
	userProps, ok := contextClaim["user"].(map[string]interface{})
	if !ok {
		return nil, "", http.StatusBadRequest, errors.New("invalid JWT: no user data")
	}
	userKey, _ := userProps["userKey"].(string)
	username, _ := userProps["username"].(string)
	displayName, _ := userProps["displayName"].(string)
	juser := jira.JiraUser{
		Key:         userKey,
		Name:        username,
		DisplayName: displayName,
	}

	requestedUserId, secret, err := up.parseAuthToken(mmtoken)
	if err != nil {
		return nil, "", http.StatusUnauthorized, err
	}

	if mattermostUserId != requestedUserId {
		return nil, "", http.StatusUnauthorized, errors.New("not authorized, user id does not match link")
	}

	user := jira.User{
		BasicUser: upstream.NewBasicUser(mattermostUserId, userKey),
		JiraUser:  juser,
	}

	return &user, secret, http.StatusOK, nil
}

func (up JiraCloudUpstream) parseAuthToken(encoded string) (string, string, error) {
	t := AuthToken{}
	err := func() error {
		decoded, err := decode(encoded)
		if err != nil {
			return err
		}

		jsonBytes, err := decrypt(decoded, up.Config().AuthTokenSecret)
		if err != nil {
			return err
		}

		err = json.Unmarshal(jsonBytes, &t)
		if err != nil {
			return err
		}

		if t.Expires.Before(time.Now()) {
			return errors.New("Expired token")
		}

		return nil
	}()
	if err != nil {
		return "", "", err
	}

	return t.MattermostUserID, t.Secret, nil
}

func encode(encrypted []byte) (string, error) {
	encoded := make([]byte, base64.URLEncoding.EncodedLen(len(encrypted)))
	base64.URLEncoding.Encode(encoded, encrypted)
	return string(encoded), nil
}

func encrypt(plain, secret []byte) ([]byte, error) {
	if len(secret) == 0 {
		return plain, nil
	}

	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	sealed := aesgcm.Seal(nil, nonce, []byte(plain), nil)
	return append(nonce, sealed...), nil
}

func decode(encoded string) ([]byte, error) {
	decoded := make([]byte, base64.URLEncoding.DecodedLen(len(encoded)))
	n, err := base64.URLEncoding.Decode(decoded, []byte(encoded))
	if err != nil {
		return nil, err
	}
	return decoded[:n], nil
}

func decrypt(encrypted, secret []byte) ([]byte, error) {
	if len(secret) == 0 {
		return encrypted, nil
	}

	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, errors.New("token too short")
	}

	nonce, encrypted := encrypted[:nonceSize], encrypted[nonceSize:]
	plain, err := aesgcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, err
	}

	return plain, nil
}
