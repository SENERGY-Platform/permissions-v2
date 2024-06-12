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

func TestResourcePermissions(t *testing.T) {
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
		r                   model.Resource
		t                   time.Time
		preventOlderUpdates bool
		expectUpdateIgnored bool
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
					GroupPermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g2": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t:                   getTestTime(0),
			preventOlderUpdates: true,
			expectUpdateIgnored: false,
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
					GroupPermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t:                   getTestTime(100),
			preventOlderUpdates: true,
			expectUpdateIgnored: false,
		},
		{
			r: model.Resource{
				Id:      "a",
				TopicId: "device",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u2": {Read: true, Write: true, Execute: true, Administrate: true},
						"u3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					GroupPermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t:                   getTestTime(50),
			preventOlderUpdates: true,
			expectUpdateIgnored: true,
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
					GroupPermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g2": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t:                   getTestTime(0),
			preventOlderUpdates: true,
			expectUpdateIgnored: false,
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
					GroupPermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t:                   getTestTime(100),
			preventOlderUpdates: true,
			expectUpdateIgnored: false,
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
					GroupPermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t:                   getTestTime(50),
			preventOlderUpdates: false,
			expectUpdateIgnored: false,
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
					GroupPermissions: map[string]model.PermissionsMap{
						"g1": {Read: true, Write: true, Execute: true, Administrate: true},
						"g3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t:                   getTestTime(50),
			preventOlderUpdates: false,
			expectUpdateIgnored: false,
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
					GroupPermissions: map[string]model.PermissionsMap{
						"g3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t:                   getTestTime(50),
			preventOlderUpdates: true,
			expectUpdateIgnored: false,
		},
	}

	t.Run("set", func(t *testing.T) {
		for i, update := range updates {
			updateIgnored, err := db.SetResourcePermissions(nil, update.r, update.t, update.preventOlderUpdates)
			if err != nil {
				t.Error(i, err)
				return
			}
			if update.expectUpdateIgnored != updateIgnored {
				t.Errorf("%v SetResourcePermissions(%#v,%#v%#v) = %#v \nexpectUpdateIgnored=%#v\n", i, update.r, update.t, update.preventOlderUpdates, updateIgnored, update.expectUpdateIgnored)
				return
			}
		}
	})

	listQueries := []struct {
		topic          string
		user           string
		group          string
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
			group:  "g1",
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
			group:          "g2",
			rights:         model.PermissionList{model.Read},
			expectedResult: []model.Resource{},
		},
		{
			topic:  "device",
			group:  "g3",
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
			group:  "g1",
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
			group:          "g2",
			rights:         model.PermissionList{model.Administrate},
			expectedResult: []model.Resource{},
		},
		{
			topic:          "device",
			group:          "g3",
			rights:         model.PermissionList{model.Administrate},
			expectedResult: []model.Resource{},
		},

		{
			topic:  "device",
			user:   "u1",
			group:  "g3",
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
			group:  "g3",
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
			result, err := db.ListResourcesByPermissions(nil, q.topic, q.user, []string{q.group}, q.options, q.rights...)
			if err != nil {
				t.Error(i, err)
				return
			}
			if !reflect.DeepEqual(result, q.expectedResult) {
				t.Errorf("%v ListResourcesByPermissions(topic=%#v,user=%#v,group=%#v,rights=%#v,options=%#v) != expected\n%#v\n%#v\n", i, q.topic, q.user, q.group, q.rights, q.options, result, q.expectedResult)
				return
			}

			expectedIds := []string{}
			for _, e := range q.expectedResult {
				expectedIds = append(expectedIds, e.Id)
			}
			actualIds, err := db.ListResourceIdsByPermissions(nil, q.topic, q.user, []string{q.group}, q.options, q.rights...)
			if err != nil {
				t.Error(i, err)
				return
			}
			if !reflect.DeepEqual(actualIds, expectedIds) {
				t.Errorf("%v ListResourceIdsByPermissions(topic=%#v,user=%#v,group=%#v,rights=%#v,options=%#v) != expected\n%#v\n%#v\n", i, q.topic, q.user, q.group, q.rights, q.options, actualIds, expectedIds)
				return
			}
		}
	})

	t.Run("check", func(t *testing.T) {
		result, err := db.CheckResourcePermissions(nil, "device", "a", "u1", []string{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if !result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "d", "u1", []string{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "d", "u1", []string{"g1"}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "d", "u1", []string{"g1"}, model.Read)
		if err != nil {
			t.Error(err)
			return
		}
		if result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "d", "u1", []string{"g1", "g3"}, model.Read)
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
		result, err := db.CheckMultipleResourcePermissions(nil, "device", []string{"a", "b", "c", "d", "e", "x", "y"}, "u1", []string{}, model.Read, model.Write, model.Administrate, model.Execute)
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

func TestDistributedRights(t *testing.T) {
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

	_, err = db.SetResourcePermissions(nil, model.Resource{
		Id:      "a",
		TopicId: "device",
		ResourcePermissions: model.ResourcePermissions{
			UserPermissions:  map[string]model.PermissionsMap{"u1": {Read: true}},
			GroupPermissions: map[string]model.PermissionsMap{"g1": {Write: true}, "g2": {Administrate: true}, "g3": {Execute: true}},
		},
	}, getTestTime(0), false)

	if err != nil {
		t.Error(err)
		return
	}

	t.Run("check", func(t *testing.T) {
		result, err := db.CheckResourcePermissions(nil, "device", "a", "u1", []string{}, model.Read)
		if err != nil {
			t.Error(err)
			return
		}
		if !result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "a", "u2", []string{"g1"}, model.Write)
		if err != nil {
			t.Error(err)
			return
		}
		if !result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "a", "u1", []string{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if result {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.CheckResourcePermissions(nil, "device", "a", "u1", []string{"g1", "g2", "g3"}, model.Read, model.Write, model.Administrate, model.Execute)
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
		result, err := db.ListResourceIdsByPermissions(nil, "device", "u1", []string{}, model.ListOptions{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if len(result) != 0 {
			t.Errorf("%#v", result)
			return
		}

		result, err = db.ListResourceIdsByPermissions(nil, "device", "u1", []string{"g1", "g2", "g3"}, model.ListOptions{}, model.Read, model.Write, model.Administrate, model.Execute)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(result, []string{"a"}) {
			t.Errorf("%#v", result)
			return
		}

		result2, err := db.ListResourcesByPermissions(nil, "device", "u1", []string{"g1", "g2", "g3"}, model.ListOptions{}, model.Read, model.Write, model.Administrate, model.Execute)
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
					GroupPermissions: map[string]model.PermissionsMap{"g1": {Write: true}, "g2": {Administrate: true}, "g3": {Execute: true}},
				},
			},
		}) {
			t.Errorf("%#v", result)
			return
		}
	})

}

func getTestTime(secDelta int) time.Time {
	return time.Date(2020, 6, 15, 12, 30, 30, 0, time.UTC).Add(time.Duration(secDelta) * time.Second)
}
