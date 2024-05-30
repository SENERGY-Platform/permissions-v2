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
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"time"
)

type Database interface {
	SetResourcePermissions(r model.Resource, t time.Time, preventOlderUpdates bool) (updateIgnored bool, err error)
	ListByRights(topicId string, userId string, groupIds []string, rights string, options model.ListOptions) (result []model.Resource, err error)
	ListIdsByRights(topicId string, userId string, groupIds []string, rights string, options model.ListOptions) ([]string, error)
	CheckMultiple(topicId string, ids []string, userId string, groupIds []string, rights string) (result map[string]bool, err error)
}
