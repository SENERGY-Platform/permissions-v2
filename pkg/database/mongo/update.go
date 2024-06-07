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
	"errors"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

func (this *Database) SetResourcePermissions(ctx context.Context, r model.Resource, t time.Time, preventOlderUpdates bool) (updateIgnored bool, err error) {
	if ctx == nil {
		ctx, _ = getTimeoutContext()
	}
	if preventOlderUpdates {
		updateIgnored, err = this.newerResourceExists(ctx, r.TopicId, r.Id, t)
		if err != nil {
			return updateIgnored, err
		}
		if updateIgnored {
			return updateIgnored, nil
		}
	}
	return false, this.SetPermissions(ctx, r.TopicId, r.Id, r.ResourcePermissions, t)
}

func (this *Database) DeleteResource(ctx context.Context, topicId string, id string) error {
	if ctx == nil {
		ctx, _ = getTimeoutContext()
	}
	_, err := this.permissionsCollection().DeleteMany(ctx, bson.M{PermissionsEntryBson.TopicId: topicId, PermissionsEntryBson.Id: id})
	return err
}

func (this *Database) newerResourceExists(ctx context.Context, topicId string, id string, t time.Time) (exists bool, err error) {
	err = this.permissionsCollection().FindOne(ctx, bson.M{PermissionsEntryBson.TopicId: topicId, PermissionsEntryBson.Id: id, PermissionsEntryTimestampBson: bson.M{"$gte": t.Unix()}}).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (this *Database) SetPermissions(ctx context.Context, topic string, id string, permissions model.ResourcePermissions, t time.Time) (err error) {
	if ctx == nil {
		ctx, _ = getTimeoutContext()
	}
	element := PermissionsEntry{
		Timestamp:     t.Unix(),
		TopicId:       topic,
		Id:            id,
		AdminUsers:    []string{},
		AdminGroups:   []string{},
		ReadUsers:     []string{},
		ReadGroups:    []string{},
		WriteUsers:    []string{},
		WriteGroups:   []string{},
		ExecuteUsers:  []string{},
		ExecuteGroups: []string{},
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
