/*
 * Copyright 2025 InfAI (CC SES)
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
	"net/http"
	"strings"
)

func init() {
	endpoints = append(endpoints, &ImportExportEndpoints{})
}

type ImportExportEndpoints struct{}

// Export godoc
// @Summary      export
// @Description  export
// @Tags         import/export
// @Produce      json
// @Security Bearer
// @Param        include_topic_config query bool false "default false; if true, export includes topic configurations"
// @Param        include_permissions query bool false "default false; if true, export includes resource permissions"
// @Param        filter_topics query string false "comma separated list of topics; export only topic-configs and resource-permissions with topics that are in this list"
// @Param        filter_resource_id query string false "comma separated list of ids; export only resource-permissions for the given ids"
// @Success      200 {object}  model.ImportExport
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /export [GET]
func (this *ImportExportEndpoints) Export(config configuration.Config, router *http.ServeMux, control Controller) {
	router.HandleFunc("GET /export", func(writer http.ResponseWriter, request *http.Request) {
		token := jwt.GetAuthToken(request)

		options := model.ImportExportOptions{}
		if request.URL.Query().Get("include_topic_config") == "true" {
			options.IncludeTopicConfig = true
		}
		if request.URL.Query().Get("include_permissions") == "true" {
			options.IncludePermissions = true
		}
		if request.URL.Query().Has("filter_topics") && request.URL.Query().Get("filter_topics") != "" {
			options.FilterTopics = strings.Split(request.URL.Query().Get("filter_topics"), ",")
		}
		if request.URL.Query().Has("filter_resource_id") && request.URL.Query().Get("filter_resource_id") != "" {
			options.FilterResourceId = strings.Split(request.URL.Query().Get("filter_resource_id"), ",")
		}

		result, err, code := control.Export(token, options)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		err = json.NewEncoder(writer).Encode(result)
		if err != nil {
			config.GetLogger().Error("unable to encode response", "error", err)
		}
		return
	})
}

// Import godoc
// @Summary      import
// @Description  import
// @Tags         import/export
// @Security Bearer
// @Param        include_topic_config query bool false "default false; if true, import handles topic-configurations"
// @Param        include_permissions query bool false "default false; if true, import handles resource-permissions"
// @Param        filter_topics query string false "comma separated list of topics; import handles only the given topics; null->all; []->none"
// @Param        filter_resource_id query string false "comma separated list of ids; import handles only the given resource-permissions ids; null->all; []->none"
// @Param        message body model.ImportExport true "import"
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /import [PUT]
func (this *ImportExportEndpoints) Import(config configuration.Config, router *http.ServeMux, control Controller) {
	router.HandleFunc("PUT /import", func(writer http.ResponseWriter, request *http.Request) {
		token := jwt.GetAuthToken(request)
		var importModel model.ImportExport
		err := json.NewDecoder(request.Body).Decode(&importModel)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		options := model.ImportExportOptions{}
		if request.URL.Query().Get("include_topic_config") == "true" {
			options.IncludeTopicConfig = true
		}
		if request.URL.Query().Get("include_permissions") == "true" {
			options.IncludePermissions = true
		}
		if request.URL.Query().Has("filter_topics") && request.URL.Query().Get("filter_topics") != "" {
			options.FilterTopics = strings.Split(request.URL.Query().Get("filter_topics"), ",")
		}
		if request.URL.Query().Has("filter_resource_id") && request.URL.Query().Get("filter_resource_id") != "" {
			options.FilterResourceId = strings.Split(request.URL.Query().Get("filter_resource_id"), ",")
		}

		err, code := control.Import(token, importModel, options)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.WriteHeader(http.StatusOK)
		return
	})
}
