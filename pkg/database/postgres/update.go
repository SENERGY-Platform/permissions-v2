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
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"log"
	"time"
)

const TxIsolation = sql.LevelSerializable

func (this *Database) SetResourcePermissions(r model.Resource, t time.Time, preventOlderUpdates bool) (updateIgnored bool, err error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	var tx *sql.Tx
	tx, err = this.db.BeginTx(ctx, &sql.TxOptions{Isolation: TxIsolation})
	if err != nil {
		return false, err
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

	if preventOlderUpdates {
		row := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM LastPermissionsKafkaTimestamp WHERE TopicId = $1 AND Id = $2 AND UnixTimestamp > $3`, r.TopicId, r.Id, t.Unix())
		err = row.Err()
		if err != nil {
			return false, err
		}
		var existingIsNewer bool
		err = row.Scan(&existingIsNewer)
		if err != nil {
			return false, err
		}
		if existingIsNewer {
			return true, nil
		}
	}

	err = removeResourcePermissions(ctx, tx, r.TopicId, r.Id)
	if err != nil {
		return false, err
	}

	for userId, rights := range r.UserRights {
		_, err = tx.ExecContext(ctx, `INSERT INTO Permissions(TopicId, Id, UserId, Read, Write, Execute, Administrate) VALUES($1, $2, $3, $4, $5, $6, $7)`,
			r.TopicId,
			r.Id,
			userId,
			rights.Read,
			rights.Write,
			rights.Execute,
			rights.Administrate)
		if err != nil {
			return false, err
		}
	}
	for groupId, rights := range r.GroupRights {
		_, err = tx.ExecContext(ctx, `INSERT INTO Permissions(TopicId, Id, GroupId, Read, Write, Execute, Administrate) VALUES($1, $2, $3, $4, $5, $6, $7)`,
			r.TopicId,
			r.Id,
			groupId,
			rights.Read,
			rights.Write,
			rights.Execute,
			rights.Administrate)
		if err != nil {
			return false, err
		}
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO LastPermissionsKafkaTimestamp(TopicId, Id, UnixTimestamp) VALUES($1, $2, $3)`,
		r.TopicId,
		r.Id,
		t.Unix())

	if err != nil {
		return false, err
	}

	return false, nil
}

func removeResourcePermissions(ctx context.Context, tx *sql.Tx, topicId string, id string) (err error) {
	_, err = tx.ExecContext(ctx, `DELETE FROM LastPermissionsKafkaTimestamp WHERE TopicId = $1 AND Id = $2;`, topicId, id)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM Permissions WHERE TopicId = $1 AND Id = $2;`, topicId, id)
	if err != nil {
		return err
	}
	return nil
}
