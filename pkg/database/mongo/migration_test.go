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

package mongo

import (
	"context"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/permissions-v2/pkg/tests/docker"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

type LegacyPermissionsEntry struct {
	TopicId       string   `json:"topic_id" bson:"topic_id"`
	Id            string   `json:"id" bson:"id"`
	Timestamp     int64    `json:"timestamp"`
	AdminUsers    []string `json:"admin_users" bson:"admin_users"`
	AdminGroups   []string `json:"admin_groups" bson:"admin_groups"`
	ReadUsers     []string `json:"read_users" bson:"read_users"`
	ReadGroups    []string `json:"read_groups" bson:"read_groups"`
	WriteUsers    []string `json:"write_users" bson:"write_users"`
	WriteGroups   []string `json:"write_groups" bson:"write_groups"`
	ExecuteUsers  []string `json:"execute_users" bson:"execute_users"`
	ExecuteGroups []string `json:"execute_groups" bson:"execute_groups"`
}

func TestMigration(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.Load("../../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	sourceConfig, err := configuration.Load("../../../config.json")
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

	port, _, err = docker.MongoDB(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.MigrateFromMongoUrl = "mongodb://localhost:" + port
	sourceConfig.MongoUrl = config.MigrateFromMongoUrl

	t.Run("create source entries", func(t *testing.T) {
		source, err := New(sourceConfig)
		if err != nil {
			t.Error(err)
			return
		}

		err = source.SetTopic(ctx, model.Topic{
			Id:                  "t1",
			PublishToKafkaTopic: "t1",
		})
		if err != nil {
			t.Error(err)
			return
		}

		err = source.SetTopic(ctx, model.Topic{
			Id: "t2",
		})
		if err != nil {
			t.Error(err)
			return
		}

		err = source.SetPermissions(ctx, "t1", "r1", model.ResourcePermissions{
			UserPermissions: map[string]model.PermissionsMap{
				"ut1r1": {Read: true, Write: true, Execute: true, Administrate: true},
			},
		}, time.Unix(10, 0), false)
		if err != nil {
			t.Error(err)
			return
		}
		err = source.SetPermissions(ctx, "t1", "r2", model.ResourcePermissions{
			UserPermissions: map[string]model.PermissionsMap{
				"ut1r2": {Read: true, Write: true, Execute: true, Administrate: true},
			},
		}, time.Unix(15, 0), true)
		if err != nil {
			t.Error(err)
			return
		}

		err = source.SetPermissions(ctx, "t2", "r1", model.ResourcePermissions{
			UserPermissions: map[string]model.PermissionsMap{
				"ut2r1": {Read: true, Write: true, Execute: true, Administrate: true},
			},
		}, time.Unix(20, 0), true)
		if err != nil {
			t.Error(err)
			return
		}

		err = source.SetPermissions(ctx, "t2", "r2", model.ResourcePermissions{
			UserPermissions: map[string]model.PermissionsMap{
				"ut2r2": {Read: true, Write: true, Execute: true, Administrate: true},
			},
		}, time.Unix(25, 0), false)
		if err != nil {
			t.Error(err)
			return
		}

	})

	var db *Database
	t.Run("migrate", func(t *testing.T) {
		db, err = New(config)
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("check topics", func(t *testing.T) {
		list, err := db.ListTopics(ctx, model.ListOptions{})
		if err != nil {
			t.Error(err)
			return
		}
		slices.SortFunc(list, func(a, b model.Topic) int {
			return strings.Compare(a.Id, b.Id)
		})

		//remove timestamps
		for i, t := range list {
			t.LastUpdateUnixTimestamp = 0
			list[i] = t
		}

		if !reflect.DeepEqual(list, []model.Topic{
			{Id: "t1", PublishToKafkaTopic: "t1"},
			{Id: "t2"},
		}) {
			t.Errorf("%#v\n", list)
			return
		}
	})

	t.Run("check resources t1", func(t *testing.T) {
		list, err := db.AdminListResources(ctx, "t1", model.ListOptions{})
		if err != nil {
			t.Error(err)
			return
		}
		slices.SortFunc(list, func(a, b model.Resource) int {
			return strings.Compare(a.Id, b.Id)
		})

		expected := []model.Resource{
			{
				Id:      "r1",
				TopicId: "t1",
				ResourcePermissions: model.ResourcePermissions{
					RolePermissions:  map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{},
					UserPermissions: map[string]model.PermissionsMap{
						"ut1r1": {Read: true, Write: true, Execute: true, Administrate: true},
					},
				},
			},
			{
				Id:      "r2",
				TopicId: "t1",
				ResourcePermissions: model.ResourcePermissions{
					RolePermissions:  map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{},
					UserPermissions: map[string]model.PermissionsMap{
						"ut1r2": {Read: true, Write: true, Execute: true, Administrate: true},
					},
				},
			},
		}

		if !reflect.DeepEqual(list, expected) {
			t.Errorf("\na:%#v\ne:%#v\n", list, expected)
			return
		}
	})

	t.Run("check resources t2", func(t *testing.T) {
		list, err := db.AdminListResources(ctx, "t2", model.ListOptions{})
		if err != nil {
			t.Error(err)
			return
		}
		slices.SortFunc(list, func(a, b model.Resource) int {
			return strings.Compare(a.Id, b.Id)
		})

		expected := []model.Resource{
			{
				Id:      "r1",
				TopicId: "t2",
				ResourcePermissions: model.ResourcePermissions{
					RolePermissions:  map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{},
					UserPermissions: map[string]model.PermissionsMap{
						"ut2r1": {Read: true, Write: true, Execute: true, Administrate: true},
					},
				},
			},
			{
				Id:      "r2",
				TopicId: "t2",
				ResourcePermissions: model.ResourcePermissions{
					RolePermissions:  map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{},
					UserPermissions: map[string]model.PermissionsMap{
						"ut2r2": {Read: true, Write: true, Execute: true, Administrate: true},
					},
				},
			},
		}

		if !reflect.DeepEqual(list, expected) {
			t.Errorf("\na:%#v\ne:%#v\n", list, expected)
			return
		}
	})

	t.Run("check sync state", func(t *testing.T) {
		list, err := db.ListUnsyncedResources(ctx)
		if err != nil {
			t.Error(err)
			return
		}
		slices.SortFunc(list, func(a, b model.Resource) int {
			return strings.Compare(a.TopicId+a.Id, b.TopicId+b.Id)
		})

		expected := []model.Resource{
			{
				Id:      "r1",
				TopicId: "t1",
				ResourcePermissions: model.ResourcePermissions{
					RolePermissions:  map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{},
					UserPermissions: map[string]model.PermissionsMap{
						"ut1r1": {Read: true, Write: true, Execute: true, Administrate: true},
					},
				},
			},
			{
				Id:      "r2",
				TopicId: "t2",
				ResourcePermissions: model.ResourcePermissions{
					RolePermissions:  map[string]model.PermissionsMap{},
					GroupPermissions: map[string]model.PermissionsMap{},
					UserPermissions: map[string]model.PermissionsMap{
						"ut2r2": {Read: true, Write: true, Execute: true, Administrate: true},
					},
				},
			},
		}

		if !reflect.DeepEqual(list, expected) {
			t.Errorf("\na:%#v\ne:%#v\n", list, expected)
			return
		}
	})
}
