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

func (this *Mongo) SetResourcePermissions(r model.Resource, t time.Time, preventOlderUpdates bool) (updateIgnored bool, err error) {
	if preventOlderUpdates {
		ctx, _ := getTimeoutContext()
		updateIgnored, err = this.newerResourceExists(ctx, r.TopicId, r.Id, t)
		if err != nil {
			return updateIgnored, err
		}
		if updateIgnored {
			return updateIgnored, nil
		}
	}
	return false, this.SetRights(r.TopicId, r.Id, r.ResourceRights, t)
}

func (this *Mongo) newerResourceExists(ctx context.Context, topicId string, id string, t time.Time) (exists bool, err error) {
	err = this.rightsCollection().FindOne(ctx, bson.M{PermissionsEntryBson.TopicId: topicId, PermissionsEntryBson.Id: id, PermissionsEntryTimestampBson: bson.M{"$gt": t.Unix()}}).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (this *Mongo) SetRights(topic string, id string, rights model.ResourceRights, t time.Time) (err error) {
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
	element.setResourceRights(rights)

	ctx, _ := getTimeoutContext()

	_, err = this.rightsCollection().ReplaceOne(ctx, bson.M{PermissionsEntryBson.TopicId: element.TopicId, PermissionsEntryBson.Id: element.Id}, element, options.Replace().SetUpsert(true))

	return err
}

func (this *PermissionsEntry) setResourceRights(rights model.ResourceRights) {
	for group, right := range rights.GroupRights {
		if right.Administrate {
			this.AdminGroups = append(this.AdminGroups, group)
		}
		if right.Execute {
			this.ExecuteGroups = append(this.ExecuteGroups, group)
		}
		if right.Write {
			this.WriteGroups = append(this.WriteGroups, group)
		}
		if right.Read {
			this.ReadGroups = append(this.ReadGroups, group)
		}
	}
	for user, right := range rights.UserRights {
		if right.Administrate {
			this.AdminUsers = append(this.AdminUsers, user)
		}
		if right.Execute {
			this.ExecuteUsers = append(this.ExecuteUsers, user)
		}
		if right.Write {
			this.WriteUsers = append(this.WriteUsers, user)
		}
		if right.Read {
			this.ReadUsers = append(this.ReadUsers, user)
		}
	}
}
