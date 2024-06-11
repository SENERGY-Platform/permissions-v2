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

package com

import (
	"context"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"log"
	"time"
)

func NewBypassProvider() *BypassProvider {
	return &BypassProvider{}
}

type BypassProvider struct{}

func (this *BypassProvider) Get(config configuration.Config, topic model.Topic, readHandler ReadHandler) (Com, error) {
	log.Println("init new bypass com", topic.Id)
	return &Bypass{readHandler: readHandler}, nil
}

type Bypass struct {
	readHandler ReadHandler
}

func (this *Bypass) Close() error {
	return nil
}

func (this *Bypass) SendPermissions(ctx context.Context, topic model.Topic, id string, permissions model.ResourcePermissions) (err error) {
	return this.readHandler.HandleReceivedCommand(topic, model.Resource{
		Id:                  id,
		TopicId:             topic.Id,
		ResourcePermissions: permissions,
	}, time.Now())
}
