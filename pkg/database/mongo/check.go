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
	"slices"
)

func (this *Database) CheckResourcePermissions(ctx context.Context, topicId string, id string, userId string, roleIds []string, groupIds []string, permissions ...model.Permission) (bool, error) {
	m, err := this.CheckMultipleResourcePermissions(ctx, topicId, []string{id}, userId, roleIds, groupIds, permissions...)
	if err != nil {
		return false, err
	}
	return m[id], nil
}

func (this *Database) CheckMultipleResourcePermissions(ctx context.Context, topicId string, ids []string, userId string, roleIds []string, groupIds []string, permissions ...model.Permission) (result map[string]bool, err error) {
	if ctx == nil {
		ctx, _ = getTimeoutContext()
	}
	cursor, err := this.permissionsCollection().Find(ctx, bson.M{PermissionsEntryBson.TopicId: topicId, PermissionsEntryBson.Id: bson.M{"$in": ids}})
	if err != nil {
		return result, err
	}
	defer cursor.Close(context.Background())
	result = map[string]bool{}
	for cursor.Next(context.Background()) {
		element := PermissionsEntry{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result[element.Id] = checkPermissions(userId, roleIds, groupIds, element, permissions...)
	}

	err = cursor.Err()
	return result, err
}

func checkPermissions(userId string, roleIds []string, groupIds []string, element PermissionsEntry, permission ...model.Permission) bool {
	for _, p := range permission {
		switch p {
		case model.Administrate:
			if !slices.Contains(element.AdminUsers, userId) && !containsAny(element.AdminGroups, groupIds) && !containsAny(element.AdminRoles, roleIds) {
				return false
			}
		case model.Read:
			if !slices.Contains(element.ReadUsers, userId) && !containsAny(element.ReadGroups, groupIds) && !containsAny(element.ReadRoles, roleIds) {
				return false
			}
		case model.Write:
			if !slices.Contains(element.WriteUsers, userId) && !containsAny(element.WriteGroups, groupIds) && !containsAny(element.WriteRoles, roleIds) {
				return false
			}
		case model.Execute:
			if !slices.Contains(element.ExecuteUsers, userId) && !containsAny(element.ExecuteGroups, groupIds) && !containsAny(element.ExecuteRoles, roleIds) {
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
