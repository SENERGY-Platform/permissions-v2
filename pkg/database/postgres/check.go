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
	"slices"
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

func (this *Database) Check(topicId string, id string, userId string, groupIds []string, rights string) (result bool, err error) {
	m, err := this.CheckMultiple(topicId, []string{id}, userId, groupIds, rights)
	if err != nil {
		return false, err
	}
	return m[id], nil

	//this in database solution would fail database.TestDistributedRights
	/*
		rightsQuery, err := getRightsQuery(rights)
		if err != nil {
			return false, err
		}
		if rightsQuery != "" {
			rightsQuery = " AND (" + rightsQuery + ")"
		}
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		row := this.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM Permissions WHERE TopicId = $1 AND Id = $2 AND (UserId = $3 OR GroupId = any($4))`+rightsQuery,
			topicId, id, userId, pq.Array(groupIds))
		err = row.Err()
		if err != nil {
			return false, err
		}
		err = row.Scan(&result)
		if err != nil {
			return false, err
		}
		return result, nil
	*/
}

func checkGroups(resource model.ResourceRights, groupIds []string, right rune) (bool, error) {
	if !slices.Contains([]rune{'r', 'w', 'x', 'a'}, right) {
		return false, errors.New("invalid rights parameter")
	}
	for _, g := range groupIds {
		if right == 'r' && resource.GroupRights[g].Read {
			return true, nil
		}
		if right == 'w' && resource.GroupRights[g].Write {
			return true, nil
		}
		if right == 'x' && resource.GroupRights[g].Execute {
			return true, nil
		}
		if right == 'a' && resource.GroupRights[g].Administrate {
			return true, nil
		}
	}
	return false, nil
}

func checkUser(resource model.ResourceRights, userId string, right rune) (bool, error) {
	switch right {
	case 'r':
		return resource.UserRights[userId].Read, nil
	case 'w':
		return resource.UserRights[userId].Write, nil
	case 'x':
		return resource.UserRights[userId].Execute, nil
	case 'a':
		return resource.UserRights[userId].Administrate, nil
	default:
		return false, errors.New("invalid rights parameter")
	}
}

func checkAccess(resource model.Resource, userId string, groupIds []string, rights string) (bool, error) {
	if rights == "" {
		return false, errors.New("invalid rights parameter")
	}
	for _, r := range rights {
		groupOk, err := checkGroups(resource.ResourceRights, groupIds, r)
		if err != nil {
			return false, err
		}
		userOk, err := checkUser(resource.ResourceRights, userId, r)
		if err != nil {
			return false, err
		}
		if !groupOk && !userOk {
			return false, nil
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
