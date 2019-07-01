// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira_cloud

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/plugin"
)

const authTokenTTL = 15 * time.Minute

type authToken struct {
	MattermostUserID string    `json:"mattermost_user_id,omitempty"`
	Secret           string    `json:"secret,omitempty"`
	Expires          time.Time `json:"expires,omitempty"`
}

func (up Upstream) newAuthToken(mattermostUserID,
	secret string) (returnToken string, returnErr error) {

	t := authToken{
		MattermostUserID: mattermostUserID,
		Secret:           secret,
		Expires:          time.Now().Add(authTokenTTL),
	}

	jsonBytes, err := json.Marshal(t)
	if err != nil {
		return "", errors.WithMessage(err, "NewAuthToken failed")
	}

	fmt.Printf("<><> NewAuthToken 1 %s\n", string(jsonBytes))
	fmt.Printf("<><> NewAuthToken 2 %s\n", string(up.Config().AuthTokenSecret))
	encrypted, err := encrypt(jsonBytes, up.Config().AuthTokenSecret)
	if err != nil {
		return "", errors.WithMessage(err, "NewAuthToken failed")
	}
	fmt.Printf("<><> NewAuthToken 3 %v\n", len(encrypted))

	return encode(encrypted)
}

func processUserConnected(api plugin.API, cloudup upstream.Upstream, ots kvstore.OneTimeStore,
	tokenUser *jira.User, tokenSecret string, mattermostUserId string) (int, error) {
	up := cloudup.(*Upstream)

	storedSecret, err := ots.Load(mattermostUserId)
	if err != nil {
		return http.StatusUnauthorized, errors.WithMessage(err, "failed to confirm the link")
	}
	if len(storedSecret) == 0 || string(storedSecret) != tokenSecret {
		return http.StatusUnauthorized, errors.New("this link has already been used")
	}

	err = proxy.StoreUserNotify(api, up, tokenUser)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func processUserDisconnected(api plugin.API, cloudup upstream.Upstream, user upstream.User) (int, error) {
	up := cloudup.(*Upstream)

	err := proxy.DeleteUserNotify(api, up, user)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (up Upstream) parseAuthToken(encoded string) (string, string, error) {
	t := authToken{}
	err := func() error {
		decoded, err := decode(encoded)
		if err != nil {
			return err
		}

		jsonBytes, err := decrypt(decoded, up.Config().AuthTokenSecret)
		if err != nil {
			return err
		}

		fmt.Printf("<><> jsonBytes %v\n", string(jsonBytes))

		err = json.Unmarshal(jsonBytes, &t)
		if err != nil {
			return err
		}

		fmt.Printf("<><> t %+v\n", t)
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
