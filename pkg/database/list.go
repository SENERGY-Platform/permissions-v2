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
	"errors"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"log"
	"strconv"
	"time"
)

func (this *Database) ListByRights(topicId string, userId string, groupId string, rights string, options model.ListOptions) (result []model.Resource, err error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	tx, err := this.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			rbErr := tx.Rollback()
			if rbErr != nil {
				log.Println("ERROR: unable to rollback", rbErr)
			}
		} else {
			err = tx.Commit()
			if err != nil {
				log.Println("ERROR: unable to commit", err)
			}
		}
	}()
	var ids []string
	ids, err = listIdsByRights(ctx, tx, topicId, userId, groupId, rights, options)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		resource := model.Resource{
			Id:      id,
			TopicId: topicId,
		}
		resource.ResourceRights, err = getResourceRights(ctx, tx, topicId, id)
		if err != nil {
			return nil, err
		}
		result = append(result, resource)
	}
	return result, err
}

func (this *Database) ListIdsByRights(topicId string, userId string, groupId string, rights string, options model.ListOptions) ([]string, error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	return listIdsByRights(ctx, this.db, topicId, userId, groupId, rights, options)
}

func listIdsByRights(ctx context.Context, db DbTxAbstract, topicId string, userId string, groupId string, rights string, options model.ListOptions) (result []string, err error) {
	if rights == "" {
		return []string{}, errors.New("invalid rights parameter")
	}
	rightsQuery := ""
	for _, r := range rights {
		switch r {
		case 'r':
			rightsQuery = rightsQuery + " AND Read = true"
		case 'w':
			rightsQuery = rightsQuery + " AND Write = true"
		case 'x':
			rightsQuery = rightsQuery + " AND Execute = true"
		case 'a':
			rightsQuery = rightsQuery + " AND Administrate = true"
		default:
			return []string{}, errors.New("invalid rights parameter")
		}
	}

	listQuery := ""
	if options.Limit > 0 {
		listQuery = listQuery + " LIMIT " + strconv.FormatInt(options.Limit, 10)
	}
	if options.Offset > 0 {
		listQuery = listQuery + " OFFSET " + strconv.FormatInt(options.Offset, 10)
	}
	listQuery = listQuery + " ORDER BY Id"

	rows, err := db.QueryContext(ctx, `SELECT DISTINCT Id FROM Permissions WHERE TopicId = $1 AND (UserId = $2 OR GroupId = $3)`+rightsQuery+listQuery, topicId, userId, groupId)
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			return []string{}, err
		}
		result = append(result, id)
	}
	return result, err
}
