/*
 * Copyright 2024 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package api

import (
	"encoding/json"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
	"log"
	"net/http"
	"strings"
)

func init() {
	endpoints = append(endpoints, &PermissionsCheckEndpoints{})
}

type PermissionsCheckEndpoints struct{}

// CheckPermission godoc
// @Summary      check permission
// @Description  check permission
// @Tags         check
// @Security Bearer
// @Param        topic path string true "Topic Id"
// @Param        id path string true "Resource Id"
// @Param        permissions query string false "checked permissions in the form of 'rwxa', defaults to 'r'"
// @Produce      json
// @Success      200 {object} bool
// @Failure      400
// @Failure      401
// @Failure      500
// @Router       /check/{topic}/{id} [get]
func (this *PermissionsCheckEndpoints) CheckPermission(config configuration.Config, router *http.ServeMux, ctrl Controller) {
	router.HandleFunc("GET /check/{topic}/{id}", func(w http.ResponseWriter, req *http.Request) {
		token := jwt.GetAuthToken(req)
		topic := req.PathValue("topic")
		if topic == "" {
			http.Error(w, "missing topic", http.StatusBadRequest)
			return
		}
		id := req.PathValue("id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		permissions := req.URL.Query().Get("permissions")
		if permissions == "" {
			permissions = "r"
		}
		result, err, code := ctrl.CheckPermission(token, topic, id, permissions)
		if err != nil {
			http.Error(w, err.Error(), code)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(result)
		if err != nil {
			log.Println("ERROR: unable to encode response", err)
		}
	})
}

// CheckMultiplePermissions godoc
// @Summary      check multiple permissions
// @Description  check multiple permissions
// @Tags         check
// @Security Bearer
// @Param        topic path string true "Topic Id"
// @Param        ids query string true "Resource Ids, comma seperated"
// @Param        permissions query string false "checked permissions in the form of 'rwxa', defaults to 'r'"
// @Produce      json
// @Success      200 {object} map[string]bool
// @Failure      400
// @Failure      401
// @Failure      500
// @Router       /check/{topic} [get]
func (this *PermissionsCheckEndpoints) CheckMultiplePermissions(config configuration.Config, router *http.ServeMux, ctrl Controller) {
	router.HandleFunc("GET /check/{topic}", func(w http.ResponseWriter, req *http.Request) {
		token := jwt.GetAuthToken(req)
		topic := req.PathValue("topic")
		if topic == "" {
			http.Error(w, "missing topic", http.StatusBadRequest)
			return
		}

		ids := req.URL.Query().Get("ids")
		idList := []string{}
		for _, id := range strings.Split(ids, ",") {
			idList = append(idList, strings.TrimSpace(id))
		}

		permissions := req.URL.Query().Get("permissions")
		if permissions == "" {
			permissions = "r"
		}
		result, err, code := ctrl.CheckMultiplePermissions(token, topic, idList, permissions)
		if err != nil {
			http.Error(w, err.Error(), code)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(result)
		if err != nil {
			log.Println("ERROR: unable to encode response", err)
		}
	})
}

// ListAccessibleResourceIds godoc
// @Summary      list accessible resource ids
// @Description  list accessible resource ids
// @Tags         accessible, resource
// @Security Bearer
// @Param        topic path string true "Topic Id"
// @Param        permissions query string false "checked permissions in the form of 'rwxa', defaults to 'r'"
// @Param        limit query integer false "limits size of result; 0 means unlimited"
// @Param        offset query integer false "offset to be used in combination with limit"
// @Produce      json
// @Success      200 {array} string
// @Failure      400
// @Failure      401
// @Failure      500
// @Router       /accessible/{topic} [get]
func (this *PermissionsCheckEndpoints) ListAccessibleResourceIds(config configuration.Config, router *http.ServeMux, ctrl Controller) {
	router.HandleFunc("GET /accessible/{topic}", func(w http.ResponseWriter, req *http.Request) {
		token := jwt.GetAuthToken(req)
		topic := req.PathValue("topic")
		if topic == "" {
			http.Error(w, "missing topic", http.StatusBadRequest)
			return
		}

		permissions := req.URL.Query().Get("permissions")
		if permissions == "" {
			permissions = "r"
		}

		listOptions, err := model.ListOptionsFromQuery(req.URL.Query())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		result, err, code := ctrl.ListAccessibleResourceIds(token, topic, permissions, listOptions)
		if err != nil {
			http.Error(w, err.Error(), code)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(result)
		if err != nil {
			log.Println("ERROR: unable to encode response", err)
		}
	})
}
