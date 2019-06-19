// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package action

import (
	"strings"
)

type Route struct {
	Handler  Func
	Metadata interface{}
}

type Router struct {
	Routes         map[string]*Route
	DefaultHandler Func
	LogHandler     Func
}

func (router Router) RunRoute(key string, a Action) {
	key = strings.TrimRight(key, "/")

	var handler Func
	// See if we have a handler for the exact route match
	route := router.Routes[key]
	if route != nil {
		handler = route.Handler
	}

	if handler == nil {
		// Look for a subpath match
		route = router.Routes[key+"/*"]
		if route != nil {
			handler = route.Handler
		}
	}

	// Look for a /* above
	for handler == nil {
		n := strings.LastIndex(key, "/")
		if n == -1 {
			break
		}
		route = router.Routes[key[:n]+"/*"]
		if route != nil {
			handler = route.Handler
		}
		key = key[:n]
	}

	// Use the default, if needed
	if handler == nil {
		handler = router.DefaultHandler
	}

	// Run the handler
	err := handler(a)
	if err != nil {
		return
	}

	// Log
	if router.LogHandler != nil {
		_ = router.LogHandler(a)
	}
}

type Script []Func

func (script Script) Run(a Action) error {
	for _, f := range script {
		if f == nil {
			continue
		}
		err := f(a)
		if err != nil {
			a.Context().LogErr = err
			return err
		}
	}
	return nil
}
