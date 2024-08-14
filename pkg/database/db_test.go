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

package database

import (
	"context"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/permissions-v2/pkg/tests/docker"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestSyncMark(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	port, _, err := docker.MongoDB(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.MongoUrl = "mongodb://localhost:" + port

	db, err := New(config)
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("init", func(t *testing.T) {
		timeNow := time.Now()
		timeOld := timeNow.Add(-1 * config.SyncAgeLimit.GetDuration()).Add(-1 * time.Minute)

		err = db.SetResource(nil, model.Resource{
			Id:      "a1",
			TopicId: "topic",
		}, timeNow, true)
		if err != nil {
			t.Error(err)
			return
		}
		err = db.SetResource(nil, model.Resource{
			Id:      "a2",
			TopicId: "topic",
		}, timeNow, true)
		if err != nil {
			t.Error(err)
			return
		}
		err = db.SetResource(nil, model.Resource{
			Id:      "b1",
			TopicId: "topic",
		}, timeOld, false)
		if err != nil {
			t.Error(err)
			return
		}
		err = db.SetResource(nil, model.Resource{
			Id:      "b2",
			TopicId: "topic",
		}, timeOld, false)
		if err != nil {
			t.Error(err)
			return
		}
		err = db.SetResource(nil, model.Resource{
			Id:      "b3",
			TopicId: "topic",
		}, timeOld, false)
		if err != nil {
			t.Error(err)
			return
		}

		err = db.SetResource(nil, model.Resource{
			Id:      "c1",
			TopicId: "topic",
		}, timeNow, false)
		if err != nil {
			t.Error(err)
			return
		}
		err = db.SetResource(nil, model.Resource{
			Id:      "c2",
			TopicId: "topic",
		}, timeNow, false)
	})

	t.Run("check unsynced list afer init", func(t *testing.T) {
		expected := []model.Resource{
			{
				Id:      "b1",
				TopicId: "topic",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{},
					RolePermissions:  map[string]model.PermissionsMap{},
				},
			},
			{
				Id:      "b2",
				TopicId: "topic",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{},
					RolePermissions:  map[string]model.PermissionsMap{},
				},
			},
			{
				Id:      "b3",
				TopicId: "topic",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{},
					RolePermissions:  map[string]model.PermissionsMap{},
				},
			},
		}
		list, err := db.ListUnsyncedResources(nil)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(list, expected) {
			t.Errorf("\n%#v\n%#v\n", list, expected)
			return
		}
	})

	t.Run("mark as synced", func(t *testing.T) {
		err = db.MarkResourceAsSynced(nil, "topic", "b2")
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("check unsynced list after mark", func(t *testing.T) {
		expected := []model.Resource{
			{
				Id:      "b1",
				TopicId: "topic",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{},
					RolePermissions:  map[string]model.PermissionsMap{},
				},
			},
			{
				Id:      "b3",
				TopicId: "topic",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{},
					RolePermissions:  map[string]model.PermissionsMap{},
				},
			},
		}
		list, err := db.ListUnsyncedResources(nil)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(list, expected) {
			t.Errorf("\n%#v\n%#v\n", list, expected)
			return
		}
	})

}

