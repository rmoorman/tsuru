// Copyright 2017 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"encoding/json"
	"net/http"

	"github.com/ajg/form"
	"github.com/pkg/errors"
	"github.com/tsuru/tsuru/auth"
	tsuruErrors "github.com/tsuru/tsuru/errors"
	"github.com/tsuru/tsuru/event"
	"github.com/tsuru/tsuru/permission"
	"github.com/tsuru/tsuru/provision/cluster"
)

// title: create or update provisioner cluster
// path: /provisioner/clusters
// method: POST
// consume: application/x-www-form-urlencoded
// produce: application/json
// responses:
//   200: Ok
//   400: Invalid data
//   401: Unauthorized
//   409: Cluster already exists
func updateCluster(w http.ResponseWriter, r *http.Request, t auth.Token) (err error) {
	allowed := permission.Check(t, permission.PermClusterUpdate)
	if !allowed {
		return permission.ErrUnauthorized
	}
	dec := form.NewDecoder(nil)
	dec.IgnoreCase(true)
	dec.IgnoreUnknownKeys(true)
	var provCluster cluster.Cluster
	err = r.ParseForm()
	if err == nil {
		err = dec.DecodeValues(&provCluster, r.Form)
	}
	if err != nil {
		return &tsuruErrors.HTTP{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		}
	}
	evt, err := event.New(&event.Opts{
		Target:     event.Target{Type: event.TargetTypeCluster, Value: provCluster.Name},
		Kind:       permission.PermClusterUpdate,
		Owner:      t,
		CustomData: event.FormToCustomData(r.Form),
		Allowed:    event.Allowed(permission.PermClusterReadEvents),
	})
	if err != nil {
		return err
	}
	defer func() { evt.Done(err) }()
	err = provCluster.Save()
	if err != nil {
		return errors.WithStack(err)
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

// title: list provisioner clusters
// path: /provisioner/clusters
// method: GET
// consume: application/x-www-form-urlencoded
// produce: application/json
// responses:
//   200: Ok
//   204: No Content
//   401: Unauthorized
func listClusters(w http.ResponseWriter, r *http.Request, t auth.Token) (err error) {
	allowed := permission.Check(t, permission.PermClusterRead)
	if !allowed {
		return permission.ErrUnauthorized
	}
	clusters, err := cluster.AllClusters()
	if err != nil {
		if err == cluster.ErrNoCluster {
			w.WriteHeader(http.StatusNoContent)
			return nil
		}
		return err
	}
	return json.NewEncoder(w).Encode(clusters)
}

// title: delete provisioner cluster
// path: /provisioner/clusters/{name}
// method: GET
// consume: application/x-www-form-urlencoded
// produce: application/json
// responses:
//   200: Ok
//   401: Unauthorized
//   404: Cluster not found
func deleteCluster(w http.ResponseWriter, r *http.Request, t auth.Token) (err error) {
	allowed := permission.Check(t, permission.PermClusterDelete)
	if !allowed {
		return permission.ErrUnauthorized
	}
	r.ParseForm()
	clusterName := r.URL.Query().Get(":name")
	evt, err := event.New(&event.Opts{
		Target:     event.Target{Type: event.TargetTypeCluster, Value: clusterName},
		Kind:       permission.PermClusterDelete,
		Owner:      t,
		CustomData: event.FormToCustomData(r.Form),
		Allowed:    event.Allowed(permission.PermClusterReadEvents),
	})
	if err != nil {
		return err
	}
	defer func() { evt.Done(err) }()
	err = cluster.DeleteCluster(clusterName)
	if err != nil {
		if errors.Cause(err) == cluster.ErrClusterNotFound {
			return &tsuruErrors.HTTP{
				Code:    http.StatusNotFound,
				Message: err.Error(),
			}
		}
		return err
	}
	return nil
}
