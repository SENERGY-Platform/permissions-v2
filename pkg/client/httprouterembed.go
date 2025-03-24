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

package client

import (
	_ "embed"
	"github.com/SENERGY-Platform/permissions-v2/pkg/api"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"net/http"
	"strings"
)

func trimPrefixPath(path string, prefix string) string {
	newPath := strings.TrimPrefix(path, prefix)
	if !strings.HasPrefix(newPath, "/") {
		newPath = "/" + newPath
	}
	return newPath
}

func EmbedPermissionsClientIntoRouter(client Client, router http.Handler, prefix string, pathFilter func(method string, path string) bool) http.Handler {
	permRouter := api.GetRouterWithoutMiddleware(configuration.Config{
		EnableSwaggerUi: false,
	}, client)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, prefix) && (pathFilter == nil || pathFilter(r.Method, r.URL.Path)) {
			r.URL.Path = trimPrefixPath(r.URL.Path, prefix)
			r.URL.RawPath = trimPrefixPath(r.URL.RawPath, prefix)
			r.RequestURI = trimPrefixPath(r.RequestURI, prefix)
			permRouter.ServeHTTP(w, r)
			return
		} else {
			router.ServeHTTP(w, r)
		}
	})
}
