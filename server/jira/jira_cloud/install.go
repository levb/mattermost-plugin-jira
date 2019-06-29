// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira_cloud

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/lib"
	"github.com/mattermost/mattermost-plugin-jira/server/store"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/plugin"
)

func ProcessInstalled(
	api plugin.API,
	upstore upstream.Store,
	authTokenSecret []byte,
	body io.Reader) (int, error) {

	data, err := ioutil.ReadAll(body)
	if err != nil {
		return http.StatusBadRequest, errors.WithMessage(err, "failed to read request")
	}

	var asc AtlassianSecurityContext
	err = json.Unmarshal(data, &asc)
	if err != nil {
		return http.StatusBadRequest, errors.WithMessage(err, "failed to unmarshal request")
	}

	// Only allow this operation once, a Jira upstream must already exist
	// for asc.BaseURL but not Installed.
	up, err := upstore.Load(asc.BaseURL)
	if err == store.ErrNotFound {
		return http.StatusNotFound,
			errors.Errorf("Jira upstream %q must first be added to Mattermost", asc.BaseURL)
	}
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessagef(err, "failed to load Jira upstream %q", asc.BaseURL)
	}
	cloudUp, ok := up.(*JiraCloudUpstream)
	if !ok {
		return http.StatusInternalServerError,
			errors.Errorf("expected a Jira Cloud upstream, got %T", up)
	}
	if cloudUp.Installed {
		return http.StatusForbidden,
			errors.Errorf("Jira upstream %q is already installed", asc.BaseURL)
	}

	up = NewUpstream(nil, true, string(data), &asc, authTokenSecret)

	// UpstreamStore.Store also updates the list of known Jira upstreams
	err = upstore.Store(up)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	err = lib.StoreCurrentUpstreamNotify(api, upstore, up)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}
