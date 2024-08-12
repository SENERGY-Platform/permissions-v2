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
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/database/mongo"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"time"
)

type Database interface {
	SetResourcePermissions(ctx context.Context, r model.Resource, t time.Time, preventOlderUpdates bool) (updateIgnored bool, err error)

	AdminListResourceIds(ctx context.Context, topicId string, options model.ListOptions) ([]string, error)
	AdminListResources(ctx context.Context, topicId string, listOptions model.ListOptions) (result []model.Resource, err error)

	ListResourcesByPermissions(ctx context.Context, topicId string, userId string, groupIds []string, options model.ListOptions, permissions ...model.Permission) (result []model.Resource, err error)
	ListResourceIdsByPermissions(ctx context.Context, topicId string, userId string, groupIds []string, options model.ListOptions, permissions ...model.Permission) ([]string, error)

	GetResource(ctx context.Context, topicId string, id string, options model.GetOptions) (resource model.Resource, err error)
	DeleteResource(ctx context.Context, topicId string, id string) error

	CheckMultipleResourcePermissions(ctx context.Context, topicId string, ids []string, userId string, groupIds []string, permissions ...model.Permission) (result map[string]bool, err error)
	CheckResourcePermissions(ctx context.Context, topicId string, id string, userId string, groupIds []string, permissions ...model.Permission) (result bool, err error)

	SetTopic(ctx context.Context, topic model.Topic) error
	GetTopic(ctx context.Context, id string) (result model.Topic, exists bool, err error)
	ListTopics(ctx context.Context, listOptions model.ListOptions) (result []model.Topic, err error)
	DeleteTopic(ctx context.Context, id string) error
}

func New(config configuration.Config) (Database, error) {
	return mongo.New(config)
}
