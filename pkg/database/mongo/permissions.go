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
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"go.mongodb.org/mongo-driver/mongo"
)

var PermissionsEntryBson = getBsonFieldObject[PermissionsEntry]()

const PermissionsEntryTimestampBson = "timestamp"

func init() {
	CreateCollections = append(CreateCollections, func(db *Database) error {
		var err error
		collection := db.client.Database(db.config.MongoDatabase).Collection(db.config.MongoPermissionsCollection)
		err = db.ensureCompoundIndex(collection, "permissionsbytopicandid", true, true, PermissionsEntryBson.TopicId, PermissionsEntryBson.Id)
		if err != nil {
			return err
		}
		return nil
	})
}

type PermissionsEntry struct {
	Timestamp     int64    `json:"timestamp"`
	TopicId       string   `json:"topic_id" bson:"topic_id"`
	Id            string   `json:"id" bson:"id"`
	AdminUsers    []string `json:"admin_users" bson:"admin_users"`
	AdminGroups   []string `json:"admin_groups" bson:"admin_groups"`
	ReadUsers     []string `json:"read_users" bson:"read_users"`
	ReadGroups    []string `json:"read_groups" bson:"read_groups"`
	WriteUsers    []string `json:"write_users" bson:"write_users"`
	WriteGroups   []string `json:"write_groups" bson:"write_groups"`
	ExecuteUsers  []string `json:"execute_users" bson:"execute_users"`
	ExecuteGroups []string `json:"execute_groups" bson:"execute_groups"`
}

func (this *Database) permissionsCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoDatabase).Collection(this.config.MongoPermissionsCollection)
}

func (this *PermissionsEntry) ToResource() model.Resource {
	result := model.Resource{
		Id:      this.Id,
		TopicId: this.TopicId,
		ResourcePermissions: model.ResourcePermissions{
			UserPermissions:  map[string]model.PermissionsMap{},
			GroupPermissions: map[string]model.PermissionsMap{},
		},
	}
	for _, user := range this.AdminUsers {
		if _, ok := result.UserPermissions[user]; !ok {
			result.UserPermissions[user] = model.PermissionsMap{}
		}
		permissions := result.UserPermissions[user]
		permissions.Administrate = true
		result.UserPermissions[user] = permissions
	}
	for _, user := range this.ReadUsers {
		if _, ok := result.UserPermissions[user]; !ok {
			result.UserPermissions[user] = model.PermissionsMap{}
		}
		permissions := result.UserPermissions[user]
		permissions.Read = true
		result.UserPermissions[user] = permissions
	}
	for _, user := range this.WriteUsers {
		if _, ok := result.UserPermissions[user]; !ok {
			result.UserPermissions[user] = model.PermissionsMap{}
		}
		permissions := result.UserPermissions[user]
		permissions.Write = true
		result.UserPermissions[user] = permissions
	}
	for _, user := range this.ExecuteUsers {
		if _, ok := result.UserPermissions[user]; !ok {
			result.UserPermissions[user] = model.PermissionsMap{}
		}
		permissions := result.UserPermissions[user]
		permissions.Execute = true
		result.UserPermissions[user] = permissions
	}

	result.GroupPermissions = map[string]model.PermissionsMap{}
	for _, group := range this.AdminGroups {
		if _, ok := result.GroupPermissions[group]; !ok {
			result.GroupPermissions[group] = model.PermissionsMap{}
		}
		permissions := result.GroupPermissions[group]
		permissions.Administrate = true
		result.GroupPermissions[group] = permissions
	}
	for _, group := range this.ReadGroups {
		if _, ok := result.GroupPermissions[group]; !ok {
			result.GroupPermissions[group] = model.PermissionsMap{}
		}
		permissions := result.GroupPermissions[group]
		permissions.Read = true
		result.GroupPermissions[group] = permissions
	}
	for _, group := range this.WriteGroups {
		if _, ok := result.GroupPermissions[group]; !ok {
			result.GroupPermissions[group] = model.PermissionsMap{}
		}
		permissions := result.GroupPermissions[group]
		permissions.Write = true
		result.GroupPermissions[group] = permissions
	}
	for _, group := range this.ExecuteGroups {
		if _, ok := result.GroupPermissions[group]; !ok {
			result.GroupPermissions[group] = model.PermissionsMap{}
		}
		permissions := result.GroupPermissions[group]
		permissions.Execute = true
		result.GroupPermissions[group] = permissions
	}
	return result
}
