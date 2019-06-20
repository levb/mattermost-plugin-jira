// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package instance

import (
	"github.com/pkg/errors"

	"github.com/andygrunwald/go-jira"

	"github.com/mattermost/mattermost-plugin-jira/server/store"
)

const wSEventInstanceStatus = "instance_status"

var ErrWrongInstanceType = errors.New("wrong instance type")

type Instance interface {
	GetKey() string
	GetType() string
	GetDisplayDetails() map[string]string
	GetMattermostKey() string
	GetURL() string
	GetUserConnectURL(ots store.OneTimeStore, pluginURL string, mattermostUserId string) (string, error)
	GetClient(string, *store.User) (*jira.Client, error)
}

type BasicInstance struct {
	InstanceKey  string `json:"Key"`
	InstanceType string `json:"Type"`
	InstanceURL  string `json:"URL"`
}

type InstanceStatus struct {
	InstanceInstalled bool `json:"instance_installed"`
}

func (instance BasicInstance) GetKey() string {
	return instance.InstanceKey
}

func (instance BasicInstance) GetType() string {
	return instance.InstanceType
}

func (instance BasicInstance) GetURL() string {
	return instance.InstanceURL
}
