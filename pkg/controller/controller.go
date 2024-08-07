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
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller/com"
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
	com       com.Provider
	topics    map[string]TopicHandler
	topicsMux sync.RWMutex
	notifier  client.Client
	done      *kafka.Writer
}

type DB = database.Database

type LogNotifier struct{}

func (this LogNotifier) SendMessage(message client.Message) error {
	log.Printf("NOTIFIER: %#v\n", message)
	return nil
}

func NewWithDependencies(ctx context.Context, config configuration.Config, db DB, c com.Provider) (*Controller, error) {
	if config.DisableCom {
		config.HandleDoneWait = false
	}
	if c == nil {
		c = com.NewKafkaComProvider()
	}
	result := &Controller{config: config, db: db, topics: map[string]TopicHandler{}, com: c}
	if config.DevNotifierUrl != "" {
		result.notifier = client.New(config.DevNotifierUrl)
	} else {
		result.notifier = LogNotifier{}
	}
	if !config.DisableCom {
		err := result.initDoneHandling(ctx)
		if err != nil {
			return nil, err
		}
		go func() {
			<-ctx.Done()
			result.topicsMux.Lock()
			defer result.topicsMux.Unlock()
			for _, topic := range result.topics {
				log.Printf("close %v %v producer %v", topic.Id, topic.KafkaTopic, topic.Close())
			}
			result.topics = map[string]TopicHandler{}
			if result.done != nil {
				log.Printf("close done (%v) producer %v", config.DoneTopic, result.done.Close())

			}
		}()
		result.startTopicUpdateWatcher(ctx)
		err = result.refreshTopics()
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (this *Controller) HandleReceivedCommand(topic model.Topic, resource model.Resource, t time.Time) error {
	if this.config.Debug {
		log.Println("handle permissions command", topic.Id, resource.Id)
	}
	updateIgnored, err := this.db.SetResourcePermissions(this.getTimeoutContext(), resource, t, true)
	if err != nil {
		return err
	}
	if updateIgnored {
		log.Println("WARNING: old kafka command to update permissions ignored", topic.Id, resource.Id, t)
	}
	go func() {
		err := this.sendDone(topic, resource.Id)
		if err != nil {
			log.Println("ERROR: unable to send done signal", err)
		}
	}()
	return nil
}

func (this *Controller) HandleReaderError(topic model.Topic, err error) {
	log.Printf("ERROR: while consuming topic %v: %v\ntopic-config = %#v\ntry updateTopicHandling()\n", topic.KafkaTopic, err, topic)
	err = this.updateTopicHandling(topic, true)
	if err != nil {
		log.Fatal("FATAL:", err)
	}
}

func (this *Controller) getTimeoutContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	return ctx
}

func (this *Controller) notifyError(info error) {
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
