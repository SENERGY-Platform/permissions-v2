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

package api

import (
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
)

type Controller interface {
	PermissionsCheckInterface
	AdminInterface
	PermissionsManagementInterface
}

type AdminInterface interface {
	ListTopics(token string, options model.ListOptions) (result []model.Topic, err error, code int)
	GetTopic(token string, id string) (result model.Topic, err error, code int)
	RemoveTopic(token string, id string) (err error, code int)
	SetTopic(token string, topic model.Topic) (result model.Topic, err error, code int)
}

type PermissionsCheckInterface interface {
	CheckPermission(token string, topicId string, id string, permissions ...model.Permission) (access bool, err error, code int)
	CheckMultiplePermissions(token string, topicId string, ids []string, permissions ...model.Permission) (access map[string]bool, err error, code int)
	ListAccessibleResourceIds(token string, topicId string, options model.ListOptions, permissions ...model.Permission) (ids []string, err error, code int)
}

type PermissionsManagementInterface interface {
	ListResourcesWithAdminPermission(token string, topicId string, options model.ListOptions) (result []model.Resource, err error, code int)
	GetResource(token string, topicId string, id string) (result model.Resource, err error, code int)
	RemoveResource(token string, topicId string, id string) (err error, code int)
	SetPermission(token string, topicId string, id string, permissions model.ResourcePermissions, options model.SetPermissionOptions) (result model.ResourcePermissions, err error, code int)
}
