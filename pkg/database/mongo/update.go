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
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

func (this *Database) SetResource(ctx context.Context, r model.Resource, t time.Time, synced bool) (err error) {
	if ctx == nil {
		ctx, _ = getTimeoutContext()
	}
	return this.SetPermissions(ctx, r.TopicId, r.Id, r.ResourcePermissions, t, synced)
}

func (this *Database) DeleteResource(ctx context.Context, topicId string, id string) error {
	if ctx == nil {
		ctx, _ = getTimeoutContext()
	}
	_, err := this.permissionsCollection().DeleteMany(ctx, bson.M{PermissionsEntryBson.TopicId: topicId, PermissionsEntryBson.Id: id})
	return err
}

func (this *Database) MarkResourceAsSynced(ctx context.Context, topicId string, id string) error {
	if ctx == nil {
		ctx, _ = getTimeoutContext()
	}
	_, err := this.permissionsCollection().UpdateMany(ctx, bson.M{
		PermissionsEntryBson.TopicId: topicId,
		PermissionsEntryBson.Id:      id,
	}, bson.M{"$set": bson.M{PermissionsEntrySyncedBson: true}})
	return err
}

func (this *Database) ListUnsyncedResources(ctx context.Context) (result []model.Resource, err error) {
	if ctx == nil {
		ctx, _ = getTimeoutContext()
	}
	opt := options.Find()
	opt.SetSort(bson.D{{PermissionsEntryBson.Id, 1}})
	cursor, err := this.permissionsCollection().Find(ctx, bson.M{
		PermissionsEntrySyncedBson:    false,
		PermissionsEntryTimestampBson: bson.M{"$lt": time.Now().Add(-1 * this.config.SyncAgeLimit.GetDuration()).UnixMilli()},
	}, opt)
	if err != nil {
		return result, err
	}
	for cursor.Next(context.Background()) {
		element := PermissionsEntry{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result = append(result, element.ToResource())
	}
	err = cursor.Err()
	return result, err
}

func (this *Database) SetPermissions(ctx context.Context, topic string, id string, permissions model.ResourcePermissions, t time.Time, synced bool) (err error) {
	if ctx == nil {
		ctx, _ = getTimeoutContext()
	}
	element := PermissionsEntry{
		Timestamp:     t.UnixMilli(),
		Synced:        synced,
		TopicId:       topic,
		Id:            id,
		AdminUsers:    []string{},
		AdminGroups:   []string{},
		AdminRoles:    []string{},
		ReadUsers:     []string{},
		ReadGroups:    []string{},
		ReadRoles:     []string{},
		WriteUsers:    []string{},
		WriteGroups:   []string{},
		WriteRoles:    []string{},
		ExecuteUsers:  []string{},
		ExecuteGroups: []string{},
		ExecuteRoles:  []string{},
	}
	element.setResourcePermissions(permissions)
	_, err = this.permissionsCollection().ReplaceOne(ctx, bson.M{PermissionsEntryBson.TopicId: element.TopicId, PermissionsEntryBson.Id: element.Id}, element, options.Replace().SetUpsert(true))
	return err
}

func (this *PermissionsEntry) setResourcePermissions(permissions model.ResourcePermissions) {
	for group, permission := range permissions.GroupPermissions {
		if permission.Administrate {
			this.AdminGroups = append(this.AdminGroups, group)
		}
		if permission.Execute {
			this.ExecuteGroups = append(this.ExecuteGroups, group)
		}
		if permission.Write {
			this.WriteGroups = append(this.WriteGroups, group)
		}
		if permission.Read {
			this.ReadGroups = append(this.ReadGroups, group)
		}
	}
	for role, permission := range permissions.RolePermissions {
		if permission.Administrate {
			this.AdminRoles = append(this.AdminRoles, role)
		}
		if permission.Execute {
			this.ExecuteRoles = append(this.ExecuteRoles, role)
		}
		if permission.Write {
			this.WriteRoles = append(this.WriteRoles, role)
		}
		if permission.Read {
			this.ReadRoles = append(this.ReadRoles, role)
		}
	}
	for user, permission := range permissions.UserPermissions {
		if permission.Administrate {
			this.AdminUsers = append(this.AdminUsers, user)
		}
		if permission.Execute {
			this.ExecuteUsers = append(this.ExecuteUsers, user)
		}
		if permission.Write {
			this.WriteUsers = append(this.WriteUsers, user)
		}
		if permission.Read {
			this.ReadUsers = append(this.ReadUsers, user)
		}
	}
}
