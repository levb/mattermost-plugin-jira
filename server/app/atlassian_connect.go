// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package app

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/instance"
	"github.com/mattermost/mattermost-plugin-jira/server/instance/jira_cloud"
	"github.com/mattermost/mattermost-plugin-jira/server/store"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

func ProcessACInstalled(
	api plugin.API,
	instanceStore instance.InstanceStore,
	currentInstanceStore instance.CurrentInstanceStore,
	ensuredStore store.EnsuredStore,
	body io.Reader) (int, error) {

	data, err := ioutil.ReadAll(body)
	if err != nil {
		return http.StatusBadRequest, errors.WithMessage(err, "failed to read request")
	}

	var asc jira_cloud.AtlassianSecurityContext
	err = json.Unmarshal(data, &asc)
	if err != nil {
		return http.StatusBadRequest, errors.WithMessage(err, "failed to unmarshal request")
	}

	// Only allow this operation once, a Jira instance must already exist
	// for asc.BaseURL but not Installed.
	cloudInstance := &jira_cloud.JiraCloudInstance{}
	err = instanceStore.Load(asc.BaseURL, &cloudInstance)
	if err == store.ErrNotFound {
		return http.StatusNotFound,
			errors.Errorf("Jira instance %q must first be added to Mattermost", asc.BaseURL)
	}
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessagef(err, "failed to load instance %q", asc.BaseURL)
	}
	if cloudInstance.Installed {
		return http.StatusForbidden,
			errors.Errorf("Jira instance %q is already installed", asc.BaseURL)
	}

	encryptSecret, err := cloudInstance.EnsureAuthTokenEncryptSecret(ensuredStore)
	if cloudInstance.Installed {
		return http.StatusForbidden, err
	}

	// Create a permanent instance record, also store it as current
	cloudInstance = jira_cloud.NewCloudInstance(asc.BaseURL, true, string(data), &asc, encryptSecret)

	// StoreInstance also updates the list of known Jira instances
	err = instanceStore.Store(cloudInstance)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	err = StoreCurrentInstanceAndNotify(api, currentInstanceStore, cloudInstance)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}
