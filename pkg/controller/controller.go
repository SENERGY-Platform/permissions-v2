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
	"context"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
	"github.com/segmentio/kafka-go"
	"log"
	"sync"
)

type Controller struct {
	config    configuration.Config
	db        DB
	topics    map[string]TopicWrapper
	topicsMux sync.Mutex
}

func NewWithDependencies(ctx context.Context, config configuration.Config, db DB) *Controller {
	result := &Controller{config: config, db: db, topics: map[string]TopicWrapper{}}
	go func() {
		<-ctx.Done()
		result.topicsMux.Lock()
		defer result.topicsMux.Unlock()
		for _, topic := range result.topics {
			log.Printf("close %v %v producer %v", topic.Id, topic.KafkaTopic, topic.Close())
		}
		result.topics = map[string]TopicWrapper{}
	}()
	return result
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

func (this *Controller) handleReceivedCommand(topic model.Topic, m kafka.Message) error {
	//TODO implement me
	panic("implement me")
}

func (this *Controller) handleReaderError(topic model.Topic, err error) {
	//TODO: try to reconnect
	log.Fatalf("ERROR: while consuming topic %v: %v\ntopic-config = %#v\n", topic.KafkaTopic, err, topic)
}
