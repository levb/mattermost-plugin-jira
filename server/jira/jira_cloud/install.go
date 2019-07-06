// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira_cloud

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

func processInstalled(
	api plugin.API,
	upstore upstream.UpstreamStore,
	ots kvstore.OneTimeStore,
	authTokenSecret []byte,
	body io.Reader) (int, error) {

	data, err := ioutil.ReadAll(body)
	if err != nil {
		return http.StatusBadRequest, errors.WithMessage(err, "failed to read request")
	}

	var asc atlassianSecurityContext
	err = json.Unmarshal(data, &asc)
	if err != nil {
		return http.StatusBadRequest, errors.WithMessage(err, "failed to unmarshal request")
	}

	// backed by a one-time store, will only work once
	err = loadUnconfirmedUpstream(ots, asc.BaseURL)
	if err == kvstore.ErrNotFound {
		return http.StatusNotFound,
			errors.Errorf("not found, already used, or expired: %q", asc.BaseURL)
	}
	if err != nil {
		return http.StatusInternalServerError, err
	}

	up := newUpstream(upstore, string(data), &asc)

	// UpstreamStore.Store also updates the list of known Jira upstreams
	err = upstore.StoreUpstream(up)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	err = upstore.StoreCurrentUpstreamNotify(up)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}
