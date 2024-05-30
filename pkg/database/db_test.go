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
	"github.com/SENERGY-Platform/permissions-v2/pkg/database/mongo"
	"github.com/SENERGY-Platform/permissions-v2/pkg/database/postgres"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/permissions-v2/pkg/tests/docker"
	"reflect"
	"sync"
	"testing"
	"time"
)

type TestDatabase interface {
	SetResourcePermissions(r model.Resource, t time.Time, preventOlderUpdates bool) (updateIgnored bool, err error)
	ListByRights(topicId string, userId string, groupIds []string, rights string, options model.ListOptions) (result []model.Resource, err error)
	CheckMultiple(topicId string, ids []string, userId string, groupIds []string, rights string) (result map[string]bool, err error)
	Reset() error //TODO: remove this test interface
}

func BenchmarkResourcePermissions(b *testing.B) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.Load("../../config.json")
	if err != nil {
		b.Error(err)
		return
	}

	config.PostgresConnStr, err = docker.Postgres(ctx, wg, "permissions")
	if err != nil {
		b.Error(err)
		return
	}

	port, _, err := docker.MongoDB(ctx, wg)
	if err != nil {
		b.Error(err)
		return
	}
	config.MongoUrl = "mongodb://localhost:" + port

	pg, err := postgres.New(config)
	if err != nil {
		b.Error(err)
		return
	}

	m, err := mongo.New(config)
	if err != nil {
		b.Error(err)
		return
	}

	time.Sleep(time.Second)

	compared := map[string]TestDatabase{"warmup_mongo": m, "warmup_postgres": pg, "mongo": m, "postgres": pg}

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
				ResourceRights: model.ResourceRights{
					UserRights: map[string]model.Right{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u2": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					GroupRights: map[string]model.Right{
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
				ResourceRights: model.ResourceRights{
					UserRights: map[string]model.Right{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					GroupRights: map[string]model.Right{
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
				ResourceRights: model.ResourceRights{
					UserRights: map[string]model.Right{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u2": {Read: true, Write: true, Execute: true, Administrate: true},
						"u3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					GroupRights: map[string]model.Right{
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
				ResourceRights: model.ResourceRights{
					UserRights: map[string]model.Right{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u2": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					GroupRights: map[string]model.Right{
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
				ResourceRights: model.ResourceRights{
					UserRights: map[string]model.Right{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					GroupRights: map[string]model.Right{
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
				ResourceRights: model.ResourceRights{
					UserRights: map[string]model.Right{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u2": {Read: true, Write: true, Execute: true, Administrate: true},
						"u3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					GroupRights: map[string]model.Right{
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
				ResourceRights: model.ResourceRights{
					UserRights: map[string]model.Right{
						"u1": {Read: true, Write: true, Execute: true, Administrate: true},
						"u2": {Read: true, Write: true, Execute: true, Administrate: true},
						"u3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					GroupRights: map[string]model.Right{
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
				ResourceRights: model.ResourceRights{
					UserRights: map[string]model.Right{
						"u2": {Read: true, Write: true, Execute: true, Administrate: true},
						"u3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
					GroupRights: map[string]model.Right{
						"g3": {Read: true, Write: false, Execute: false, Administrate: false},
					},
				},
			},
			t:                   getTestTime(50),
			preventOlderUpdates: true,
			expectUpdateIgnored: false,
		},
	}

	getSetTest := func(implName string, impl TestDatabase) func(b *testing.B) {
		return func(b *testing.B) {
			err = impl.Reset()
			if err != nil {
				b.Error(implName, err)
				return
			}
			b.ResetTimer()
			for i, update := range updates {
				updateIgnored, err := impl.SetResourcePermissions(update.r, update.t, update.preventOlderUpdates)
				if err != nil {
					b.Error(implName, err)
					return
				}
				if update.expectUpdateIgnored != updateIgnored {
					b.Errorf("%v %v SetResourcePermissions(%#v,%#v%#v) = %#v \nexpectUpdateIgnored=%#v\n", implName, i, update.r, update.t, update.preventOlderUpdates, updateIgnored, update.expectUpdateIgnored)
					return
				}
			}
		}
	}

	for implName, impl := range compared {
		b.Run("set "+implName, getSetTest(implName, impl))
	}

	listQueries := []struct {
		topic          string
		user           string
		group          string
		rights         string
		options        model.ListOptions
		expectedResult []model.Resource
	}{
		{
			topic:  "device",
			user:   "u1",
			rights: "r",
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
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
			rights: "r",
			expectedResult: []model.Resource{
				{
					Id:      "b",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "d",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:  "device",
			user:   "u3",
			rights: "r",
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "d",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:   "device",
			user:    "u3",
			rights:  "r",
			options: model.ListOptions{Limit: 2, Offset: 1},
			expectedResult: []model.Resource{
				{
					Id:      "b",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
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
			rights: "r",
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
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
			rights:         "r",
			expectedResult: []model.Resource{},
		},
		{
			topic:  "device",
			group:  "g3",
			rights: "r",
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "d",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},

		{
			topic:  "device",
			user:   "u1",
			rights: "a",
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
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
			rights: "a",
			expectedResult: []model.Resource{
				{
					Id:      "b",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "d",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
		{
			topic:          "device",
			user:           "u3",
			rights:         "a",
			expectedResult: []model.Resource{},
		},
		{
			topic:  "device",
			group:  "g1",
			rights: "a",
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
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
			rights:         "a",
			expectedResult: []model.Resource{},
		},
		{
			topic:          "device",
			group:          "g3",
			rights:         "a",
			expectedResult: []model.Resource{},
		},

		{
			topic:  "device",
			user:   "u1",
			group:  "g3",
			rights: "r",
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "d",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
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
			rights: "ra",
			expectedResult: []model.Resource{
				{
					Id:      "a",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "b",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
				{
					Id:      "c",
					TopicId: "device",
					ResourceRights: model.ResourceRights{
						UserRights: map[string]model.Right{
							"u1": {Read: true, Write: true, Execute: true, Administrate: true},
							"u2": {Read: true, Write: true, Execute: true, Administrate: true},
							"u3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
						GroupRights: map[string]model.Right{
							"g1": {Read: true, Write: true, Execute: true, Administrate: true},
							"g3": {Read: true, Write: false, Execute: false, Administrate: false},
						},
					},
				},
			},
		},
	}

	getListTest := func(implName string, impl TestDatabase) func(b *testing.B) {
		return func(b *testing.B) {
			for i, q := range listQueries {
				result, err := impl.ListByRights(q.topic, q.user, []string{q.group}, q.rights, q.options)
				if err != nil {
					b.Error(i, err)
					return
				}
				if !reflect.DeepEqual(result, q.expectedResult) {
					b.Errorf("%v %v ListByRights(topic=%#v,user=%#v,group=%#v,rights=%#v,options=%#v) != expected\n%#v\n%#v\n", implName, i, q.topic, q.user, q.group, q.rights, q.options, result, q.expectedResult)
					return
				}
			}
		}
	}

	for implName, impl := range compared {
		b.Run("list "+implName, getListTest(implName, impl))
	}

	checkMultipleTest := func(implName string, impl TestDatabase) func(b *testing.B) {
		return func(b *testing.B) {
			result, err := impl.CheckMultiple("device", []string{"a", "b", "c", "d", "e", "x", "y"}, "u1", []string{}, "rwxa")
			if err != nil {
				b.Error(err)
				return
			}
			if !reflect.DeepEqual(result, map[string]bool{
				"a": true,
				"b": true,
				"c": true,
				"d": false,
			}) {
				b.Errorf("%#v", result)
				return
			}
		}
	}

	for implName, impl := range compared {
		b.Run("check "+implName, checkMultipleTest(implName, impl))
	}
}

func getTestTime(secDelta int) time.Time {
	return time.Date(2020, 6, 15, 12, 30, 30, 0, time.UTC).Add(time.Duration(secDelta) * time.Second)
}
