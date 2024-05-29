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

const SqlCreateTopicsTable = `CREATE TABLE IF NOT EXISTS Topics (
	Id 						TEXT NOT NULL PRIMARY KEY,
	KafkaTopic				TEXT NOT NULL UNIQUE,
	KafkaConsumerGroup		TEXT NOT NULL,
	InitialGroupRights		JSON
);`

const SqlCreatePermissionsTable = `CREATE TABLE IF NOT EXISTS Permissions (
	TopicId 		TEXT NOT NULL,
	Id      		TEXT NOT NULL,
	UserId 			TEXT,
	GroupId 		TEXT,
	Read     		BOOLEAN,
	Write       	BOOLEAN,
	Execute      	BOOLEAN,
	Administrate 	BOOLEAN,
	PRIMARY KEY (TopicId, Id, UserId, GroupId),
	constraint PermissionsUserGroupXorConstraint 
        check (        (UserId is null or GroupId is null) 
               and not (UserId is null and GroupId is null) )
);
CREATE INDEX IF NOT EXISTS PermissionsIdTopicIndex ON Permissions (TopicId, Id);
CREATE INDEX IF NOT EXISTS PermissionsUserIndex ON Permissions (TopicId, UserId);
CREATE INDEX IF NOT EXISTS PermissionsGroupIndex ON Permissions (TopicId, GroupId);
`

const SqlCreateLastKafkaTimestampTable = `CREATE TABLE IF NOT EXISTS LastPermissionsKafkaTimestamp (
	TopicId 		TEXT NOT NULL,
	Id      		TEXT NOT NULL,
	UnixTimestamp 	BIGINT,
	PRIMARY KEY (TopicId, Id)
);`

func (this *Database) init() error {
	_, err := this.db.Exec(SqlCreateTopicsTable)
	if err != nil {
		return err
	}

	_, err = this.db.Exec(SqlCreatePermissionsTable)
	if err != nil {
		return err
	}

	_, err = this.db.Exec(SqlCreateLastKafkaTimestampTable)
	if err != nil {
		return err
	}

	return nil
}
