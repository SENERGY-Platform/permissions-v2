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
	"go.mongodb.org/mongo-driver/bson"
	"slices"
)

func (this *Database) CheckResourcePermissions(ctx context.Context, topicId string, id string, userId string, groupIds []string, permissions string) (bool, error) {
	m, err := this.CheckMultipleResourcePermissions(ctx, topicId, []string{id}, userId, groupIds, permissions)
	if err != nil {
		return false, err
	}
	return m[id], nil
}

func (this *Database) CheckMultipleResourcePermissions(ctx context.Context, topicId string, ids []string, userId string, groupIds []string, permissions string) (result map[string]bool, err error) {
	if ctx == nil {
		ctx, _ = getTimeoutContext()
	}
	cursor, err := this.permissionsCollection().Find(ctx, bson.M{PermissionsEntryBson.TopicId: topicId, PermissionsEntryBson.Id: bson.M{"$in": ids}})
	if err != nil {
		return result, err
	}
	result = map[string]bool{}
	for cursor.Next(context.Background()) {
		element := PermissionsEntry{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result[element.Id] = checkPermissions(userId, groupIds, element, permissions)
	}

	err = cursor.Err()
	return result, err
}

func checkPermissions(userId string, groupIds []string, element PermissionsEntry, permissions string) bool {
	for _, p := range permissions {
		switch p {
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
