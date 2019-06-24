// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package instance

import (
	"crypto/md5"
	"fmt"

	"github.com/mattermost/mattermost-plugin-jira/server/user"
)

type userStore struct {
	userStore user.Store
	instance  Instance
}

func NewUserStore(ustore user.Store, instance Instance) user.Store {
	return &userStore{
		userStore: ustore,
		instance:  instance,
	}
}

func (s userStore) Store(mattermostUserId string, user user.User) error {
	return s.userStore.Store(instanceKey(s.instance, mattermostUserId), user)
}

func (s userStore) Load(mattermostUserId string, userRef interface{}) error {
	return s.userStore.Load(instanceKey(s.instance, mattermostUserId), userRef)
}

func (s userStore) LoadMattermostUserId(upstreamUserKey string) (string, error) {
	return s.userStore.LoadMattermostUserId(instanceKey(s.instance, upstreamUserKey))
}

func (s userStore) Delete(mattermostUserId, userKey string) error {
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
