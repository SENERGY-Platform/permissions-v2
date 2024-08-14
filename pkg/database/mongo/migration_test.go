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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"reflect"
	"sync"
	"testing"
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

	port, _, err := docker.MongoDB(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.MongoUrl = "mongodb://localhost:" + port

	t.Run("create legacy entries", func(t *testing.T) {
		ctx, _ := getTimeoutContext()
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.MongoUrl))
		if err != nil {
			t.Error(err)
			return
		}
		db := &Database{config: config, client: client}
		collection := db.client.Database(db.config.MongoDatabase).Collection(db.config.MongoPermissionsCollection)
		err = db.ensureCompoundIndex(collection, "permissionsbytopicandid", true, true, PermissionsEntryBson.TopicId, PermissionsEntryBson.Id)
		if err != nil {
			t.Error(err)
			return
		}

		list := []LegacyPermissionsEntry{
			{
				TopicId:       "foo",
				Id:            "a",
				Timestamp:     21,
				AdminUsers:    []string{"owner"},
				AdminGroups:   []string{"admin"},
				ReadUsers:     []string{"owner", "ur"},
				ReadGroups:    []string{"gr", "admin"},
				WriteUsers:    []string{"owner", "uw"},
				WriteGroups:   []string{"gw", "admin"},
				ExecuteUsers:  []string{"owner", "ux"},
				ExecuteGroups: []string{"gx", "admin"},
			},
			{
				TopicId:       "foo",
				Id:            "b",
				Timestamp:     42,
				AdminUsers:    []string{"owner"},
				AdminGroups:   nil,
				ReadUsers:     nil,
				ReadGroups:    nil,
				WriteUsers:    nil,
				WriteGroups:   nil,
				ExecuteUsers:  nil,
				ExecuteGroups: nil,
			},
			{
				TopicId:       "foo",
				Id:            "c",
				Timestamp:     42,
				AdminUsers:    []string{"owner"},
				AdminGroups:   []string{},
				ReadUsers:     []string{},
				ReadGroups:    []string{},
				WriteUsers:    []string{},
				WriteGroups:   []string{},
				ExecuteUsers:  []string{},
				ExecuteGroups: []string{},
			},
		}

		for _, element := range list {
			_, err = collection.ReplaceOne(ctx, bson.M{PermissionsEntryBson.TopicId: element.TopicId, PermissionsEntryBson.Id: element.Id}, element, options.Replace().SetUpsert(true))
			if err != nil {
				t.Error(err)
			}
		}

	})

	expected := []model.Resource{
		{
			Id:      "a",
			TopicId: "foo",
			ResourcePermissions: model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{
					"owner": {Read: true, Write: true, Execute: true, Administrate: true},
					"ur":    {Read: true, Write: false, Execute: false, Administrate: false},
					"uw":    {Read: false, Write: true, Execute: false, Administrate: false},
					"ux":    {Read: false, Write: false, Execute: true, Administrate: false},
				},
				GroupPermissions: map[string]model.PermissionsMap{},
				RolePermissions: map[string]model.PermissionsMap{
					"admin": {Read: true, Write: true, Execute: true, Administrate: true},
					"gr":    {Read: true, Write: false, Execute: false, Administrate: false},
					"gw":    {Read: false, Write: true, Execute: false, Administrate: false},
					"gx":    {Read: false, Write: false, Execute: true, Administrate: false},
				},
			},
		},
		{
			Id:      "b",
			TopicId: "foo",
			ResourcePermissions: model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{
					"owner": {Read: false, Write: false, Execute: false, Administrate: true},
				},
				GroupPermissions: map[string]model.PermissionsMap{},
				RolePermissions:  map[string]model.PermissionsMap{},
			},
		},
		{
			Id:      "c",
			TopicId: "foo",
			ResourcePermissions: model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{
					"owner": {Read: false, Write: false, Execute: false, Administrate: true},
				},
				GroupPermissions: map[string]model.PermissionsMap{},
				RolePermissions:  map[string]model.PermissionsMap{},
			},
		},
	}

	t.Run("init", func(t *testing.T) {
		c, err := New(config)
		if err != nil {
			t.Error(err)
			return
		}
		ctx, _ := getTimeoutContext()
		list, err := c.AdminListResources(ctx, "foo", model.ListOptions{})
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(list, expected) {
			t.Errorf("\n%#v\n%#v\n", list, expected)
			return
		}
	})

	t.Run("re-init", func(t *testing.T) {
		c, err := New(config)
		if err != nil {
			t.Error(err)
			return
		}
		ctx, _ := getTimeoutContext()
		list, err := c.AdminListResources(ctx, "foo", model.ListOptions{})
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
