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
	"fmt"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
	"log"
	"net/http"
	"strconv"
)

func init() {
	endpoints = append(endpoints, &PermissionsManagementEndpoints{})
}

type PermissionsManagementEndpoints struct{}

// ListResourcesWithAdminPermission godoc
// @Summary      lists resources the user has admin rights to
// @Description  lists resources the user has admin rights to
// @Tags         manage, resource
// @Param        topic path string true "Topic Id"
// @Param        limit query integer false "limits size of result; 0 means unlimited"
// @Param        offset query integer false "offset to be used in combination with limit"
// @Produce      json
// @Success      200 {array}  model.Resource
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      500
// @Router       /manage/{topic} [get]
func (this *PermissionsManagementEndpoints) ListResourcesWithAdminPermission(config configuration.Config, router *http.ServeMux, ctrl Controller) {
	router.HandleFunc("GET /manage/{topic}", func(w http.ResponseWriter, req *http.Request) {
		token, err := jwt.GetParsedToken(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		topic := req.PathValue("topic")
		if topic == "" {
			http.Error(w, "missing topic", http.StatusBadRequest)
			return
		}

		listOptions, err := model.ListOptionsFromQuery(req.URL.Query())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		result, err, code := ctrl.ListResourcesWithAdminPermission(token, topic, listOptions)
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

// GetResource godoc
// @Summary      get resource
// @Description  get resource, requesting user must have admin right
// @Tags         manage, resource
// @Param        topic path string true "Topic Id"
// @Param        id path string true "Resource Id"
// @Produce      json
// @Success      200 {object}  model.Resource
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      500
// @Router       /manage/{topic}/{id} [get]
func (this *PermissionsManagementEndpoints) GetResource(config configuration.Config, router *http.ServeMux, ctrl Controller) {
	router.HandleFunc("GET /manage/{topic}/{id}", func(w http.ResponseWriter, req *http.Request) {
		token, err := jwt.GetParsedToken(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
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

		result, err, code := ctrl.GetResource(token, topic, id)
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

// SetPermission godoc
// @Summary      set resource rights
// @Description  get resource rights, requesting user must have admin right
// @Tags         manage, resource-rights
// @Param        topic path string true "Topic Id"
// @Param        id path string true "Resource Id"
// @Param        wait query bool false "if set to true, the response will be sent after the corresponding kafka done signal has been received"
// @Param        message body model.ResourcePermissions true "Topic"
// @Accept       json
// @Produce      json
// @Success      200 {object}  model.ResourcePermissions
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      500
// @Router       /manage/{topic}/{id} [put]
func (this *PermissionsManagementEndpoints) SetPermission(config configuration.Config, router *http.ServeMux, ctrl Controller) {
	router.HandleFunc("PUT /manage/{topic}/{id}", func(w http.ResponseWriter, req *http.Request) {
		token, err := jwt.GetParsedToken(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
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

		options := model.SetPermissionOptions{}
		if waitQueryParam := req.URL.Query().Get("wait"); waitQueryParam != "" {
			options.Wait, err = strconv.ParseBool(waitQueryParam)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid wait query parameter %v", err.Error()), http.StatusBadRequest)
				return
			}
		}

		permissions := model.ResourcePermissions{}
		err = json.NewDecoder(req.Body).Decode(&permissions)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		result, err, code := ctrl.SetPermission(token, topic, id, permissions, options)
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
