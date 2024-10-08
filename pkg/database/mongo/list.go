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
	"runtime/debug"
)

func (this *Database) GetResource(ctx context.Context, topicId string, id string, options model.GetOptions) (resource model.Resource, err error) {
	result := this.permissionsCollection().FindOne(ctx, bson.M{PermissionsEntryBson.TopicId: topicId, PermissionsEntryBson.Id: id})
	err = result.Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return resource, model.ErrNotFound
	}
	if err != nil {
		return resource, err
	}
	entry := PermissionsEntry{}
	err = result.Decode(&entry)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return resource, model.ErrNotFound
	}
	if err != nil {
		return resource, err
	}
	if options.CheckPermission {
		if !checkPermissions(options.UserId, options.RoleIds, options.GroupIds, entry, options.Permissions...) {
			return resource, model.PermissionCheckFailed
		}
	}
	return entry.ToResource(), nil
}

func (this *Database) AdminListResourceIds(ctx context.Context, topicId string, listOptions model.ListOptions) (result []string, err error) {
	result = []string{}
	if ctx == nil {
		ctx, _ = getTimeoutContext()
	}

	opt := options.Find()
	if listOptions.Limit > 0 {
		opt.SetLimit(listOptions.Limit)
	}
	if listOptions.Offset > 0 {
		opt.SetSkip(listOptions.Offset)
	}
	opt.SetSort(bson.D{{PermissionsEntryBson.Id, 1}})

	filter := bson.M{PermissionsEntryBson.TopicId: topicId}
	if listOptions.Ids != nil {
		filter[PermissionsEntryBson.Id] = bson.M{"$in": listOptions.Ids}
	}
	cursor, err := this.permissionsCollection().Find(ctx, filter, opt)
	if err != nil {
		return result, err
	}
	defer cursor.Close(context.Background())
	for cursor.Next(context.Background()) {
		element := PermissionsEntry{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result = append(result, element.Id)
	}

	err = cursor.Err()
	return result, err
}

func (this *Database) AdminListResources(ctx context.Context, topicId string, listOptions model.ListOptions) (result []model.Resource, err error) {
	result = []model.Resource{}
	if ctx == nil {
		ctx, _ = getTimeoutContext()
	}

	opt := options.Find()
	if listOptions.Limit > 0 {
		opt.SetLimit(listOptions.Limit)
	}
	if listOptions.Offset > 0 {
		opt.SetSkip(listOptions.Offset)
	}
	opt.SetSort(bson.D{{PermissionsEntryBson.Id, 1}})

	filter := bson.M{PermissionsEntryBson.TopicId: topicId}
	if listOptions.Ids != nil {
		filter[PermissionsEntryBson.Id] = bson.M{"$in": listOptions.Ids}
	}
	cursor, err := this.permissionsCollection().Find(ctx, filter, opt)
	if err != nil {
		debug.PrintStack()
		return result, err
	}
	defer cursor.Close(context.Background())
	for cursor.Next(context.Background()) {
		element := PermissionsEntry{}
		err = cursor.Decode(&element)
		if err != nil {
			debug.PrintStack()
			return nil, err
		}
		result = append(result, element.ToResource())
	}

	err = cursor.Err()
	if err != nil {
		debug.PrintStack()
	}
	return result, err
}

func (this *Database) ListResourceIdsByPermissions(ctx context.Context, topicId string, userId string, roleIds []string, groupIds []string, options model.ListOptions, permissions ...model.Permission) ([]string, error) {
	temp, err := this.ListResourcesByPermissions(ctx, topicId, userId, roleIds, groupIds, options, permissions...)
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, e := range temp {
		result = append(result, e.Id)
	}
	return result, err
}

func (this *Database) ListResourcesByPermissions(ctx context.Context, topicId string, userId string, roleIds []string, groupIds []string, listOptions model.ListOptions, permissions ...model.Permission) (result []model.Resource, err error) {
	result = []model.Resource{}
	if ctx == nil {
		ctx, _ = getTimeoutContext()
	}
	if groupIds == nil {
		groupIds = []string{}
	}
	if roleIds == nil {
		roleIds = []string{}
	}
	permissionsFilter := bson.A{}
	for _, r := range permissions {
		switch r {
		case 'r':
			permissionsFilter = append(permissionsFilter, bson.M{"$or": bson.A{bson.M{PermissionsEntryBson.ReadUsers[0]: userId}, bson.M{PermissionsEntryBson.ReadGroups[0]: bson.M{"$in": groupIds}}, bson.M{PermissionsEntryBson.ReadRoles[0]: bson.M{"$in": roleIds}}}})
		case 'w':
			permissionsFilter = append(permissionsFilter, bson.M{"$or": bson.A{bson.M{PermissionsEntryBson.WriteUsers[0]: userId}, bson.M{PermissionsEntryBson.WriteGroups[0]: bson.M{"$in": groupIds}}, bson.M{PermissionsEntryBson.WriteRoles[0]: bson.M{"$in": roleIds}}}})
		case 'x':
			permissionsFilter = append(permissionsFilter, bson.M{"$or": bson.A{bson.M{PermissionsEntryBson.ExecuteUsers[0]: userId}, bson.M{PermissionsEntryBson.ExecuteGroups[0]: bson.M{"$in": groupIds}}, bson.M{PermissionsEntryBson.ExecuteRoles[0]: bson.M{"$in": roleIds}}}})
		case 'a':
			permissionsFilter = append(permissionsFilter, bson.M{"$or": bson.A{bson.M{PermissionsEntryBson.AdminUsers[0]: userId}, bson.M{PermissionsEntryBson.AdminGroups[0]: bson.M{"$in": groupIds}}, bson.M{PermissionsEntryBson.AdminRoles[0]: bson.M{"$in": roleIds}}}})
		default:
			return []model.Resource{}, errors.New("invalid permissions parameter")
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

	filter := bson.M{PermissionsEntryBson.TopicId: topicId, "$and": permissionsFilter}
	cursor, err := this.permissionsCollection().Find(ctx, filter, opt)
	if err != nil {
		return result, err
	}
	defer cursor.Close(context.Background())
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
