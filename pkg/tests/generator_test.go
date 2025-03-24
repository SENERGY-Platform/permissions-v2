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

package tests

import (
	"github.com/SENERGY-Platform/permissions-v2/pkg/client"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestGenerator(t *testing.T) {
	filename := t.TempDir() + "/test.go"
	//filename := "./testgen.go"
	err := client.GenerateGoFileWithSwaggoCommentsForEmbeddedPermissionsClient(
		"test",
		"permissions",
		filename,
		[]string{"foo", "bar"},
		func(method string, path string) bool {
			if method == http.MethodDelete {
				return false
			}
			if strings.Contains(path, "admin") {
				return false
			}
			return true
		})
	if err != nil {
		t.Error(err)
		return
	}

	temp, err := os.ReadFile(filename)
	if err != nil {
		t.Error(err)
		return
	}

	if strings.Contains(string(temp), "DELETE") {
		t.Error("found filtered delete")
	}

	if strings.Contains(string(temp), "/admin/") {
		t.Error("found filtered admin path")
	}

	if !strings.Contains(string(temp), "package test") {
		t.Error("missing package name")
	}
	if strings.Contains(string(temp), "{topic}") {
		t.Error("found topic placeholder")
	}
	if strings.Contains(string(temp), "GET /manage/foo") {
		t.Error("missing foo")
	}
	if strings.Contains(string(temp), "GET /manage/bar") {
		t.Error("missing foo")
	}
	if strings.Contains(string(temp), "topic path string true") {
		t.Error("found topic path param")
	}

}
