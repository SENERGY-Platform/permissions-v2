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
	"encoding/json"
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
	done      *kafka.Writer
}

type DB = database.Database

func NewWithDependencies(ctx context.Context, config configuration.Config, db DB, disableKafka bool) (*Controller, error) {
	result := &Controller{config: config, db: db, topics: map[string]TopicWrapper{}}
	if config.DevNotifierUrl != "" {
		result.notifier = client.New(config.DevNotifierUrl)
	}
	if !disableKafka {
		result.initDoneProducer()
		go func() {
			<-ctx.Done()
			result.topicsMux.Lock()
			defer result.topicsMux.Unlock()
			for _, topic := range result.topics {
				log.Printf("close %v %v producer %v", topic.Id, topic.KafkaTopic, topic.Close())
			}
			result.topics = map[string]TopicWrapper{}
			if result.done != nil {
				log.Printf("close done (%v) producer %v", config.DoneTopic, result.done.Close())

			}
		}()
		result.startTopicUpdateWatcher(ctx)
		err := result.refreshTopics()
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (this *Controller) handleReceivedCommand(topic model.Topic, m kafka.Message) error {
	cmd := Command{}
	err := json.Unmarshal(m.Value, &cmd)
	if err != nil {
		return err
	}
	if cmd.Command != "RIGHTS" {
		return nil
	}

	resourcePermissions := model.Resource{
		Id:                  cmd.Id,
		TopicId:             topic.Id,
		ResourcePermissions: rightsToPermissions(cmd.Rights),
	}
	updateIgnored, err := this.db.SetResourcePermissions(this.getTimeoutContext(), resourcePermissions, m.Time, true)
	if err != nil {
		return err
	}
	if updateIgnored {
		log.Println("WARNING: old kafka command to update permissions ignored", resourcePermissions.TopicId, cmd.Id, m.Time)
	}

	go func() {
		err := this.sendDone(topic, cmd.Id)
		if err != nil {
			log.Println("ERROR: unable to send done signal", err)
		}
	}()

	return nil
}

func (this *Controller) handleReaderError(topic model.Topic, err error) {
	log.Printf("ERROR: while consuming topic %v: %v\ntopic-config = %#v\ntry updateTopicHandling()\n", topic.KafkaTopic, err, topic)
	err = this.updateTopicHandling(topic)
	if err != nil {
		log.Fatal("FATAL:", err)
	}
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
		if err != nil {
			log.Println("ERROR: unable to send notification", err)
		}
	}
}
