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
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller/com"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/service-commons/pkg/donewait"
	kafka2 "github.com/SENERGY-Platform/service-commons/pkg/kafka"
	"github.com/segmentio/kafka-go"
	"log"
	"time"
)

func (this *Controller) optionalWait(wait bool, kafkaTopic string, id string) func() error {
	f := func() error { return nil }
	if wait && this.config.HandleDoneWait {
		ctx, _ := context.WithTimeout(context.Background(), time.Minute)
		f = donewait.AsyncWait(ctx, donewait.DoneMsg{
			ResourceKind: kafkaTopic,
			ResourceId:   id,
			Command:      "RIGHTS",
			Handler:      "github.com/SENERGY-Platform/permissions-v2",
		}, nil)
	}
	return f
}

func (this *Controller) initDoneHandling(ctx context.Context) error {
	if this.config.KafkaUrl != "" && this.config.DoneTopic != "" {
		log.Printf("init done topic (%v) err=%v\n", this.config.DoneTopic, com.InitKafkaTopic(this.config.KafkaUrl, 1, this.config.DoneTopic))
		this.done = &kafka.Writer{
			Addr:        kafka.TCP(this.config.KafkaUrl),
			Topic:       this.config.DoneTopic,
			MaxAttempts: 10,
			BatchSize:   this.config.KafkaDoneBatchSize,
		}
		if this.config.HandleDoneWait {
			err := donewait.StartDoneWaitListener(ctx, kafka2.Config{
				KafkaUrl:    this.config.KafkaUrl,
				StartOffset: kafka.LastOffset,
				Debug:       this.config.Debug,
			}, []string{this.config.DoneTopic}, nil)
			if err != nil {
				return err
			}
		}
	} else {
		log.Println("no done topic handling")
	}
	return nil
}

type Done struct {
	ResourceKind string `json:"resource_kind"`
	ResourceId   string `json:"resource_id"`
	Handler      string `json:"handler"` // == github.com/SENERGY-Platform/permission-search
	Command      string `json:"command"` // PUT | DELETE | RIGHTS
}

func (this *Controller) sendDone(topic model.Topic, id string) interface{} {
	if this.done == nil {
		return nil
	}
	msg := Done{
		ResourceKind: topic.KafkaTopic,
		ResourceId:   id,
		Handler:      "github.com/SENERGY-Platform/permissions-v2",
		Command:      "RIGHTS",
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return this.done.WriteMessages(this.getTimeoutContext(), kafka.Message{
		Key:   []byte(msg.ResourceId),
		Value: payload,
		Time:  time.Now(),
	})
}
