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
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/segmentio/kafka-go"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

func (this *Controller) initDoneProducer() {
	if this.config.KafkaUrl != "" && this.config.DoneTopic != "" {
		log.Printf("init done topic (%v) err=%v\n", this.config.DoneTopic, initTopic(this.config.KafkaUrl, 1, this.config.DoneTopic))
		this.done = &kafka.Writer{
			Addr:        kafka.TCP(this.config.KafkaUrl),
			Topic:       this.config.DoneTopic,
			MaxAttempts: 10,
			BatchSize:   this.config.KafkaDoneBatchSize,
		}
	}
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

func (this *Controller) newKafkaWriter(topic model.Topic) *kafka.Writer {
	var logger *log.Logger
	if this.config.Debug {
		logger = log.New(os.Stdout, "[KAFKA-PRODUCER] ", 0)
	} else {
		logger = log.New(io.Discard, "", 0)
	}
	writer := &kafka.Writer{
		Addr:        kafka.TCP(this.config.KafkaUrl),
		Topic:       topic.KafkaTopic,
		MaxAttempts: 10,
		Logger:      logger,
		BatchSize:   1,
		Balancer:    &KeySeparationBalancer{SubBalancer: &kafka.Hash{}, Seperator: "/"},
	}
	return writer
}

func (this *TopicWrapper) SendPermissions(ctx context.Context, id string, permissions model.ResourcePermissions) (err error) {
	if this.writer == nil {
		log.Println("WARNING: unable to send message to nil topic kafka writer (topic may be disabled by config.DisabledTopicConsumers)")
		return nil
	}
	cmd := Command{
		Command: "RIGHTS",
		Id:      id,
		Rights:  permissionsToRights(permissions),
	}
	var temp []byte
	temp, err = json.Marshal(cmd)
	if err != nil {
		return err
	}
	key := id + "/rights"
	if this.debug {
		log.Println("produce:", this.Id, this.KafkaTopic, key, string(temp))
	}
	return this.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: temp,
		Time:  time.Now(),
	})
}

type Command struct {
	Command string               `json:"command"`
	Id      string               `json:"id"`
	Rights  *ResourcePermissions `json:"rights"`
}

type ResourcePermissions struct {
	UserRights  map[string]Right `json:"user_rights"`
	GroupRights map[string]Right `json:"group_rights"`
}

type Right struct {
	Read         bool `json:"read"`
	Write        bool `json:"write"`
	Execute      bool `json:"execute"`
	Administrate bool `json:"administrate"`
}

func permissionsToRights(permissions model.ResourcePermissions) *ResourcePermissions {
	result := ResourcePermissions{
		UserRights:  map[string]Right{},
		GroupRights: map[string]Right{},
	}
	for user, perm := range permissions.UserPermissions {
		result.UserRights[user] = Right{
			Read:         perm.Read,
			Write:        perm.Write,
			Execute:      perm.Execute,
			Administrate: perm.Administrate,
		}
	}
	for group, perm := range permissions.GroupPermissions {
		result.GroupRights[group] = Right{
			Read:         perm.Read,
			Write:        perm.Write,
			Execute:      perm.Execute,
			Administrate: perm.Administrate,
		}
	}
	return &result
}

func rightsToPermissions(permissions *ResourcePermissions) model.ResourcePermissions {
	result := model.ResourcePermissions{
		UserPermissions:  map[string]model.Permissions{},
		GroupPermissions: map[string]model.Permissions{},
	}
	if permissions != nil {
		for user, perm := range permissions.UserRights {
			result.UserPermissions[user] = model.Permissions{
				Read:         perm.Read,
				Write:        perm.Write,
				Execute:      perm.Execute,
				Administrate: perm.Administrate,
			}
		}
		for group, perm := range permissions.GroupRights {
			result.GroupPermissions[group] = model.Permissions{
				Read:         perm.Read,
				Write:        perm.Write,
				Execute:      perm.Execute,
				Administrate: perm.Administrate,
			}
		}
	}
	return result
}

type KeySeparationBalancer struct {
	SubBalancer kafka.Balancer
	Seperator   string
}

func (this *KeySeparationBalancer) Balance(msg kafka.Message, partitions ...int) (partition int) {
	key := string(msg.Key)
	if this.Seperator != "" {
		keyParts := strings.Split(key, this.Seperator)
		key = keyParts[0]
	}
	msg.Key = []byte(key)
	return this.SubBalancer.Balance(msg, partitions...)
}
