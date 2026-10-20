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
	"reflect"
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

func TestRestrictedGroupGranters(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const protected = "/shared-resources"
	owner := map[string]model.PermissionsMap{"owner": {Read: true, Write: true, Execute: true, Administrate: true}}
	stored := map[string]model.PermissionsMap{
		protected:     {Read: true, Execute: true},
		"other-group": {Read: true},
	}
	with := func(changes map[string]*model.PermissionsMap) map[string]model.PermissionsMap {
		result := map[string]model.PermissionsMap{}
		for k, v := range stored {
			result[k] = v
		}
		for k, v := range changes {
			if v == nil {
				delete(result, k)
			} else {
				result[k] = *v
			}
		}
		return result
	}

	newCtrl := func(t *testing.T, config configuration.Config) *Controller {
		db := mock.New()
		for id, groups := range map[string]map[string]model.PermissionsMap{
			"with-protected":    stored,
			"without-protected": {"other-group": {Read: true}},
		} {
			err := db.SetResource(ctx, model.Resource{Id: id, TopicId: "topic", ResourcePermissions: model.ResourcePermissions{UserPermissions: owner, GroupPermissions: groups}}, time.Now(), true)
			if err != nil {
				t.Fatal(err)
			}
		}
		ctrl, err := NewWithDependencies(ctx, config, db, kafka.NewVoidProducerProvider())
		if err != nil {
			t.Fatal(err)
		}
		return ctrl
	}

	member := jwt.Token{Sub: "member", Groups: []string{protected, "other-group"}, RealmAccess: map[string][]string{"roles": {"user"}}}
	granter := jwt.Token{Sub: "granter", Groups: []string{protected, "other-group"}, RealmAccess: map[string][]string{"roles": {"user"}}}
	otherGranter := jwt.Token{Sub: "other-granter", Groups: []string{protected, "other-group"}, RealmAccess: map[string][]string{"roles": {"user"}}}
	noSub := jwt.Token{Sub: "", Groups: []string{protected, "other-group"}, RealmAccess: map[string][]string{"roles": {"user"}}}
	admin := jwt.Token{Sub: "admin-user", RealmAccess: map[string][]string{"roles": {"admin"}}}

	type testCase struct {
		name   string
		token  jwt.Token
		id     string
		groups map[string]model.PermissionsMap
		want   int
	}
	run := func(t *testing.T, ctrl *Controller, cases []testCase) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				_, err, code := ctrl.checkEditPermission(c.token, "topic", c.id, model.ResourcePermissions{UserPermissions: owner, GroupPermissions: c.groups})
				if code != c.want {
					t.Errorf("code = %v, want %v (err: %v)", code, c.want, err)
				}
				if (err == nil) != (c.want == http.StatusOK) {
					t.Errorf("err = %v with code %v", err, code)
				}
			})
		}
	}

	t.Run("restricted", func(t *testing.T) {
		ctrl := newCtrl(t, configuration.Config{RestrictedGroupGranters: configuration.GroupGranters{Granters: map[string][]string{
			protected:       {"granter"},
			"/other-locked": {"other-granter"},
			"/admins-only":  {},
		}}})
		changed := model.PermissionsMap{Read: true, Write: true, Execute: true}
		otherChanged := model.PermissionsMap{Read: true, Write: true}
		run(t, ctrl, []testCase{
			{"member adds protected group", member, "without-protected", map[string]model.PermissionsMap{"other-group": {Read: true}, protected: {Read: true}}, http.StatusForbidden},
			{"member adds protected group to new resource", member, "new-resource", map[string]model.PermissionsMap{protected: {Read: true}}, http.StatusForbidden},
			{"member changes protected entry", member, "with-protected", with(map[string]*model.PermissionsMap{protected: &changed}), http.StatusForbidden},
			{"member removes protected entry", member, "with-protected", with(map[string]*model.PermissionsMap{protected: nil}), http.StatusForbidden},
			{"member sends nil group permissions", member, "with-protected", nil, http.StatusForbidden},
			{"member sets protected entry all false", member, "with-protected", with(map[string]*model.PermissionsMap{protected: {}}), http.StatusForbidden},
			{"member removes protected entry via modified id", member, "with-protected$foo=bar", with(map[string]*model.PermissionsMap{protected: nil}), http.StatusForbidden},
			{"user without id removes protected entry", noSub, "with-protected", with(map[string]*model.PermissionsMap{protected: nil}), http.StatusForbidden},
			{"granter of another group changes protected entry", otherGranter, "with-protected", with(map[string]*model.PermissionsMap{protected: &changed}), http.StatusForbidden},
			{"granter adds admins-only group", granter, "with-protected", with(map[string]*model.PermissionsMap{"/admins-only": {Read: true}}), http.StatusForbidden},

			{"member changes only other entries", member, "with-protected", with(map[string]*model.PermissionsMap{"other-group": &otherChanged}), http.StatusOK},
			{"member removes other entry", member, "with-protected", with(map[string]*model.PermissionsMap{"other-group": nil}), http.StatusOK},
			{"member keeps protected entry via modified id", member, "with-protected$foo=bar", stored, http.StatusOK},
			{"member sends all-false entry for absent protected group", member, "without-protected", map[string]model.PermissionsMap{"other-group": {Read: true}, protected: {}}, http.StatusOK},
			{"member sends nil group permissions without protected group", member, "without-protected", nil, http.StatusOK},
			{"granter adds protected group", granter, "without-protected", map[string]model.PermissionsMap{"other-group": {Read: true}, protected: {Read: true}}, http.StatusOK},
			{"granter changes protected entry", granter, "with-protected", with(map[string]*model.PermissionsMap{protected: &changed}), http.StatusOK},
			{"granter removes protected entry", granter, "with-protected", with(map[string]*model.PermissionsMap{protected: nil}), http.StatusOK},
			{"admin changes protected entry", admin, "with-protected", with(map[string]*model.PermissionsMap{protected: &changed}), http.StatusOK},
			{"admin removes protected entry", admin, "with-protected", nil, http.StatusOK},
			{"admin adds admins-only group", admin, "with-protected", with(map[string]*model.PermissionsMap{"/admins-only": {Read: true}}), http.StatusOK},
		})
	})

	t.Run("feature off", func(t *testing.T) {
		ctrl := newCtrl(t, configuration.Config{})
		changed := model.PermissionsMap{Read: true, Write: true, Execute: true}
		run(t, ctrl, []testCase{
			{"member adds protected group", member, "without-protected", map[string]model.PermissionsMap{"other-group": {Read: true}, protected: {Read: true}}, http.StatusOK},
			{"member changes protected entry", member, "with-protected", with(map[string]*model.PermissionsMap{protected: &changed}), http.StatusOK},
			{"member removes protected entry", member, "with-protected", with(map[string]*model.PermissionsMap{protected: nil}), http.StatusOK},
			{"non-member still may not add a group", jwt.Token{Sub: "outsider", RealmAccess: map[string][]string{"roles": {"user"}}}, "without-protected", map[string]model.PermissionsMap{protected: {Read: true}}, http.StatusBadRequest},
		})
	})
}

