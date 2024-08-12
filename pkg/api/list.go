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
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
	"log"
	"net/http"
	"strings"
)

func init() {
	endpoints = append(endpoints, &PermissionsListEndpoints{})
}

type PermissionsListEndpoints struct{}

// ListComputedPermissions godoc
// @Summary      list the computed permissions to resources of the given topic and ids
// @Description  list the computed permissions to resources of the given topic and ids, group and user permissions are merged, unknown ids will get entries in the result
// @Tags         permissions, check, list
// @Security Bearer
// @Param        topic path string true "Topic Id"
// @Param        ids query string true "Resource Ids, comma seperated"
// @Produce      json
// @Success      200 {array} model.ComputedPermissions
// @Failure      400
// @Failure      401
// @Failure      500
// @Router       /permissions/{topic} [get]
func (this *PermissionsListEndpoints) ListComputedPermissions(config configuration.Config, router *http.ServeMux, ctrl Controller) {
	router.HandleFunc("GET /permissions/{topic}", func(w http.ResponseWriter, req *http.Request) {
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

		result, err, code := ctrl.ListComputedPermissions(token, topic, idList)
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

// QueryListComputedPermissions godoc
// @Summary      list the computed permissions to resources of the given topic and ids
// @Description  list the computed permissions to resources of the given topic and ids, group and user permissions are merged, unknown ids will get entries in the result
// @Tags         permissions, check, list, query
// @Security Bearer
// @Param        topic path string true "Topic Id"
// @Param        ids body []string true "Resource Ids"
// @Produce      json
// @Success      200 {array} model.ComputedPermissions
// @Failure      400
// @Failure      401
// @Failure      500
// @Router       /query/permissions/{topic} [post]
func (this *PermissionsListEndpoints) QueryListComputedPermissions(config configuration.Config, router *http.ServeMux, ctrl Controller) {
	router.HandleFunc("POST /query/permissions/{topic}", func(w http.ResponseWriter, req *http.Request) {
		token := jwt.GetAuthToken(req)
		topic := req.PathValue("topic")
		if topic == "" {
			http.Error(w, "missing topic", http.StatusBadRequest)
			return
		}

		idList := []string{}
		err := json.NewDecoder(req.Body).Decode(&idList)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		result, err, code := ctrl.ListComputedPermissions(token, topic, idList)
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
