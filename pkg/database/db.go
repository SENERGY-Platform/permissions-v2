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
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	_ "github.com/lib/pq"
)

func New(config configuration.Config) (*Database, error) {
	db, err := sql.Open("postgres", config.PostgresConnStr)
	if err != nil {
		return nil, err
	}
	result := &Database{db: db, config: config}
	return result, result.init()
}

type Database struct {
	db     *sql.DB
	config configuration.Config
}

type DbTxAbstract interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}
