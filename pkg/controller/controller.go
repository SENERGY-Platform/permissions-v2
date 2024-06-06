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
	"github.com/SENERGY-Platform/developer-notifications/pkg/client"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/database"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/segmentio/kafka-go"
	"log"
	"sync"
	"time"
)

type Controller struct {
	config    configuration.Config
	db        DB
	topics    map[string]TopicWrapper
	topicsMux sync.RWMutex
	notifier  client.Client
}

type DB = database.Database

func NewWithDependencies(ctx context.Context, config configuration.Config, db DB) (*Controller, error) {
	result := &Controller{config: config, db: db, topics: map[string]TopicWrapper{}, notifier: client.New(config.DevNotifierUrl)}
	go func() {
		<-ctx.Done()
		result.topicsMux.Lock()
		defer result.topicsMux.Unlock()
		for _, topic := range result.topics {
			log.Printf("close %v %v producer %v", topic.Id, topic.KafkaTopic, topic.Close())
		}
		result.topics = map[string]TopicWrapper{}
	}()
	result.startTopicUpdateWatcher(ctx)
	return result, result.refreshTopics()
}

func (this *Controller) handleReceivedCommand(topic model.Topic, m kafka.Message) error {
	//TODO implement me
	panic("implement me")
}

func (this *Controller) handleReaderError(topic model.Topic, err error) {
	//TODO: try to reconnect
	log.Fatalf("ERROR: while consuming topic %v: %v\ntopic-config = %#v\n", topic.KafkaTopic, err, topic)
}

func (this *Controller) getTimeoutContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	return ctx
}

func (this *Controller) notifyError(info error) {
	if this.notifier != nil && this.config.DevNotifierUrl != "" {
		err := this.notifier.SendMessage(client.Message{
			Sender: "github.com/SENERGY-Platform/permissions-v2",
			Title:  "PermissionsV2 Error",
			Tags:   []string{"error", "permissions"},
			Body:   info.Error(),
		})
		log.Println("ERROR: unable to send notification", err)
	}
}