func TestCheckEditPermissionReadsStoredResourceForModifiedId(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db := mock.New()
	roles := map[string]model.PermissionsMap{"role": {Read: true}}
	owner := map[string]model.PermissionsMap{"owner": {Read: true, Write: true, Execute: true, Administrate: true}}
	err := db.SetResource(ctx, model.Resource{Id: "resource", TopicId: "topic", ResourcePermissions: model.ResourcePermissions{UserPermissions: owner, RolePermissions: roles}}, time.Now(), true)
	if err != nil {
		t.Fatal(err)
	}
	ctrl, err := NewWithDependencies(ctx, configuration.Config{OnlyAdminsMayEditRolePermissions: true}, db, kafka.NewVoidProducerProvider())
	if err != nil {
		t.Fatal(err)
	}
	result, err, _ := ctrl.checkEditPermission(jwt.Token{Sub: "owner"}, "topic", "resource$foo=bar", model.ResourcePermissions{UserPermissions: owner})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.RolePermissions, roles) {
		t.Errorf("missing role permissions must be taken from the stored resource, got %#v", result.RolePermissions)
	}
	_, err, code := ctrl.checkEditPermission(jwt.Token{Sub: "owner"}, "topic", "resource$foo=bar", model.ResourcePermissions{UserPermissions: owner, RolePermissions: map[string]model.PermissionsMap{}})
	if code != http.StatusForbidden {
		t.Errorf("removing role permissions via a modified id: code = %v, err = %v", code, err)
	}
}
