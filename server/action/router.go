// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package action

import (
	"fmt"
	"strings"
)

type Route struct {
	Script   Script
	Metadata interface{}
}

func NewRoute(ff ...Func) *Route {
	return &Route{
		Script: Script(ff),
	}
}

func (r *Route) With(metadata interface{}) *Route {
	r.Metadata = metadata
	return r
}

type Router struct {
	Routes  map[string]*Route
	Before  Script
	After   Script
	Default Func
}

func (router *Router) AddRoute(key string, route Route) {
	router.Routes[key] = &route
}

func (router *Router) AddRoutes(routes map[string]*Route) {
	for k, v := range routes {
		router.Routes[k] = v
	}
}

func AppendRoutes(routes ...map[string]*Route) map[string]*Route {
	to := map[string]*Route{}
	for _, set := range routes {
		for k, v := range set {
			to[k] = v
		}
	}
	return to
}

func (router Router) RunRoute(key string, a Action) {
	key = strings.TrimRight(key, "/")

	var handler Script
	// See if we have a handler for the exact route match
	route := router.Routes[key]
	if route != nil {
		handler = route.Script
	}

	if handler == nil {
		// Look for a subpath match
		route = router.Routes[key+"/*"]
		if route != nil {
			handler = route.Script
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
			handler = route.Script
		}
		key = key[:n]
	}

	// Use the default, if needed
	if handler == nil {
		handler = Script{router.Default}
	}

	if len(router.Before) > 0 {
		err := router.Before.Run(a)
		if err != nil {
			a.Context().LogErr = err
			return
		}
	}

	err := handler.Run(a)
	if err != nil {
		fmt.Printf("<><> Got error %v\n", err)
		a.Context().LogErr = err
	}

	if len(router.After) > 0 {
		_ = router.After.Run(a)
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
			return err
		}
	}
	return nil
}
