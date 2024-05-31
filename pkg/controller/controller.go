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

package controller

import (
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
)

type Controller struct {
	config configuration.Config
	db     DB
}

func NewWithDependencies(config configuration.Config, db DB) *Controller {
	return &Controller{config: config, db: db}
}

func (this *Controller) ListTopics(token jwt.Token, options model.ListOptions) (result []model.Topic, err error, code int) {
	//TODO implement me
	panic("implement me")
}

func (this *Controller) GetTopic(token jwt.Token, id string) (result model.Topic, err error, code int) {
	//TODO implement me
	panic("implement me")
}

func (this *Controller) RemoveTopic(token jwt.Token, id string) (err error, code int) {
	//TODO implement me
	panic("implement me")
}

func (this *Controller) SetTopic(token jwt.Token, topic model.Topic) (result model.Topic, err error, code int) {
	//TODO implement me
	panic("implement me")
}

func (this *Controller) CheckPermission(token jwt.Token, topicId string, id string, permissions string) (access bool, err error, code int) {
	//must handle id-modifiers
	//TODO implement me
	panic("implement me")
}

func (this *Controller) CheckMultiplePermissions(token jwt.Token, topicId string, ids []string, permissions string) (access map[string]bool, err error, code int) {
	//must handle id-modifiers
	//TODO implement me
	panic("implement me")
}

func (this *Controller) ListAccessibleResourceIds(token jwt.Token, topicId string, permissions string, options model.ListOptions) (ids []string, err error, code int) {
	//TODO implement me
	panic("implement me")
}

func (this *Controller) ListResourcesWithAdminPermission(token jwt.Token, topicId string, options model.ListOptions) (result []model.Resource, err error, code int) {
	//TODO implement me
	panic("implement me")
}

func (this *Controller) GetResource(token jwt.Token, topicId string, id string) (result model.Resource, err error, code int) {
	//must handle id-modifiers
	//TODO implement me
	panic("implement me")
}

func (this *Controller) SetPermission(token jwt.Token, topicId string, id string, permissions model.ResourcePermissions) (result model.ResourcePermissions, err error, code int) {
	//TODO implement me
	panic("implement me")
}
