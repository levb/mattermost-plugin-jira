// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package instance

import (
	"crypto/md5"
	"fmt"

	"github.com/mattermost/mattermost-plugin-jira/server/store"
)

type userStore struct {
	userStore store.UserStore
	instance  Instance
}

func NewUserStore(us store.UserStore, instance Instance) store.UserStore {
	return &userStore{
		userStore: us,
		instance:  instance,
	}
}

func (s userStore) Store(mattermostUserId string, user *store.User) error {
	return s.userStore.Store(instanceKey(s.instance, mattermostUserId), user)
}

func (s userStore) Load(mattermostUserId string) (*store.User, error) {
	return s.userStore.Load(instanceKey(s.instance, mattermostUserId))
}

func (s userStore) LoadMattermostUserId(upstreamUserKey string) (string, error) {
	return s.userStore.LoadMattermostUserId(instanceKey(s.instance, upstreamUserKey))
}

func (s userStore) Delete(mattermostUserId string) error {
	return s.userStore.Delete(instanceKey(s.instance, mattermostUserId))
}

func instanceKey(instance Instance, key string) string {
	if disablePrefixForInstance {
		h := md5.New()
		fmt.Fprintf(h, "%s/%s", instance.GetURL(), key)
		key = fmt.Sprintf("%x", h.Sum(nil))
	}
	return key
}
