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

package tests

import (
	"context"
	"github.com/SENERGY-Platform/permissions-v2/pkg"
	"github.com/SENERGY-Platform/permissions-v2/pkg/client"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/permissions-v2/pkg/tests/docker"
	"net/http"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestDefaultPermissions(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	config.Debug = true
	config.DevNotifierUrl = ""

	dockerPort, _, err := docker.MongoDB(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.MongoUrl = "mongodb://localhost:" + dockerPort

	freePort, err := docker.GetFreePort()
	if err != nil {
		t.Error(err)
		return
	}
	config.Port = strconv.Itoa(freePort)

	err = pkg.Start(ctx, config)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(time.Second)

	c := client.New("http://localhost:" + config.Port)

	t.Run("init", func(t *testing.T) {
		_, err, _ = c.SetTopic(client.InternalAdminToken, model.Topic{
			Id: "partial",
			DefaultPermissions: model.ResourcePermissions{
				GroupPermissions: map[string]model.PermissionsMap{
					GroupTestTokenGroup: {
						Read:         true,
						Write:        false,
						Execute:      false,
						Administrate: false,
					},
				},
				RolePermissions: map[string]model.PermissionsMap{
					"user": {
						Read:         true,
						Write:        false,
						Execute:      true,
						Administrate: false,
					},
				},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		_, err, _ = c.SetTopic(client.InternalAdminToken, model.Topic{
			Id: "full",
			DefaultPermissions: model.ResourcePermissions{
				GroupPermissions: map[string]model.PermissionsMap{
					GroupTestTokenGroup: {
						Read:         true,
						Write:        true,
						Execute:      true,
						Administrate: true,
					},
				},
				RolePermissions: map[string]model.PermissionsMap{
					"user": {
						Read:         true,
						Write:        true,
						Execute:      true,
						Administrate: true,
					},
				},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		_, err, _ = c.SetTopic(client.InternalAdminToken, model.Topic{
			Id: "none",
			DefaultPermissions: model.ResourcePermissions{
				GroupPermissions: map[string]model.PermissionsMap{
					GroupTestTokenGroup: {
						Read:         false,
						Write:        false,
						Execute:      false,
						Administrate: false,
					},
				},
				RolePermissions: map[string]model.PermissionsMap{
					"user": {
						Read:         false,
						Write:        false,
						Execute:      false,
						Administrate: false,
					},
				},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
	})
	t.Run("set permissions", func(t *testing.T) {
		_, err, _ = c.SetPermission(client.InternalAdminToken, "full", "a", model.ResourcePermissions{
			UserPermissions: map[string]model.PermissionsMap{"someone": {Read: true, Write: true, Execute: true, Administrate: true}},
		})
		if err != nil {
			t.Error(err)
			return
		}
		_, err, _ = c.SetPermission(client.InternalAdminToken, "partial", "a", model.ResourcePermissions{
			UserPermissions: map[string]model.PermissionsMap{"someone": {Read: true, Write: true, Execute: true, Administrate: true}},
		})
		if err != nil {
			t.Error(err)
			return
		}
		_, err, _ = c.SetPermission(client.InternalAdminToken, "none", "a", model.ResourcePermissions{
			UserPermissions: map[string]model.PermissionsMap{"someone": {Read: true, Write: true, Execute: true, Administrate: true}},
		})
		if err != nil {
			t.Error(err)
			return
		}
	})
	t.Run("check", func(t *testing.T) {
		t.Run("partial", func(t *testing.T) {
			t.Run("admin", func(t *testing.T) {
				t.Run("r", func(t *testing.T) {
					access, err, _ := c.CheckPermission(client.InternalAdminToken, "partial", "a", model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(client.InternalAdminToken, "partial", []string{"a"}, model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("rw", func(t *testing.T) {
					access, err, _ := c.CheckPermission(client.InternalAdminToken, "partial", "a", model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(client.InternalAdminToken, "partial", []string{"a"}, model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("rx", func(t *testing.T) {
					access, err, _ := c.CheckPermission(client.InternalAdminToken, "partial", "a", model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(client.InternalAdminToken, "partial", []string{"a"}, model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("a", func(t *testing.T) {
					access, err, _ := c.CheckPermission(client.InternalAdminToken, "partial", "a", model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(client.InternalAdminToken, "partial", []string{"a"}, model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
			})

			t.Run("group", func(t *testing.T) {
				t.Run("r", func(t *testing.T) {
					access, err, _ := c.CheckPermission(GroupTestToken, "partial", "a", model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(GroupTestToken, "partial", []string{"a"}, model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("rw", func(t *testing.T) {
					access, err, _ := c.CheckPermission(GroupTestToken, "partial", "a", model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if access {
						t.Error("unexpected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(GroupTestToken, "partial", []string{"a"}, model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": false}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("rx", func(t *testing.T) {
					access, err, _ := c.CheckPermission(GroupTestToken, "partial", "a", model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if access {
						t.Error("unexpected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(GroupTestToken, "partial", []string{"a"}, model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": false}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("a", func(t *testing.T) {
					access, err, _ := c.CheckPermission(GroupTestToken, "partial", "a", model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if access {
						t.Error("unexpected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(GroupTestToken, "partial", []string{"a"}, model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": false}) {
						t.Error("expected access")
						return
					}
				})
			})

			t.Run("user", func(t *testing.T) {
				t.Run("r", func(t *testing.T) {
					access, err, _ := c.CheckPermission(TestToken, "partial", "a", model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(TestToken, "partial", []string{"a"}, model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("rw", func(t *testing.T) {
					access, err, _ := c.CheckPermission(TestToken, "partial", "a", model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if access {
						t.Error("unexpected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(TestToken, "partial", []string{"a"}, model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": false}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("rx", func(t *testing.T) {
					access, err, _ := c.CheckPermission(TestToken, "partial", "a", model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(TestToken, "partial", []string{"a"}, model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("a", func(t *testing.T) {
					access, err, _ := c.CheckPermission(TestToken, "partial", "a", model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if access {
						t.Error("unexpected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(TestToken, "partial", []string{"a"}, model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": false}) {
						t.Error("expected access")
						return
					}
				})
			})
		})

		t.Run("full", func(t *testing.T) {
			t.Run("admin", func(t *testing.T) {
				t.Run("r", func(t *testing.T) {
					access, err, _ := c.CheckPermission(client.InternalAdminToken, "full", "a", model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(client.InternalAdminToken, "full", []string{"a"}, model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("rw", func(t *testing.T) {
					access, err, _ := c.CheckPermission(client.InternalAdminToken, "full", "a", model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(client.InternalAdminToken, "full", []string{"a"}, model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("rx", func(t *testing.T) {
					access, err, _ := c.CheckPermission(client.InternalAdminToken, "full", "a", model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(client.InternalAdminToken, "full", []string{"a"}, model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("a", func(t *testing.T) {
					access, err, _ := c.CheckPermission(client.InternalAdminToken, "full", "a", model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(client.InternalAdminToken, "full", []string{"a"}, model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
			})

			t.Run("group", func(t *testing.T) {
				t.Run("r", func(t *testing.T) {
					access, err, _ := c.CheckPermission(GroupTestToken, "full", "a", model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(GroupTestToken, "full", []string{"a"}, model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("rw", func(t *testing.T) {
					access, err, _ := c.CheckPermission(GroupTestToken, "full", "a", model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(GroupTestToken, "full", []string{"a"}, model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("rx", func(t *testing.T) {
					access, err, _ := c.CheckPermission(GroupTestToken, "full", "a", model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(GroupTestToken, "full", []string{"a"}, model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("a", func(t *testing.T) {
					access, err, _ := c.CheckPermission(GroupTestToken, "full", "a", model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(GroupTestToken, "full", []string{"a"}, model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
			})

			t.Run("user", func(t *testing.T) {
				t.Run("r", func(t *testing.T) {
					access, err, _ := c.CheckPermission(TestToken, "full", "a", model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(TestToken, "full", []string{"a"}, model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("rw", func(t *testing.T) {
					access, err, _ := c.CheckPermission(TestToken, "full", "a", model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(TestToken, "full", []string{"a"}, model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("rx", func(t *testing.T) {
					access, err, _ := c.CheckPermission(TestToken, "full", "a", model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(TestToken, "full", []string{"a"}, model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("a", func(t *testing.T) {
					access, err, _ := c.CheckPermission(TestToken, "full", "a", model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(TestToken, "full", []string{"a"}, model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
			})
		})

		t.Run("none", func(t *testing.T) {
			t.Run("admin", func(t *testing.T) {
				t.Run("r", func(t *testing.T) {
					access, err, _ := c.CheckPermission(client.InternalAdminToken, "full", "a", model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(client.InternalAdminToken, "full", []string{"a"}, model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("rw", func(t *testing.T) {
					access, err, _ := c.CheckPermission(client.InternalAdminToken, "full", "a", model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(client.InternalAdminToken, "full", []string{"a"}, model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("rx", func(t *testing.T) {
					access, err, _ := c.CheckPermission(client.InternalAdminToken, "full", "a", model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(client.InternalAdminToken, "full", []string{"a"}, model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("a", func(t *testing.T) {
					access, err, _ := c.CheckPermission(client.InternalAdminToken, "full", "a", model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(client.InternalAdminToken, "full", []string{"a"}, model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
			})

			t.Run("group", func(t *testing.T) {
				t.Run("r", func(t *testing.T) {
					access, err, _ := c.CheckPermission(GroupTestToken, "full", "a", model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(GroupTestToken, "full", []string{"a"}, model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("rw", func(t *testing.T) {
					access, err, _ := c.CheckPermission(GroupTestToken, "full", "a", model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(GroupTestToken, "full", []string{"a"}, model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("rx", func(t *testing.T) {
					access, err, _ := c.CheckPermission(GroupTestToken, "full", "a", model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(GroupTestToken, "full", []string{"a"}, model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("a", func(t *testing.T) {
					access, err, _ := c.CheckPermission(GroupTestToken, "full", "a", model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(GroupTestToken, "full", []string{"a"}, model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
			})

			t.Run("user", func(t *testing.T) {
				t.Run("r", func(t *testing.T) {
					access, err, _ := c.CheckPermission(TestToken, "full", "a", model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(TestToken, "full", []string{"a"}, model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("rw", func(t *testing.T) {
					access, err, _ := c.CheckPermission(TestToken, "full", "a", model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(TestToken, "full", []string{"a"}, model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("rx", func(t *testing.T) {
					access, err, _ := c.CheckPermission(TestToken, "full", "a", model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(TestToken, "full", []string{"a"}, model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
				t.Run("a", func(t *testing.T) {
					access, err, _ := c.CheckPermission(TestToken, "full", "a", model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if !access {
						t.Error("expected access")
						return
					}
					accessMap, err, _ := c.CheckMultiplePermissions(TestToken, "full", []string{"a"}, model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(accessMap, map[string]bool{"a": true}) {
						t.Error("expected access")
						return
					}
				})
			})
		})
	})
	t.Run("list", func(t *testing.T) {
		t.Run("partial", func(t *testing.T) {
			t.Run("admin", func(t *testing.T) {
				t.Run("admin ids", func(t *testing.T) {
					results, err, _ := c.ListResourcesWithAdminPermission(client.InternalAdminToken, "partial", model.ListOptions{})
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("r", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(client.InternalAdminToken, "partial", model.ListOptions{}, model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("rw", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(client.InternalAdminToken, "partial", model.ListOptions{}, model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("rx", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(client.InternalAdminToken, "partial", model.ListOptions{}, model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("a", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(client.InternalAdminToken, "partial", model.ListOptions{}, model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
			})
			t.Run("group", func(t *testing.T) {
				t.Run("admin ids", func(t *testing.T) {
					results, err, _ := c.ListResourcesWithAdminPermission(GroupTestToken, "partial", model.ListOptions{})
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 0 {
						t.Error("expected 0 result")
						return
					}
				})
				t.Run("r", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(GroupTestToken, "partial", model.ListOptions{}, model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("rw", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(GroupTestToken, "partial", model.ListOptions{}, model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 0 {
						t.Error("expected 0 result")
						return
					}
				})
				t.Run("rx", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(GroupTestToken, "partial", model.ListOptions{}, model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 0 {
						t.Error("expected 0 result")
						return
					}
				})
				t.Run("a", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(GroupTestToken, "partial", model.ListOptions{}, model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 0 {
						t.Error("expected 0 result")
						return
					}
				})
			})
			t.Run("user", func(t *testing.T) {
				t.Run("admin ids", func(t *testing.T) {
					results, err, _ := c.ListResourcesWithAdminPermission(TestToken, "partial", model.ListOptions{})
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 0 {
						t.Error("expected 0 result")
						return
					}
				})
				t.Run("r", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(TestToken, "partial", model.ListOptions{}, model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("rw", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(TestToken, "partial", model.ListOptions{}, model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 0 {
						t.Error("expected 0 result")
						return
					}
				})
				t.Run("rx", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(TestToken, "partial", model.ListOptions{}, model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("a", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(TestToken, "partial", model.ListOptions{}, model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 0 {
						t.Error("expected 0 result")
						return
					}
				})
			})
		})

		t.Run("full", func(t *testing.T) {
			t.Run("admin", func(t *testing.T) {
				t.Run("admin ids", func(t *testing.T) {
					results, err, _ := c.ListResourcesWithAdminPermission(client.InternalAdminToken, "full", model.ListOptions{})
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("r", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(client.InternalAdminToken, "full", model.ListOptions{}, model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("rw", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(client.InternalAdminToken, "full", model.ListOptions{}, model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("rx", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(client.InternalAdminToken, "full", model.ListOptions{}, model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("a", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(client.InternalAdminToken, "full", model.ListOptions{}, model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
			})
			t.Run("group", func(t *testing.T) {
				t.Run("admin ids", func(t *testing.T) {
					results, err, _ := c.ListResourcesWithAdminPermission(GroupTestToken, "full", model.ListOptions{})
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("r", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(GroupTestToken, "full", model.ListOptions{}, model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("rw", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(GroupTestToken, "full", model.ListOptions{}, model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("rx", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(GroupTestToken, "full", model.ListOptions{}, model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("a", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(GroupTestToken, "full", model.ListOptions{}, model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
			})
			t.Run("user", func(t *testing.T) {
				t.Run("admin ids", func(t *testing.T) {
					results, err, _ := c.ListResourcesWithAdminPermission(TestToken, "full", model.ListOptions{})
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("r", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(TestToken, "full", model.ListOptions{}, model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("rw", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(TestToken, "full", model.ListOptions{}, model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("rx", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(TestToken, "full", model.ListOptions{}, model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("a", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(TestToken, "full", model.ListOptions{}, model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
			})
		})

		t.Run("none", func(t *testing.T) {
			t.Run("admin", func(t *testing.T) {
				t.Run("admin ids", func(t *testing.T) {
					results, err, _ := c.ListResourcesWithAdminPermission(client.InternalAdminToken, "none", model.ListOptions{})
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("r", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(client.InternalAdminToken, "none", model.ListOptions{}, model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("rw", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(client.InternalAdminToken, "none", model.ListOptions{}, model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("rx", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(client.InternalAdminToken, "none", model.ListOptions{}, model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
				t.Run("a", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(client.InternalAdminToken, "none", model.ListOptions{}, model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 1 {
						t.Error("expected 1 result")
						return
					}
				})
			})
			t.Run("group", func(t *testing.T) {
				t.Run("admin ids", func(t *testing.T) {
					results, err, _ := c.ListResourcesWithAdminPermission(GroupTestToken, "none", model.ListOptions{})
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 0 {
						t.Error("expected 0 result")
						return
					}
				})
				t.Run("r", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(GroupTestToken, "none", model.ListOptions{}, model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 0 {
						t.Error("expected 0 result")
						return
					}
				})
				t.Run("rw", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(GroupTestToken, "none", model.ListOptions{}, model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 0 {
						t.Error("expected 0 result")
						return
					}
				})
				t.Run("rx", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(GroupTestToken, "none", model.ListOptions{}, model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 0 {
						t.Error("expected 0 result")
						return
					}
				})
				t.Run("a", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(GroupTestToken, "none", model.ListOptions{}, model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 0 {
						t.Error("expected 0 result")
						return
					}
				})
			})
			t.Run("user", func(t *testing.T) {
				t.Run("admin ids", func(t *testing.T) {
					results, err, _ := c.ListResourcesWithAdminPermission(TestToken, "none", model.ListOptions{})
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 0 {
						t.Error("expected 0 result")
						return
					}
				})
				t.Run("r", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(TestToken, "none", model.ListOptions{}, model.Read)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 0 {
						t.Error("expected 0 result")
						return
					}
				})
				t.Run("rw", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(TestToken, "none", model.ListOptions{}, model.Read, model.Write)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 0 {
						t.Error("expected 0 result")
						return
					}
				})
				t.Run("rx", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(TestToken, "none", model.ListOptions{}, model.Read, model.Execute)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 0 {
						t.Error("expected 0 result")
						return
					}
				})
				t.Run("a", func(t *testing.T) {
					results, err, _ := c.ListAccessibleResourceIds(TestToken, "none", model.ListOptions{}, model.Administrate)
					if err != nil {
						t.Error(err)
						return
					}
					if len(results) != 0 {
						t.Error("expected 0 result")
						return
					}
				})
			})
		})

	})
	t.Run("computed permissions", func(t *testing.T) {
		t.Run("partial", func(t *testing.T) {
			t.Run("admin", func(t *testing.T) {
				result, err, _ := c.ListComputedPermissions(client.InternalAdminToken, "partial", []string{"a"})
				if err != nil {
					t.Error(err)
					return
				}
				if !reflect.DeepEqual(result, []model.ComputedPermissions{{
					Id: "a",
					PermissionsMap: model.PermissionsMap{
						Read:         true,
						Write:        true,
						Execute:      true,
						Administrate: true,
					}},
				}) {
					t.Error(result)
					return
				}
			})
			t.Run("user", func(t *testing.T) {
				result, err, _ := c.ListComputedPermissions(TestToken, "partial", []string{"a"})
				if err != nil {
					t.Error(err)
					return
				}
				if !reflect.DeepEqual(result, []model.ComputedPermissions{{
					Id: "a",
					PermissionsMap: model.PermissionsMap{
						Read:         true,
						Write:        false,
						Execute:      true,
						Administrate: false,
					}},
				}) {
					t.Error(result)
					return
				}
			})

			t.Run("group", func(t *testing.T) {
				result, err, _ := c.ListComputedPermissions(GroupTestToken, "partial", []string{"a"})
				if err != nil {
					t.Error(err)
					return
				}
				if !reflect.DeepEqual(result, []model.ComputedPermissions{{
					Id: "a",
					PermissionsMap: model.PermissionsMap{
						Read:         true,
						Write:        false,
						Execute:      false,
						Administrate: false,
					}},
				}) {
					t.Error(result)
					return
				}
			})
		})
		t.Run("full", func(t *testing.T) {
			t.Run("admin", func(t *testing.T) {
				result, err, _ := c.ListComputedPermissions(client.InternalAdminToken, "full", []string{"a"})
				if err != nil {
					t.Error(err)
					return
				}
				if !reflect.DeepEqual(result, []model.ComputedPermissions{{
					Id: "a",
					PermissionsMap: model.PermissionsMap{
						Read:         true,
						Write:        true,
						Execute:      true,
						Administrate: true,
					}},
				}) {
					t.Error(result)
					return
				}
			})
			t.Run("user", func(t *testing.T) {
				result, err, _ := c.ListComputedPermissions(TestToken, "full", []string{"a"})
				if err != nil {
					t.Error(err)
					return
				}
				if !reflect.DeepEqual(result, []model.ComputedPermissions{{
					Id: "a",
					PermissionsMap: model.PermissionsMap{
						Read:         true,
						Write:        true,
						Execute:      true,
						Administrate: true,
					}},
				}) {
					t.Error(result)
					return
				}
			})

			t.Run("group", func(t *testing.T) {
				result, err, _ := c.ListComputedPermissions(GroupTestToken, "full", []string{"a"})
				if err != nil {
					t.Error(err)
					return
				}
				if !reflect.DeepEqual(result, []model.ComputedPermissions{{
					Id: "a",
					PermissionsMap: model.PermissionsMap{
						Read:         true,
						Write:        true,
						Execute:      true,
						Administrate: true,
					}},
				}) {
					t.Error(result)
					return
				}
			})
		})
		t.Run("none", func(t *testing.T) {
			t.Run("admin", func(t *testing.T) {
				result, err, _ := c.ListComputedPermissions(client.InternalAdminToken, "none", []string{"a"})
				if err != nil {
					t.Error(err)
					return
				}
				if !reflect.DeepEqual(result, []model.ComputedPermissions{{
					Id: "a",
					PermissionsMap: model.PermissionsMap{
						Read:         true,
						Write:        true,
						Execute:      true,
						Administrate: true,
					}},
				}) {
					t.Error(result)
					return
				}
			})
			t.Run("user", func(t *testing.T) {
				result, err, _ := c.ListComputedPermissions(TestToken, "none", []string{"a"})
				if err != nil {
					t.Error(err)
					return
				}
				if !reflect.DeepEqual(result, []model.ComputedPermissions{{
					Id: "a",
					PermissionsMap: model.PermissionsMap{
						Read:         false,
						Write:        false,
						Execute:      false,
						Administrate: false,
					}},
				}) {
					t.Error(result)
					return
				}
			})

			t.Run("group", func(t *testing.T) {
				result, err, _ := c.ListComputedPermissions(GroupTestToken, "none", []string{"a"})
				if err != nil {
					t.Error(err)
					return
				}
				if !reflect.DeepEqual(result, []model.ComputedPermissions{{
					Id: "a",
					PermissionsMap: model.PermissionsMap{
						Read:         false,
						Write:        false,
						Execute:      false,
						Administrate: false,
					}},
				}) {
					t.Error(result)
					return
				}
			})
		})
	})
	t.Run("update", func(t *testing.T) {
		t.Run("partial", func(t *testing.T) {
			t.Run("admin", func(t *testing.T) {
				_, err, _ = c.SetPermission(client.InternalAdminToken, "partial", "a", model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{"someone": {Read: true, Write: true, Execute: true, Administrate: true}},
				})
				if err != nil {
					t.Error(err)
					return
				}
			})
			t.Run("user", func(t *testing.T) {
				_, err, code := c.SetPermission(TestToken, "partial", "a", model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{"someone": {Read: true, Write: true, Execute: true, Administrate: true}},
				})
				if err == nil {
					t.Error("expected error")
					return
				}
				if code != http.StatusForbidden {
					t.Errorf("expected status code %d, got %d", http.StatusForbidden, code)
				}
			})
			t.Run("group", func(t *testing.T) {
				_, err, code := c.SetPermission(GroupTestToken, "partial", "a", model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{"someone": {Read: true, Write: true, Execute: true, Administrate: true}},
				})
				if err == nil {
					t.Error("expected error")
					return
				}
				if code != http.StatusForbidden {
					t.Errorf("expected status code %d, got %d", http.StatusForbidden, code)
				}
			})
		})
		t.Run("none", func(t *testing.T) {
			t.Run("admin", func(t *testing.T) {
				_, err, _ = c.SetPermission(client.InternalAdminToken, "none", "a", model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{"someone": {Read: true, Write: true, Execute: true, Administrate: true}},
				})
				if err != nil {
					t.Error(err)
					return
				}
			})
			t.Run("user", func(t *testing.T) {
				_, err, code := c.SetPermission(TestToken, "none", "a", model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{"someone": {Read: true, Write: true, Execute: true, Administrate: true}},
				})
				if err == nil {
					t.Error("expected error")
					return
				}
				if code != http.StatusForbidden {
					t.Errorf("expected status code %d, got %d", http.StatusForbidden, code)
				}
			})
			t.Run("group", func(t *testing.T) {
				_, err, code := c.SetPermission(GroupTestToken, "none", "a", model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{"someone": {Read: true, Write: true, Execute: true, Administrate: true}},
				})
				if err == nil {
					t.Error("expected error")
					return
				}
				if code != http.StatusForbidden {
					t.Errorf("expected status code %d, got %d", http.StatusForbidden, code)
				}
			})
		})
		t.Run("full", func(t *testing.T) {
			t.Run("admin", func(t *testing.T) {
				_, err, _ = c.SetPermission(client.InternalAdminToken, "full", "a", model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{"someone": {Read: true, Write: true, Execute: true, Administrate: true}},
				})
				if err != nil {
					t.Error(err)
					return
				}
			})
			t.Run("user", func(t *testing.T) {
				_, err, _ = c.SetPermission(TestToken, "full", "a", model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{"someone": {Read: true, Write: true, Execute: true, Administrate: true}},
				})
				if err != nil {
					t.Error(err)
					return
				}
			})
			t.Run("group", func(t *testing.T) {
				_, err, _ = c.SetPermission(GroupTestToken, "full", "a", model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{"someone": {Read: true, Write: true, Execute: true, Administrate: true}},
				})
				if err != nil {
					t.Error(err)
					return
				}
			})
		})
	})
	t.Run("delete", func(t *testing.T) {
		t.Run("partial", func(t *testing.T) {
			t.Run("user", func(t *testing.T) {
				err, code := c.RemoveResource(TestToken, "partial", "a")
				if err == nil {
					t.Error("expected error")
					return
				}
				if code != http.StatusForbidden {
					t.Errorf("expected status code %d, got %d", http.StatusForbidden, code)
				}
			})
			t.Run("group", func(t *testing.T) {
				err, code := c.RemoveResource(GroupTestToken, "partial", "a")
				if err == nil {
					t.Error("expected error")
					return
				}
				if code != http.StatusForbidden {
					t.Errorf("expected status code %d, got %d", http.StatusForbidden, code)
				}
			})
			t.Run("admin", func(t *testing.T) {
				err, _ = c.RemoveResource(client.InternalAdminToken, "partial", "a")
				if err != nil {
					t.Error(err)
					return
				}
			})
		})
		t.Run("none", func(t *testing.T) {
			t.Run("user", func(t *testing.T) {
				err, code := c.RemoveResource(TestToken, "none", "a")
				if err == nil {
					t.Error("expected error")
					return
				}
				if code != http.StatusForbidden {
					t.Errorf("expected status code %d, got %d", http.StatusForbidden, code)
				}
			})
			t.Run("group", func(t *testing.T) {
				err, code := c.RemoveResource(GroupTestToken, "none", "a")
				if err == nil {
					t.Error("expected error")
					return
				}
				if code != http.StatusForbidden {
					t.Errorf("expected status code %d, got %d", http.StatusForbidden, code)
				}
			})
			t.Run("admin", func(t *testing.T) {
				err, _ = c.RemoveResource(client.InternalAdminToken, "none", "a")
				if err != nil {
					t.Error(err)
					return
				}
			})
		})
		t.Run("full", func(t *testing.T) {
			t.Run("group", func(t *testing.T) {
				err, _ = c.RemoveResource(GroupTestToken, "full", "a")
				if err == nil {
					t.Error("expected error")
					return
				}
			})
			t.Run("user", func(t *testing.T) {
				err, _ = c.RemoveResource(TestToken, "full", "a")
				if err == nil {
					t.Error("expected error")
					return
				}
			})
			t.Run("admin", func(t *testing.T) {
				err, _ = c.RemoveResource(client.InternalAdminToken, "full", "a")
				if err != nil {
					t.Error(err)
					return
				}
			})
		})
	})
}
