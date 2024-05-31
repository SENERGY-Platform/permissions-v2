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
	"slices"
	"time"
)

func init() {
	CreateCollections = append(CreateCollections, func(db *Mongo) error {
		var err error
		collection := db.client.Database(db.config.MongoTable).Collection(db.config.MongoPermissionsCollection)
		err = db.ensureCompoundIndex(collection, "rightsbykindandid", true, true, "kind", "id")
		if err != nil {
			return err
		}
		return nil
	})
}

type RightsEntry struct {
	Timestamp     int64    `json:"timestamp"`
	Kind          string   `json:"kind" bson:"kind"`
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

func (this *Mongo) rightsCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoTable).Collection(this.config.MongoPermissionsCollection)
}

func (this *Mongo) ListIdsByRights(topicId string, userId string, groupIds []string, rights string, options model.ListOptions) ([]string, error) {
	temp, err := this.ListByRights(topicId, userId, groupIds, rights, options)
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, e := range temp {
		result = append(result, e.Id)
	}
	return result, err
}

func (this *Mongo) Check(topicId string, id string, userId string, groupIds []string, rights string) (bool, error) {
	m, err := this.CheckMultiple(topicId, []string{id}, userId, groupIds, rights)
	if err != nil {
		return false, err
	}
	return m[id], nil
}

