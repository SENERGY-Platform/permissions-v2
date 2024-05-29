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

package database

import (
	"context"
	"database/sql"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
)

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
	for rows.Next() {
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
