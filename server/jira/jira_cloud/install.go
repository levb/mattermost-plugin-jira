// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira_cloud

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

func processInstalled(
	api plugin.API,
	upstore upstream.Store,
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

	// Only allow this operation once, a Jira upstream must already exist
	// for asc.BaseURL but not Installed.
	up, err := upstore.Load(asc.BaseURL)
	if err == kvstore.ErrNotFound {
		return http.StatusNotFound,
			errors.Errorf("Jira upstream %q must first be added to Mattermost", asc.BaseURL)
	}
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessagef(err, "failed to load Jira upstream %q", asc.BaseURL)
	}
	cloudUp, ok := up.(*Upstream)
	if !ok {
		return http.StatusInternalServerError,
			errors.Errorf("expected a Jira Cloud upstream, got %T", up)
	}
	if cloudUp.Installed {
		return http.StatusForbidden,
			errors.Errorf("Jira upstream %q is already installed", asc.BaseURL)
	}

	up = newUpstream(upstore, true, string(data), &asc)

	// UpstreamStore.Store also updates the list of known Jira upstreams
	err = upstore.Store(up)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	err = proxy.StoreCurrentUpstreamNotify(api, upstore, up)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}
