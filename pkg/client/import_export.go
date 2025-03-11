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

package client

import (
	"bytes"
	"encoding/json"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"net/http"
	"net/url"
	"strings"
)

type ImportExportOptions = model.ImportExportOptions
type ImportExport = model.ImportExport

func (this *ClientImpl) Export(token string, options model.ImportExportOptions) (result model.ImportExport, err error, code int) {
	queryString := ""
	query := url.Values{}
	if options.IncludeTopicConfig {
		query.Set("include_topic_config", "true")
	}
	if options.IncludePermissions {
		query.Set("include_permissions", "true")
	}
	if options.FilterTopics != nil {
		query.Set("filter_topics", strings.Join(options.FilterTopics, ","))
	}
	if options.FilterResourceId != nil {
		query.Set("filter_resource_id", strings.Join(options.FilterResourceId, ","))
	}
	if len(query) > 0 {
		queryString = "?" + query.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, this.serverUrl+"/export"+queryString, nil)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	req.Header.Set("Authorization", token)
	return do[model.ImportExport](token, req)
}

func (this *ClientImpl) Import(token string, importModel model.ImportExport, options model.ImportExportOptions) (err error, code int) {
	queryString := ""
	query := url.Values{}
	if options.IncludeTopicConfig {
		query.Set("include_topic_config", "true")
	}
	if options.IncludePermissions {
		query.Set("include_permissions", "true")
	}
	if options.FilterTopics != nil {
		query.Set("filter_topics", strings.Join(options.FilterTopics, ","))
	}
	if options.FilterResourceId != nil {
		query.Set("filter_resource_id", strings.Join(options.FilterResourceId, ","))
	}
	if len(query) > 0 {
		queryString = "?" + query.Encode()
	}
	b, err := json.Marshal(importModel)
	if err != nil {
		return err, http.StatusBadRequest
	}
	req, err := http.NewRequest(http.MethodPut, this.serverUrl+"/import"+queryString, bytes.NewBuffer(b))
	if err != nil {
		return err, http.StatusInternalServerError
	}
	req.Header.Set("Authorization", token)
	return doVoid(token, req)
}
