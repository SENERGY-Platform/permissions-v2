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

package util

import (
	"fmt"
	"net/http"
)

func NewVersionCheck(handler http.Handler, version string) *VersionCheck {
	return &VersionCheck{handler: handler, version: version}
}

type VersionCheck struct {
	handler http.Handler
	version string
}

func (this *VersionCheck) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	version := r.URL.Query().Get("version")
	if version == "" || version == this.version {
		this.handler.ServeHTTP(w, r)
	} else {
		http.Error(w, fmt.Sprintf("if query-parameter version is set it should be '%v'; got '%v'", this.version, version), http.StatusUpgradeRequired)
	}
}
