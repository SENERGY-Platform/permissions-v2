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
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

func (this *Mongo) ListByRights(topicId string, userId string, groupIds []string, rights string, listOptions model.ListOptions) (result []model.Resource, err error) {
	result = []model.Resource{}
	ctx, _ := getTimeoutContext()
	rightsFilter := bson.A{}
	for _, r := range rights {
		switch r {
		case 'r':
			rightsFilter = append(rightsFilter, bson.M{"$or": bson.A{bson.M{PermissionsEntryBson.ReadUsers[0]: userId}, bson.M{PermissionsEntryBson.ReadGroups[0]: bson.M{"$in": groupIds}}}})
		case 'w':
			rightsFilter = append(rightsFilter, bson.M{"$or": bson.A{bson.M{PermissionsEntryBson.WriteUsers[0]: userId}, bson.M{PermissionsEntryBson.WriteGroups[0]: bson.M{"$in": groupIds}}}})
		case 'x':
			rightsFilter = append(rightsFilter, bson.M{"$or": bson.A{bson.M{PermissionsEntryBson.ExecuteUsers[0]: userId}, bson.M{PermissionsEntryBson.ExecuteGroups[0]: bson.M{"$in": groupIds}}}})
		case 'a':
			rightsFilter = append(rightsFilter, bson.M{"$or": bson.A{bson.M{PermissionsEntryBson.AdminUsers[0]: userId}, bson.M{PermissionsEntryBson.AdminGroups[0]: bson.M{"$in": groupIds}}}})
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
	opt.SetSort(bson.D{{PermissionsEntryBson.Id, 1}})

	filter := bson.M{PermissionsEntryBson.TopicId: topicId, "$and": rightsFilter}
	cursor, err := this.rightsCollection().Find(ctx, filter, opt)
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
