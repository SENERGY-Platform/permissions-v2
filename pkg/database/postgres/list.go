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
	"github.com/lib/pq"
	"log"
	"slices"
	"strconv"
	"strings"
	"time"
)

func (this *Database) ListByRights(topicId string, userId string, groupIds []string, rights string, options model.ListOptions) (result []model.Resource, err error) {
	result = []model.Resource{}
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
	ids, err = listIdsByRights(ctx, tx, topicId, userId, groupIds, rights, options)
	if err != nil {
		return nil, err
	}

	return listByIds(ctx, tx, topicId, ids)
}

func listByIds(ctx context.Context, db DbTxAbstract, topicId string, ids []string) (result []model.Resource, err error) {
	query := `SELECT Id,UserId,GroupId,Read,Write,Execute,Administrate FROM Permissions WHERE TopicId = $1 AND Id = any($2)`
	rows, err := db.QueryContext(ctx, query, topicId, pq.Array(ids))
	if err != nil {
		return nil, err
	}

	return rowsToResources(rows, topicId)
}

func rowsToResources(rows *sql.Rows, topicId string) (result []model.Resource, err error) {
	result = []model.Resource{}
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
		resource := model.Resource{
			TopicId: topicId,
			ResourceRights: model.ResourceRights{
				UserRights:  map[string]model.Right{},
				GroupRights: map[string]model.Right{},
			},
		}
		var right model.Right
		var userId sql.NullString
		var groupId sql.NullString
		err = rows.Scan(&resource.Id, &userId, &groupId, &right.Read, &right.Write, &right.Execute, &right.Administrate)
		if err != nil {
			return result, err
		}
		if userId.Valid {
			resource.UserRights[userId.String] = right
		}
		if groupId.Valid {
			resource.GroupRights[groupId.String] = right
		}
		result = mergeResourceList(result, resource)
	}
	slices.SortFunc(result, func(a, b model.Resource) int {
		return strings.Compare(a.Id, b.Id)
	})
	return result, err
}

func mergeResourceList(list []model.Resource, element model.Resource) (result []model.Resource) {
	found := false
	for _, e := range list {
		if e.Id == element.Id {
			found = true
			for g, r := range element.GroupRights {
				e.GroupRights[g] = r
			}
			for u, r := range element.UserRights {
				e.UserRights[u] = r
			}
		}
		result = append(result, e)
	}
	if !found {
		result = append(result, element)
	}
	return result
}

func (this *Database) ListIdsByRights(topicId string, userId string, groupIds []string, rights string, options model.ListOptions) ([]string, error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	return listIdsByRights(ctx, this.db, topicId, userId, groupIds, rights, options)
}

func listIdsByRights(ctx context.Context, db DbTxAbstract, topicId string, userId string, groupIds []string, rights string, options model.ListOptions) (result []string, err error) {
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

	listQuery := " ORDER BY Id"
	if options.Limit > 0 {
		listQuery = listQuery + " LIMIT " + strconv.FormatInt(options.Limit, 10)
	}
	if options.Offset > 0 {
		listQuery = listQuery + " OFFSET " + strconv.FormatInt(options.Offset, 10)
	}

	query := `SELECT DISTINCT Id FROM Permissions WHERE TopicId = $1 AND (UserId = $2 OR GroupId = any($3))` + rightsQuery + listQuery
	rows, err := db.QueryContext(ctx, query, topicId, userId, pq.Array(groupIds))
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
