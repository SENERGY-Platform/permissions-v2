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
)

func init() {
	endpoints = append(endpoints, &AdminEndpoints{})
}

type AdminEndpoints struct{}

// AdminListResourceIds godoc
// @Summary      lists resource ids in topic
// @Description  lists resource ids in topic, requesting user must be in admin group
// @Tags         topics, resources, admin
// @Security Bearer
// @Param        topic path string true "Topic Id"
// @Param        limit query integer false "limits size of result; 0 means unlimited"
// @Param        offset query integer false "offset to be used in combination with limit"
// @Produce      json
// @Success      200 {array}  string
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      500
// @Router       /admin/resources/{topic} [get]
func (this *TopicsEndpoints) AdminListResourceIds(config configuration.Config, router *http.ServeMux, ctrl Controller) {
	router.HandleFunc("GET /admin/resources/{topic}", func(w http.ResponseWriter, req *http.Request) {
		token := jwt.GetAuthToken(req)
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
		result, err, code := ctrl.AdminListResourceIds(token, topic, listOptions)
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