func TestResourcePermissionsWithRoles(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	port, _, err := docker.MongoDB(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.MongoUrl = "mongodb://localhost:" + port

	db, err := New(config)
	if err != nil {
		t.Error(err)
		return
	}

	updates := []struct {
		r model.Resource
		t time.Time
	}{
		{
			r: model.Resource{
				Id:      "a",
				TopicId: "device",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u2": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					RolePermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g2": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t: getTestTime(0),
		},
		{
			r: model.Resource{
				Id:      "a",
				TopicId: "device",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					RolePermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t: getTestTime(100),
		},

		{
			r: model.Resource{
				Id:      "b",
				TopicId: "device",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u2": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					RolePermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g2": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t: getTestTime(0),
		},
		{
			r: model.Resource{
				Id:      "b",
				TopicId: "device",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					RolePermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t: getTestTime(100),
		},
		{
			r: model.Resource{
				Id:      "b",
				TopicId: "device",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u2": {Read: true, Write: true, Execute: true, Administrate: true},
						"u3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					RolePermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t: getTestTime(50),
		},
		{
			r: model.Resource{
				Id:      "c",
				TopicId: "device",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u2": {Read: true, Write: true, Execute: true, Administrate: true},
						"u3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					RolePermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t: getTestTime(50),
		},
		{
			r: model.Resource{
				Id:      "d",
				TopicId: "device",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{
						"u2": {Read: true, Write: true, Execute: true, Administrate: true},
						"u3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					RolePermissions: map[string]model.PermissionsMap{
						"g3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t: getTestTime(50),
		},
	}

	t.Run("set", func(t *testing.T) {
		for i, update := range updates {
			err := db.SetResource(nil, update.r, update.t, true)
			if err != nil {
				t.Error(i, err)
				return
			}
		}
	})

	listQueries := []struct {
		topic          string
		user           string
		roles          string
		rights         model.PermissionList
		options        model.ListOptions
		expectedResult []model.Resource
	}{
		{
			topic:  "device",
			user:   "u1",
			rights: model.PermissionList{model.Read},
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:  "device",
			user:   "u2",
			rights: model.PermissionList{model.Read},
			expectedResult: []model.Resource{
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "d",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:  "device",
			user:   "u3",
			rights: model.PermissionList{model.Read},
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "d",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:   "device",
			user:    "u3",
			rights:  model.PermissionList{model.Read},
			options: model.ListOptions{Limit: 2, Offset: 1},
			expectedResult: []model.Resource{
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:  "device",
			roles:  "g1",
			rights: model.PermissionList{model.Read},
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:          "device",
			roles:          "g2",
			rights:         model.PermissionList{model.Read},
			expectedResult: []model.Resource{},
		},
		{
			topic:  "device",
			roles:  "g3",
			rights: model.PermissionList{model.Read},
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "d",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},

		{
			topic:  "device",
			user:   "u1",
			rights: model.PermissionList{model.Administrate},
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:  "device",
			user:   "u2",
			rights: model.PermissionList{model.Administrate},
			expectedResult: []model.Resource{
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "d",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:          "device",
			user:           "u3",
			rights:         model.PermissionList{model.Administrate},
			expectedResult: []model.Resource{},
		},
		{
			topic:  "device",
			roles:  "g1",
			rights: model.PermissionList{model.Administrate},
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:          "device",
			roles:          "g2",
			rights:         model.PermissionList{model.Administrate},
			expectedResult: []model.Resource{},
		},
		{
			topic:          "device",
			roles:          "g3",
			rights:         model.PermissionList{model.Administrate},
			expectedResult: []model.Resource{},
		},

		{
			topic:  "device",
			user:   "u1",
			roles:  "g3",
			rights: model.PermissionList{model.Read},
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "d",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:  "device",
			user:   "u1",
			roles:  "g3",
			rights: model.PermissionList{model.Read, model.Administrate},
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
	}

	t.Run("list", func(t *testing.T) {
		for i, q := range listQueries {
			result, err := db.ListResourcesByPermissions(nil, q.topic, q.user, []string{q.roles}, []string{}, q.options, q.rights...)
			if err != nil {
				t.Error(i, err)
				return
			}
			if !reflect.DeepEqual(result, q.expectedResult) {
				t.Errorf("%v ListResourcesByPermissions(topic=%#v,user=%#v,roles=%#v,rights=%#v,options=%#v) != expected\n%#v\n%#v\n", i, q.topic, q.user, q.roles, q.rights, q.options, result, q.expectedResult)
				return
			}

			expectedIds := []string{}
			for _, e := range q.expectedResult {
				expectedIds = append(expectedIds, e.Id)
			}
			actualIds, err := db.ListResourceIdsByPermissions(nil, q.topic, q.user, []string{q.roles}, []string{}, q.options, q.rights...)
			if err != nil {
				t.Error(i, err)
				return
			}
			if !reflect.DeepEqual(actualIds, expectedIds) {
				t.Errorf("%v ListResourceIdsByPermissions(topic=%#v,user=%#v,roles=%#v,rights=%#v,options=%#v) != expected\n%#v\n%#v\n", i, q.topic, q.user, q.roles, q.rights, q.options, actualIds, expectedIds)
				return
			}
		}
	})

	t.Run("check", func(t *testing.T) {
		result, err := db.CheckResourcePermissions(nil, "device", "a", "u1", []string{}, []string{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if !result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "d", "u1", []string{}, []string{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "d", "u1", []string{"g1"}, []string{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "d", "u1", []string{"g1"}, []string{}, model.Read)
		if err != nil {
			t.Error(err)
			return
		}
		if result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "d", "u1", []string{"g1", "g3"}, []string{}, model.Read)
		if err != nil {
			t.Error(err)
			return
		}
		if !result {
			t.Errorf("%#v", result)
			return
		}
	})

	t.Run("check multiple", func(t *testing.T) {
		result, err := db.CheckMultipleResourcePermissions(nil, "device", []string{"a", "b", "c", "d", "e", "x", "y"}, "u1", []string{}, []string{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(result, map[string]bool{
			"a": true,
			"b": true,
			"c": true,
			"d": false,
		}) {
			t.Errorf("%#v", result)
			return
		}
	})
}

func TestResourcePermissionsWithGroups(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	port, _, err := docker.MongoDB(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.MongoUrl = "mongodb://localhost:" + port

	db, err := New(config)
	if err != nil {
		t.Error(err)
		return
	}

	updates := []struct {
		r model.Resource
		t time.Time
	}{
		{
			r: model.Resource{
				Id:      "a",
				TopicId: "device",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u2": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					RolePermissions: map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g2": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t: getTestTime(0),
		},
		{
			r: model.Resource{
				Id:      "a",
				TopicId: "device",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					RolePermissions: map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t: getTestTime(100),
		},

		{
			r: model.Resource{
				Id:      "b",
				TopicId: "device",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u2": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					RolePermissions: map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g2": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t: getTestTime(0),
		},
		{
			r: model.Resource{
				Id:      "b",
				TopicId: "device",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					RolePermissions: map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t: getTestTime(100),
		},
		{
			r: model.Resource{
				Id:      "b",
				TopicId: "device",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u2": {Read: true, Write: true, Execute: true, Administrate: true},
						"u3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					RolePermissions: map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t: getTestTime(50),
		},
		{
			r: model.Resource{
				Id:      "c",
				TopicId: "device",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u2": {Read: true, Write: true, Execute: true, Administrate: true},
						"u3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					RolePermissions: map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t: getTestTime(50),
		},
		{
			r: model.Resource{
				Id:      "d",
				TopicId: "device",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{
						"u2": {Read: true, Write: true, Execute: true, Administrate: true},
						"u3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					RolePermissions: map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{
						"g3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t: getTestTime(50),
		},
	}

	t.Run("set", func(t *testing.T) {
		for i, update := range updates {
			err := db.SetResource(nil, update.r, update.t, true)
			if err != nil {
				t.Error(i, err)
				return
			}
		}
	})

	listQueries := []struct {
		topic          string
		user           string
		groups         string
		rights         model.PermissionList
		options        model.ListOptions
		expectedResult []model.Resource
	}{
		{
			topic:  "device",
			user:   "u1",
			rights: model.PermissionList{model.Read},
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:  "device",
			user:   "u2",
			rights: model.PermissionList{model.Read},
			expectedResult: []model.Resource{
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "d",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:  "device",
			user:   "u3",
			rights: model.PermissionList{model.Read},
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "d",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:   "device",
			user:    "u3",
			rights:  model.PermissionList{model.Read},
			options: model.ListOptions{Limit: 2, Offset: 1},
			expectedResult: []model.Resource{
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:  "device",
			groups: "g1",
			rights: model.PermissionList{model.Read},
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:          "device",
			groups:         "g2",
			rights:         model.PermissionList{model.Read},
			expectedResult: []model.Resource{},
		},
		{
			topic:  "device",
			groups: "g3",
			rights: model.PermissionList{model.Read},
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "d",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},

		{
			topic:  "device",
			user:   "u1",
			rights: model.PermissionList{model.Administrate},
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:  "device",
			user:   "u2",
			rights: model.PermissionList{model.Administrate},
			expectedResult: []model.Resource{
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "d",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:          "device",
			user:           "u3",
			rights:         model.PermissionList{model.Administrate},
			expectedResult: []model.Resource{},
		},
		{
			topic:  "device",
			groups: "g1",
			rights: model.PermissionList{model.Administrate},
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:          "device",
			groups:         "g2",
			rights:         model.PermissionList{model.Administrate},
			expectedResult: []model.Resource{},
		},
		{
			topic:          "device",
			groups:         "g3",
			rights:         model.PermissionList{model.Administrate},
			expectedResult: []model.Resource{},
		},

		{
			topic:  "device",
			user:   "u1",
			groups: "g3",
			rights: model.PermissionList{model.Read},
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "d",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:  "device",
			user:   "u1",
			groups: "g3",
			rights: model.PermissionList{model.Read, model.Administrate},
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						RolePermissions: map[string]model.PermissionsMap{},
						GroupPermissions: map[string]model.PermissionsMap{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
	}

	t.Run("list", func(t *testing.T) {
		for i, q := range listQueries {
			result, err := db.ListResourcesByPermissions(nil, q.topic, q.user, []string{}, []string{q.groups}, q.options, q.rights...)
			if err != nil {
				t.Error(i, err)
				return
			}
			if !reflect.DeepEqual(result, q.expectedResult) {
				t.Errorf("%v ListResourcesByPermissions(topic=%#v,user=%#v,roles=%#v,rights=%#v,options=%#v) != expected\n%#v\n%#v\n", i, q.topic, q.user, q.groups, q.rights, q.options, result, q.expectedResult)
				return
			}

			expectedIds := []string{}
			for _, e := range q.expectedResult {
				expectedIds = append(expectedIds, e.Id)
			}
			actualIds, err := db.ListResourceIdsByPermissions(nil, q.topic, q.user, []string{}, []string{q.groups}, q.options, q.rights...)
			if err != nil {
				t.Error(i, err)
				return
			}
			if !reflect.DeepEqual(actualIds, expectedIds) {
				t.Errorf("%v ListResourceIdsByPermissions(topic=%#v,user=%#v,roles=%#v,rights=%#v,options=%#v) != expected\n%#v\n%#v\n", i, q.topic, q.user, q.groups, q.rights, q.options, actualIds, expectedIds)
				return
			}
		}
	})

	t.Run("check", func(t *testing.T) {
		result, err := db.CheckResourcePermissions(nil, "device", "a", "u1", []string{}, []string{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if !result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "d", "u1", []string{}, []string{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "d", "u1", []string{}, []string{"g1"}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "d", "u1", []string{}, []string{"g1"}, model.Read)
		if err != nil {
			t.Error(err)
			return
		}
		if result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "d", "u1", []string{}, []string{"g1", "g3"}, model.Read)
		if err != nil {
			t.Error(err)
			return
		}
		if !result {
			t.Errorf("%#v", result)
			return
		}
	})

	t.Run("check multiple", func(t *testing.T) {
		result, err := db.CheckMultipleResourcePermissions(nil, "device", []string{"a", "b", "c", "d", "e", "x", "y"}, "u1", []string{}, []string{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(result, map[string]bool{
			"a": true,
			"b": true,
			"c": true,
			"d": false,
		}) {
			t.Errorf("%#v", result)
			return
		}
	})
}

func TestDistributedRightsRoles(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	port, _, err := docker.MongoDB(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.MongoUrl = "mongodb://localhost:" + port

	db, err := New(config)
	if err != nil {
		t.Error(err)
		return
	}

	err = db.SetResource(nil, model.Resource{
		Id:      "a",
		TopicId: "device",
		ResourcePermissions: model.ResourcePermissions{
			UserPermissions: map[string]model.PermissionsMap{"u1": {Read: true}},
			RolePermissions: map[string]model.PermissionsMap{"g1": {Write: true}, "g2": {Administrate: true}, "g3": {Execute: true}},
		},
	}, getTestTime(0), true)

	if err != nil {
		t.Error(err)
		return
	}

	t.Run("check", func(t *testing.T) {
		result, err := db.CheckResourcePermissions(nil, "device", "a", "u1", []string{}, []string{}, model.Read)
		if err != nil {
			t.Error(err)
			return
		}
		if !result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "a", "u2", []string{"g1"}, []string{}, model.Write)
		if err != nil {
			t.Error(err)
			return
		}
		if !result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "a", "u1", []string{}, []string{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "a", "u1", []string{"g1", "g2", "g3"}, []string{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if !result {
			t.Errorf("%#v", result)
			return
		}
	})

	t.Run("list", func(t *testing.T) {
		result, err := db.ListResourceIdsByPermissions(nil, "device", "u1", []string{}, []string{}, model.ListOptions{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if len(result) != 0 {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.ListResourceIdsByPermissions(nil, "device", "u1", []string{"g1", "g2", "g3"}, []string{}, model.ListOptions{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(result, []string{"a"}) {
			t.Errorf("%#v", result)
			return
		}

		result2, err := db.ListResourcesByPermissions(nil, "device", "u1", []string{"g1", "g2", "g3"}, []string{}, model.ListOptions{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(result2, []model.Resource{
			{
				Id:      "a",
				TopicId: "device",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{"u1": {Read: true}},
					GroupPermissions: map[string]model.PermissionsMap{},
					RolePermissions:  map[string]model.PermissionsMap{"g1": {Write: true}, "g2": {Administrate: true}, "g3": {Execute: true}},
				},
			},
		}) {
			t.Errorf("%#v", result)
			return
		}
	})

}

func TestDistributedRightsGroups(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	port, _, err := docker.MongoDB(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.MongoUrl = "mongodb://localhost:" + port

	db, err := New(config)
	if err != nil {
		t.Error(err)
		return
	}

	err = db.SetResource(nil, model.Resource{
		Id:      "a",
		TopicId: "device",
		ResourcePermissions: model.ResourcePermissions{
			UserPermissions:  map[string]model.PermissionsMap{"u1": {Read: true}},
			GroupPermissions: map[string]model.PermissionsMap{"g1": {Write: true}, "g2": {Administrate: true}, "g3": {Execute: true}},
		},
	}, getTestTime(0), true)

	if err != nil {
		t.Error(err)
		return
	}

	t.Run("check", func(t *testing.T) {
		result, err := db.CheckResourcePermissions(nil, "device", "a", "u1", []string{}, []string{}, model.Read)
		if err != nil {
			t.Error(err)
			return
		}
		if !result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "a", "u2", []string{}, []string{"g1"}, model.Write)
		if err != nil {
			t.Error(err)
			return
		}
		if !result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "a", "u1", []string{}, []string{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "a", "u1", []string{}, []string{"g1", "g2", "g3"}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if !result {
			t.Errorf("%#v", result)
			return
		}
	})

	t.Run("list", func(t *testing.T) {
		result, err := db.ListResourceIdsByPermissions(nil, "device", "u1", []string{}, []string{}, model.ListOptions{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if len(result) != 0 {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.ListResourceIdsByPermissions(nil, "device", "u1", []string{}, []string{"g1", "g2", "g3"}, model.ListOptions{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(result, []string{"a"}) {
			t.Errorf("%#v", result)
			return
		}

		result2, err := db.ListResourcesByPermissions(nil, "device", "u1", []string{}, []string{"g1", "g2", "g3"}, model.ListOptions{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(result2, []model.Resource{
			{
				Id:      "a",
				TopicId: "device",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{"u1": {Read: true}},
					RolePermissions:  map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{"g1": {Write: true}, "g2": {Administrate: true}, "g3": {Execute: true}},
				},
			},
		}) {
			t.Errorf("%#v", result2)
			return
		}
	})

}

func TestDistributedRightsMixed(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	port, _, err := docker.MongoDB(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.MongoUrl = "mongodb://localhost:" + port

	db, err := New(config)
	if err != nil {
		t.Error(err)
		return
	}

	err = db.SetResource(nil, model.Resource{
		Id:      "a",
		TopicId: "device",
		ResourcePermissions: model.ResourcePermissions{
			UserPermissions:  map[string]model.PermissionsMap{"u1": {Read: true}},
			RolePermissions:  map[string]model.PermissionsMap{"g1": {Write: true}},
			GroupPermissions: map[string]model.PermissionsMap{"g2": {Administrate: true}, "g3": {Execute: true}},
		},
	}, getTestTime(0), true)

	if err != nil {
		t.Error(err)
		return
	}

	t.Run("check", func(t *testing.T) {
		result, err := db.CheckResourcePermissions(nil, "device", "a", "u1", []string{}, []string{}, model.Read)
		if err != nil {
			t.Error(err)
			return
		}
		if !result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "a", "u2", []string{"g1"}, []string{}, model.Write)
		if err != nil {
			t.Error(err)
			return
		}
		if !result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "a", "u1", []string{}, []string{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "a", "u1", []string{"g1"}, []string{"g2", "g3"}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if !result {
			t.Errorf("%#v", result)
			return
		}
	})

	t.Run("list", func(t *testing.T) {
		result, err := db.ListResourceIdsByPermissions(nil, "device", "u1", []string{}, []string{}, model.ListOptions{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if len(result) != 0 {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.ListResourceIdsByPermissions(nil, "device", "u1", []string{"g1"}, []string{"g2", "g3"}, model.ListOptions{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(result, []string{"a"}) {
			t.Errorf("%#v", result)
			return
		}

		result2, err := db.ListResourcesByPermissions(nil, "device", "u1", []string{"g1"}, []string{"g2", "g3"}, model.ListOptions{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(result2, []model.Resource{
			{
				Id:      "a",
				TopicId: "device",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{"u1": {Read: true}},
					RolePermissions:  map[string]model.PermissionsMap{"g1": {Write: true}},
					GroupPermissions: map[string]model.PermissionsMap{"g2": {Administrate: true}, "g3": {Execute: true}},
				},
			},
		}) {
			t.Errorf("%#v", result2)
			return
		}
	})

}

func getTestTime(secDelta int) time.Time {
	return time.Date(2020, 6, 15, 12, 30, 30, 0, time.UTC).Add(time.Duration(secDelta) * time.Second)
}
