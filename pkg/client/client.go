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

package client

import (
	"context"
	"github.com/SENERGY-Platform/permissions-v2/pkg/api"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller"
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller/com"
	"github.com/SENERGY-Platform/permissions-v2/pkg/database"
	"github.com/SENERGY-Platform/permissions-v2/pkg/database/mock"
	"github.com/SENERGY-Platform/permissions-v2/pkg/database/mongo"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/permissions-v2/pkg/tests/docker"
	"sync"
)

type Client interface {
	api.Controller
}

func New(serverUrl string) (client Client) {
	return &Impl{serverUrl: serverUrl}
}

func NewTestClient(ctx context.Context) (client Client, err error) {
	return NewTestClientFromDb(ctx, mock.New())
}

func NewTestClientWithDocker(ctx context.Context, wg *sync.WaitGroup) (client Client, err error) {
	port, _, err := docker.MongoDB(ctx, wg)
	if err != nil {
		return nil, err
	}
	db, err := mongo.New(configuration.Config{
		MongoUrl:                   "mongodb://localhost:" + port,
		MongoDatabase:              "permissions",
		MongoPermissionsCollection: "permissions",
		MongoTopicsCollection:      "topics",
	})
	if err != nil {
		return nil, err
	}
	return NewTestClientFromDb(ctx, db)
}

func NewTestClientFromDb(ctx context.Context, db database.Database) (client Client, err error) {
	return controller.NewWithDependencies(ctx,
		configuration.Config{},
		db,
		com.NewBypassProvider(),
		false)
}

type Impl struct {
	serverUrl string
}

func (this *Impl) ListTopics(token string, options model.ListOptions) (result []model.Topic, err error, code int) {
	//TODO implement me
	panic("implement me")
}

func (this *Impl) GetTopic(token string, id string) (result model.Topic, err error, code int) {
	//TODO implement me
	panic("implement me")
}

func (this *Impl) RemoveTopic(token string, id string) (err error, code int) {
	//TODO implement me
	panic("implement me")
}

func (this *Impl) SetTopic(token string, topic model.Topic) (result model.Topic, err error, code int) {
	//TODO implement me
	panic("implement me")
}

func (this *Impl) CheckPermission(token string, topicId string, id string, permissions string) (access bool, err error, code int) {
	//TODO implement me
	panic("implement me")
}

func (this *Impl) CheckMultiplePermissions(token string, topicId string, ids []string, permissions string) (access map[string]bool, err error, code int) {
	//TODO implement me
	panic("implement me")
}

func (this *Impl) ListAccessibleResourceIds(token string, topicId string, permissions string, options model.ListOptions) (ids []string, err error, code int) {
	//TODO implement me
	panic("implement me")
}

func (this *Impl) ListResourcesWithAdminPermission(token string, topicId string, options model.ListOptions) (result []model.Resource, err error, code int) {
	//TODO implement me
	panic("implement me")
}

func (this *Impl) GetResource(token string, topicId string, id string) (result model.Resource, err error, code int) {
	//TODO implement me
	panic("implement me")
}

func (this *Impl) SetPermission(token string, topicId string, id string, permissions model.ResourcePermissions, options model.SetPermissionOptions) (result model.ResourcePermissions, err error, code int) {
	//TODO implement me
	panic("implement me")
}
