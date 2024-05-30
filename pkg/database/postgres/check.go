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

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"time"
)

func (this *Database) CheckMultiple(topicId string, ids []string, userId string, groupIds []string, rights string) (result map[string]bool, err error) {
	result = map[string]bool{}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	resources, err := listByIds(ctx, this.db, topicId, ids)
	if err != nil {
		return result, err
	}
	for _, resource := range resources {
		result[resource.Id], err = checkAccess(resource, userId, groupIds, rights)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func checkAccess(resource model.Resource, userId string, groupIds []string, rights string) (bool, error) {
	if rights == "" {
		return false, errors.New("invalid rights parameter")
	}
	for _, r := range rights {
		switch r {
		case 'r':
			if resource.UserRights[userId].Read {
				continue
			}
			for _, g := range groupIds {
				if resource.GroupRights[g].Read {
					continue
				}
			}
			return false, nil
		case 'w':
			if resource.UserRights[userId].Write {
				continue
			}
			for _, g := range groupIds {
				if resource.GroupRights[g].Write {
					continue
				}
			}
			return false, nil
		case 'x':
			if resource.UserRights[userId].Execute {
				continue
			}
			for _, g := range groupIds {
				if resource.GroupRights[g].Execute {
					continue
				}
			}
			return false, nil
		case 'a':
			if resource.UserRights[userId].Administrate {
				continue
			}
			for _, g := range groupIds {
				if resource.GroupRights[g].Administrate {
					continue
				}
			}
			return false, nil
		default:
			return false, errors.New("invalid rights parameter")
		}
	}
	return true, nil
}

func getResourceRights(ctx context.Context, db DbTxAbstract, topicId string, id string) (result model.ResourceRights, err error) {
	result = model.ResourceRights{
		UserRights:  map[string]model.Right{},
		GroupRights: map[string]model.Right{},
	}
	rows, err := db.QueryContext(ctx, `SELECT UserId,GroupId,Read,Write,Execute,Administrate FROM Permissions WHERE TopicId = $1 AND Id = $2`, topicId, id)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	err = rows.Err()
	if err != nil {
		return result, err
	}
	for rows.Next() {
		err = rows.Err()
		if err != nil {
			return result, err
		}
		var right model.Right
		var userId sql.NullString
		var groupId sql.NullString
		err = rows.Scan(&userId, &groupId, &right.Read, &right.Write, &right.Execute, &right.Administrate)
		if err != nil {
			return result, err
		}
		if userId.Valid {
			result.UserRights[userId.String] = right
		}
		if groupId.Valid {
			result.GroupRights[groupId.String] = right
		}
	}
	return result, nil
}
