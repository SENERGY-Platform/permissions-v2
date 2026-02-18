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

package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller/kafka"
	"github.com/SENERGY-Platform/permissions-v2/pkg/database/mock"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
)

func TestCheckGroupMembership(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("no user-management request", func(t *testing.T) {
		db := mock.New()
		producer := kafka.NewVoidProducerProvider()
		ctrl, err := NewWithDependencies(ctx, configuration.Config{OnlyAdminsMayEditRolePermissions: true}, db, producer)
		if err != nil {
			t.Error(err)
			return
		}

		initialResourcePermissions := model.ResourcePermissions{
			UserPermissions: map[string]model.PermissionsMap{
				"existing-user-1": {
					Read:         true,
					Write:        true,
					Execute:      true,
					Administrate: true,
				},
				"existing-user-2": {
					Read:         true,
					Write:        false,
					Execute:      false,
					Administrate: false,
				},
			},
			GroupPermissions: map[string]model.PermissionsMap{
				"existing-group-1": {
					Read:         true,
					Write:        true,
					Execute:      true,
					Administrate: true,
				},
				"existing-group-2": {
					Read:         true,
					Write:        false,
					Execute:      false,
					Administrate: false,
				},
			},
		}

		t.Run("init existing resource", func(t *testing.T) {
			err = db.SetResource(ctrl.getTimeoutContext(), model.Resource{
				Id:                  "test-resource-1",
				TopicId:             "topic",
				ResourcePermissions: initialResourcePermissions,
			}, time.Now(), true)
			if err != nil {
				t.Error(err)
				return
			}
		})

		t.Run("admin may edit role perm", func(t *testing.T) {
			_, err, _ := ctrl.checkEditPermission(jwt.Token{
				Sub:         "requesting-user",
				Groups:      []string{},
				RealmAccess: map[string][]string{"roles": {"admin"}},
			}, "topic", "test-resource-1", model.ResourcePermissions{
				RolePermissions:  map[string]model.PermissionsMap{"testrole": {Read: true, Write: true, Execute: true, Administrate: true}},
				UserPermissions:  initialResourcePermissions.UserPermissions,
				GroupPermissions: initialResourcePermissions.GroupPermissions,
			})
			if err != nil {
				t.Error(err)
				return
			}
		})

		t.Run("none admin may not edit role perm", func(t *testing.T) {
			_, err, _ := ctrl.checkEditPermission(jwt.Token{
				Sub:         "requesting-user",
				Groups:      []string{},
				RealmAccess: map[string][]string{"roles": {"user"}},
			}, "topic", "test-resource-1", model.ResourcePermissions{
				RolePermissions:  map[string]model.PermissionsMap{"testrole": {Read: true, Write: true, Execute: true, Administrate: true}},
				UserPermissions:  initialResourcePermissions.UserPermissions,
				GroupPermissions: initialResourcePermissions.GroupPermissions,
			})
			if err == nil {
				t.Error("expected error")
				return
			}
		})

		t.Run("unchanged is ok", func(t *testing.T) {
			_, err, _ := ctrl.checkEditPermission(jwt.Token{
				Sub:    "requesting-user",
				Groups: []string{},
			}, "topic", "test-resource-1", model.ResourcePermissions{})
			if err != nil {
				t.Error(err)
				return
			}
		})

		t.Run("remove is ok", func(t *testing.T) {
			_, err, _ := ctrl.checkEditPermission(jwt.Token{
				Sub:    "requesting-user",
				Groups: []string{},
			}, "topic", "test-resource-1", model.ResourcePermissions{})
			if err != nil {
				t.Error(err)
				return
			}
		})

		t.Run("change is ok", func(t *testing.T) {
			initialResourcePermissions.UserPermissions["existing-user-2"] = model.PermissionsMap{
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			}
			initialResourcePermissions.GroupPermissions["existing-group-2"] = model.PermissionsMap{
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			}
			_, err, _ := ctrl.checkEditPermission(jwt.Token{
				Sub:    "requesting-user",
				Groups: []string{},
			}, "topic", "test-resource-1", initialResourcePermissions)
			if err != nil {
				t.Error(err)
				return
			}
		})

		t.Run("adding group is ok if in same group", func(t *testing.T) {
			_, err, _ := ctrl.checkEditPermission(jwt.Token{
				Sub:    "requesting-user",
				Groups: []string{"new-group-1"},
			}, "topic", "test-resource-1", model.ResourcePermissions{GroupPermissions: map[string]model.PermissionsMap{
				"new-group-1": {
					Read:         true,
					Write:        true,
					Execute:      true,
					Administrate: true,
				},
			}})
			if err != nil {
				t.Error(err)
				return
			}
		})

		t.Run("adding group fails if not in same group", func(t *testing.T) {
			_, err, _ := ctrl.checkEditPermission(jwt.Token{
				Sub:    "requesting-user",
				Groups: []string{"new-group-2"},
			}, "topic", "test-resource-1", model.ResourcePermissions{GroupPermissions: map[string]model.PermissionsMap{
				"new-group-1": {
					Read:         true,
					Write:        true,
					Execute:      true,
					Administrate: true,
				},
			}})
			if err == nil {
				t.Error("expected error", err)
				return
			}
		})

		t.Run("new resource will be checked", func(t *testing.T) {
			t.Run("adding group is ok if in same group", func(t *testing.T) {
				_, err, _ := ctrl.checkEditPermission(jwt.Token{
					Sub:    "requesting-user",
					Groups: []string{"new-group-1"},
				}, "topic", "test-resource-2", model.ResourcePermissions{GroupPermissions: map[string]model.PermissionsMap{
					"new-group-1": {
						Read:         true,
						Write:        true,
						Execute:      true,
						Administrate: true,
					},
				}})
				if err != nil {
					t.Error(err)
					return
				}
			})

			t.Run("adding group fails if not in same group", func(t *testing.T) {
				_, err, _ := ctrl.checkEditPermission(jwt.Token{
					Sub:    "requesting-user",
					Groups: []string{"new-group-2"},
				}, "topic", "test-resource-2", model.ResourcePermissions{GroupPermissions: map[string]model.PermissionsMap{
					"new-group-1": {
						Read:         true,
						Write:        true,
						Execute:      true,
						Administrate: true,
					},
				}})
				if err == nil {
					t.Error("expected error", err)
					return
				}
			})
		})
	})

	t.Run("with user-management request", func(t *testing.T) {
		db := mock.New()
		producer := kafka.NewVoidProducerProvider()

		called := 0

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == "/user-list" {
				if r.Header.Get("Authorization") != "testtoken" {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				json.NewEncoder(w).Encode([]User{
					{
						Id:   "new-user-in-group-1",
						Name: "foo",
					}, {
						Id:   "requesting-user",
						Name: "bar",
					},
				})
				called = called + 1
			}
		}))
		go func() {
			<-ctx.Done()
			server.Close()
		}()

		ctrl, err := NewWithDependencies(ctx, configuration.Config{UserManagementUrl: server.URL}, db, producer)
		if err != nil {
			t.Error(err)
			return
		}

		initialResourcePermissions := model.ResourcePermissions{
			UserPermissions: map[string]model.PermissionsMap{
				"existing-user-1": {
					Read:         true,
					Write:        true,
					Execute:      true,
					Administrate: true,
				},
				"existing-user-2": {
					Read:         true,
					Write:        false,
					Execute:      false,
					Administrate: false,
				},
			},
			GroupPermissions: map[string]model.PermissionsMap{
				"existing-group-1": {
					Read:         true,
					Write:        true,
					Execute:      true,
					Administrate: true,
				},
				"existing-group-2": {
					Read:         true,
					Write:        false,
					Execute:      false,
					Administrate: false,
				},
			},
		}

		t.Run("init existing resource", func(t *testing.T) {
			err = db.SetResource(ctrl.getTimeoutContext(), model.Resource{
				Id:                  "test-resource-1",
				TopicId:             "topic",
				ResourcePermissions: initialResourcePermissions,
			}, time.Now(), true)
			if err != nil {
				t.Error(err)
				return
			}
		})

		t.Run("adding user is ok if in same group", func(t *testing.T) {
			_, err, _ := ctrl.checkEditPermission(jwt.Token{
				Token:  "testtoken",
				Sub:    "requesting-user",
				Groups: []string{"group-1"},
			}, "topic", "test-resource-1", model.ResourcePermissions{UserPermissions: map[string]model.PermissionsMap{
				"new-user-in-group-1": {
					Read:         true,
					Write:        true,
					Execute:      true,
					Administrate: true,
				},
			}})
			if err != nil {
				t.Error(err)
				return
			}
		})

		t.Run("adding user fails if not in same group", func(t *testing.T) {
			_, err, _ := ctrl.checkEditPermission(jwt.Token{
				Token:  "testtoken",
				Sub:    "requesting-user",
				Groups: []string{"group-1"},
			}, "topic", "test-resource-1", model.ResourcePermissions{UserPermissions: map[string]model.PermissionsMap{
				"new-user-in-group-2": {
					Read:         true,
					Write:        true,
					Execute:      true,
					Administrate: true,
				},
			}})
			if err == nil {
				t.Error("expected error", err)
				return
			}
		})

		t.Run("new resource will be checked", func(t *testing.T) {
			t.Run("adding user is ok if in same group", func(t *testing.T) {
				_, err, _ := ctrl.checkEditPermission(jwt.Token{
					Token:  "testtoken",
					Sub:    "requesting-user",
					Groups: []string{"group-1"},
				}, "topic", "test-resource-2", model.ResourcePermissions{UserPermissions: map[string]model.PermissionsMap{
					"new-user-in-group-1": {
						Read:         true,
						Write:        true,
						Execute:      true,
						Administrate: true,
					},
				}})
				if err != nil {
					t.Error(err)
					return
				}
			})

			t.Run("adding user fails if not in same group", func(t *testing.T) {
				_, err, _ := ctrl.checkEditPermission(jwt.Token{
					Token:  "testtoken",
					Sub:    "requesting-user",
					Groups: []string{"group-1"},
				}, "topic", "test-resource-2", model.ResourcePermissions{UserPermissions: map[string]model.PermissionsMap{
					"new-user-in-group-2": {
						Read:         true,
						Write:        true,
						Execute:      true,
						Administrate: true,
					},
				}})
				if err == nil {
					t.Error("expected error", err)
					return
				}
			})
		})

		t.Run("check user service called", func(t *testing.T) {
			if called != 4 {
				t.Error(called)
			}
		})
	})
}
