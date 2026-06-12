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
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/SENERGY-Platform/developer-notifications/pkg/client"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller/kafka"
	"github.com/SENERGY-Platform/permissions-v2/pkg/database"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
)

type Controller struct {
	config           configuration.Config
	db               DB
	notifier         client.Client
	producerMux      sync.Mutex
	producer         map[string]kafka.Producer
	producerProvider kafka.Provider
}

type DB = database.Database

type LogNotifier struct {
	log *slog.Logger
}

func (this LogNotifier) SendMessage(message client.Message) error {
	this.log.Info(fmt.Sprintf("NOTIFIER: %#v\n", message))
	return nil
}

func NewWithDependencies(ctx context.Context, config configuration.Config, db DB, producerProvider kafka.Provider) (*Controller, error) {
	if producerProvider == nil {
		producerProvider = kafka.NewKafkaProducerProvider()
	}
	result := &Controller{config: config, db: db, producer: map[string]kafka.Producer{}, producerProvider: producerProvider}
	if config.DevNotifierUrl != "" {
		result.notifier = client.New(config.DevNotifierUrl)
	} else {
		result.notifier = LogNotifier{log: config.GetLogger()}
	}
	err := result.RetryPublishOfUnsyncedResourcesContext(ctx)
	if err != nil {
		return nil, err
	}
	result.StartSyncLoop(ctx)
	return result, nil
}

func (this *Controller) getTimeoutContext(parent ...context.Context) context.Context {
	ctxParent := context.TODO()
	if len(parent) > 0 && parent[0] != nil {
		ctxParent = parent[0]
	}
	ctx, _ := context.WithTimeout(ctxParent, 10*time.Second)
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
		this.config.GetLogger().Error("unable to send notification", "error", err)
	}
}

func (this *Controller) publishPermission(ctx context.Context, topic model.Topic, id string, permissions model.ResourcePermissions) error {
	if topic.PublishToKafkaTopic == "" || topic.PublishToKafkaTopic == "-" {
		return nil
	}
	producer, err := this.getProducer(topic)
	if err != nil {
		return err
	}
	err = producer.SendPermissions(this.getTimeoutContext(ctx), topic, id, permissions)
	if err != nil {
		return err
	}
	return nil
}

func (this *Controller) getProducer(topic model.Topic) (producer kafka.Producer, err error) {
	this.producerMux.Lock()
	defer this.producerMux.Unlock()
	if this.producer == nil {
		this.producer = map[string]kafka.Producer{}
	}
	var ok bool
	if producer, ok = this.producer[topic.PublishToKafkaTopic]; ok {
		return producer, nil
	}
	producer, err = this.producerProvider.GetProducer(this.config, topic)
	if err != nil {
		return nil, err
	}
	this.producer[topic.PublishToKafkaTopic] = producer
	return producer, nil
}

func (this *Controller) RetryPublishOfUnsyncedResources() error {
	return this.RetryPublishOfUnsyncedResourcesContext(context.TODO())
}

func (this *Controller) RetryPublishOfUnsyncedResourcesContext(ctx context.Context) error {
	list, err := this.db.ListUnsyncedResources(this.getTimeoutContext(ctx))
	if err != nil {
		return err
	}
	for _, e := range list {
		this.config.GetLogger().Info("retry to publish resource to kafka", "topicId", e.TopicId, "id", e.Id)
		topic, exists, err := this.db.GetTopic(this.getTimeoutContext(ctx), e.TopicId)
		if err != nil {
			this.config.GetLogger().Warn("RetryPublishOfUnsyncedResources: unable to get topic", "topicId", e.TopicId, "error", err)
			continue
		}
		if !exists {
			continue
		}
		err = this.publishPermission(ctx, topic, e.Id, e.ResourcePermissions)
		if err != nil {
			this.config.GetLogger().Warn("RetryPublishOfUnsyncedResources: unable to publishPermission()", "topicId", e.TopicId, "id", e.Id, "error", err)
			continue
		}
		err = this.db.MarkResourceAsSynced(this.getTimeoutContext(ctx), topic.Id, e.Id)
		if err != nil {
			this.config.GetLogger().Warn("RetryPublishOfUnsyncedResources: unable to mark resource as synced", "topicId", e.TopicId, "id", e.Id, "error", err)
		}
	}
	return nil
}

func (this *Controller) StartSyncLoop(ctx context.Context) {
	dur := this.config.SyncCheckInterval.GetDuration()
	if dur == 0 {
		return
	}
	ticker := time.NewTicker(dur)
	go func() {
		for {
			select {
			case <-ticker.C:
				this.config.GetLogger().Info(fmt.Sprint("refresh unsynced resources:", this.RetryPublishOfUnsyncedResourcesContext(ctx)))
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}