func (this *Mongo) ListByRights(topicId string, userId string, groupIds []string, rights string, listOptions model.ListOptions) (result []model.Resource, err error) {
	result = []model.Resource{}
	ctx, _ := getTimeoutContext()
	rightsFilter := bson.A{}
	for _, r := range rights {
		switch r {
		case 'r':
			rightsFilter = append(rightsFilter, bson.M{"$or": bson.A{bson.M{"read_users": userId}, bson.M{"read_groups": bson.M{"$in": groupIds}}}})
		case 'w':
			rightsFilter = append(rightsFilter, bson.M{"$or": bson.A{bson.M{"write_users": userId}, bson.M{"write_groups": bson.M{"$in": groupIds}}}})
		case 'x':
			rightsFilter = append(rightsFilter, bson.M{"$or": bson.A{bson.M{"execute_users": userId}, bson.M{"execute_groups": bson.M{"$in": groupIds}}}})
		case 'a':
			rightsFilter = append(rightsFilter, bson.M{"$or": bson.A{bson.M{"admin_users": userId}, bson.M{"admin_groups": bson.M{"$in": groupIds}}}})
		default:
			return []model.Resource{}, errors.New("invalid rights parameter")
		}
	}

	opt := options.Find()
	if listOptions.Limit > 0 {
		opt.SetLimit(listOptions.Limit)
	}
	if listOptions.Offset > 0 {
		opt.SetSkip(listOptions.Offset)
	}
	opt.SetSort(bson.D{{"id", 1}})

	filter := bson.M{"kind": topicId, "$and": rightsFilter}
	cursor, err := this.rightsCollection().Find(ctx, filter, opt)
	if err != nil {
		return result, err
	}
	for cursor.Next(context.Background()) {
		element := RightsEntry{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result = append(result, element.ToResource())
	}

	err = cursor.Err()
	return result, err
}

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

func (this *Mongo) newerResourceExists(ctx context.Context, kind string, resourceId string, t time.Time) (exists bool, err error) {
	err = this.rightsCollection().FindOne(ctx, bson.M{"kind": kind, "id": resourceId, "timestamp": bson.M{"$gt": t.Unix()}}).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (this *Mongo) SetRights(topic string, resourceId string, rights model.ResourceRights, t time.Time) (err error) {
	element := RightsEntry{
		Timestamp:     t.Unix(),
		Kind:          topic,
		Id:            resourceId,
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

	_, err = this.rightsCollection().ReplaceOne(ctx, bson.M{"kind": element.Kind, "id": element.Id}, element, options.Replace().SetUpsert(true))

	return err
}

func (this *RightsEntry) setResourceRights(rights model.ResourceRights) {
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

func (this *RightsEntry) ToResource() model.Resource {
	result := model.Resource{
		Id:      this.Id,
		TopicId: this.Kind,
		ResourceRights: model.ResourceRights{
			UserRights:  map[string]model.Right{},
			GroupRights: map[string]model.Right{},
		},
	}
	for _, user := range this.AdminUsers {
		if _, ok := result.UserRights[user]; !ok {
			result.UserRights[user] = model.Right{}
		}
		right := result.UserRights[user]
		right.Administrate = true
		result.UserRights[user] = right
	}
	for _, user := range this.ReadUsers {
		if _, ok := result.UserRights[user]; !ok {
			result.UserRights[user] = model.Right{}
		}
		right := result.UserRights[user]
		right.Read = true
		result.UserRights[user] = right
	}
	for _, user := range this.WriteUsers {
		if _, ok := result.UserRights[user]; !ok {
			result.UserRights[user] = model.Right{}
		}
		right := result.UserRights[user]
		right.Write = true
		result.UserRights[user] = right
	}
	for _, user := range this.ExecuteUsers {
		if _, ok := result.UserRights[user]; !ok {
			result.UserRights[user] = model.Right{}
		}
		right := result.UserRights[user]
		right.Execute = true
		result.UserRights[user] = right
	}

	result.GroupRights = map[string]model.Right{}
	for _, group := range this.AdminGroups {
		if _, ok := result.GroupRights[group]; !ok {
			result.GroupRights[group] = model.Right{}
		}
		right := result.GroupRights[group]
		right.Administrate = true
		result.GroupRights[group] = right
	}
	for _, group := range this.ReadGroups {
		if _, ok := result.GroupRights[group]; !ok {
			result.GroupRights[group] = model.Right{}
		}
		right := result.GroupRights[group]
		right.Read = true
		result.GroupRights[group] = right
	}
	for _, group := range this.WriteGroups {
		if _, ok := result.GroupRights[group]; !ok {
			result.GroupRights[group] = model.Right{}
		}
		right := result.GroupRights[group]
		right.Write = true
		result.GroupRights[group] = right
	}
	for _, group := range this.ExecuteGroups {
		if _, ok := result.GroupRights[group]; !ok {
			result.GroupRights[group] = model.Right{}
		}
		right := result.GroupRights[group]
		right.Execute = true
		result.GroupRights[group] = right
	}
	return result
}

func (this *Mongo) Reset() error {
	_, err := this.rightsCollection().DeleteMany(context.Background(), bson.M{})
	return err
}

func (this *Mongo) CheckMultiple(topicId string, ids []string, userId string, groupIds []string, rights string) (result map[string]bool, err error) {
	ctx, _ := getTimeoutContext()
	cursor, err := this.rightsCollection().Find(ctx, bson.M{"kind": topicId, "id": bson.M{"$in": ids}})
	if err != nil {
		return result, err
	}
	result = map[string]bool{}
	for cursor.Next(context.Background()) {
		element := RightsEntry{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result[element.Id] = checkRights(userId, groupIds, element, rights)
	}

	err = cursor.Err()
	return result, err
}

func checkRights(userId string, groupIds []string, element RightsEntry, rights string) bool {
	for _, r := range rights {
		switch r {
		case 'a':
			if !slices.Contains(element.AdminUsers, userId) && !containsAny(element.AdminGroups, groupIds) {
				return false
			}
		case 'r':
			if !slices.Contains(element.ReadUsers, userId) && !containsAny(element.ReadGroups, groupIds) {
				return false
			}
		case 'w':
			if !slices.Contains(element.WriteUsers, userId) && !containsAny(element.WriteGroups, groupIds) {
				return false
			}
		case 'x':
			if !slices.Contains(element.ExecuteUsers, userId) && !containsAny(element.ExecuteGroups, groupIds) {
				return false
			}
		}
	}
	return true
}

func containsAny(list []string, any []string) bool {
	for _, e := range any {
		if slices.Contains(list, e) {
			return true
		}
	}
	return false
}
