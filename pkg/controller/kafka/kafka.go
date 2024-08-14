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

package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/segmentio/kafka-go"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func NewKafkaProducerProvider() *KafkaProducerProvider {
	return &KafkaProducerProvider{}
}

type KafkaProducerProvider struct{}

type KafkaProducer struct {
	config configuration.Config
	writer *kafka.Writer
}

func (this *KafkaProducerProvider) GetProducer(config configuration.Config, topic model.Topic) (result Producer, err error) {
	log.Println("init new producer", topic.Id)
	if topic.EnsureKafkaTopicInit {
		err = InitKafkaTopic(config.KafkaUrl, topic.EnsureKafkaTopicInitPartitionNumber, topic.PublishToKafkaTopic)
		if err != nil {
			log.Println("WARNING: unable to create topic", topic.Id, topic.PublishToKafkaTopic, err)
		}
	}
	return &KafkaProducer{config: config, writer: NewKafkaWriter(config, topic)}, nil
}

func (this *KafkaProducer) Close() (err error) {
	if this.writer != nil {
		err = errors.Join(err, this.writer.Close())
	}
	return err
}

func (this *KafkaProducer) SendPermissions(ctx context.Context, topic model.Topic, id string, permissions model.ResourcePermissions) (err error) {
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
	if this.config.Debug {
		log.Println("produce:", id, topic.PublishToKafkaTopic, key, string(temp))
	}
	return this.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: temp,
		Time:  time.Now(),
	})
}

func NewKafkaWriter(config configuration.Config, topic model.Topic) *kafka.Writer {
	var logger *log.Logger
	if config.Debug {
		logger = log.New(os.Stdout, "[KAFKA-PRODUCER] ", 0)
	} else {
		logger = log.New(io.Discard, "", 0)
	}
	writer := &kafka.Writer{
		Addr:        kafka.TCP(config.KafkaUrl),
		Topic:       topic.PublishToKafkaTopic,
		MaxAttempts: 10,
		Logger:      logger,
		BatchSize:   1,
		Balancer:    &KeySeparationBalancer{SubBalancer: &kafka.Hash{}, Seperator: "/"},
	}
	return writer
}

func InitKafkaTopic(bootstrapUrl string, partitionNumber int, topics ...string) (err error) {
	if partitionNumber == 0 {
		partitionNumber = 1
	}
	conn, err := kafka.Dial("tcp", bootstrapUrl)
	if err != nil {
		return err
	}
	defer conn.Close()
	controller, err := conn.Controller()
	if err != nil {
		return err
	}
	var controllerConn *kafka.Conn
	controllerConn, err = kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return err
	}
	defer controllerConn.Close()
	topicConfigs := []kafka.TopicConfig{}
	for _, topic := range topics {
		topicConfigs = append(topicConfigs, kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     partitionNumber,
			ReplicationFactor: 1,
			ConfigEntries: []kafka.ConfigEntry{
				{
					ConfigName:  "retention.ms",
					ConfigValue: "-1",
				},
				{
					ConfigName:  "retention.bytes",
					ConfigValue: "-1",
				},
				{
					ConfigName:  "cleanup.policy",
					ConfigValue: "compact",
				},
				{
					ConfigName:  "delete.retention.ms",
					ConfigValue: "86400000",
				},
				{
					ConfigName:  "segment.ms",
					ConfigValue: "604800000",
				},
				{
					ConfigName:  "min.cleanable.dirty.ratio",
					ConfigValue: "0.1",
				},
			},
		})
	}

	return controllerConn.CreateTopics(topicConfigs...)
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

type Command struct {
	Command string               `json:"command"`
	Id      string               `json:"id"`
	Rights  *ResourcePermissions `json:"rights"`
	Owner   string               `json:"owner,omitempty"`
}

func (this Command) String() string {
	temp, err := json.Marshal(this)
	if err != nil {
		return err.Error()
	}
	return string(temp)
}

type ResourcePermissions struct {
	UserRights           map[string]Right `json:"user_rights"`
	GroupRights          map[string]Right `json:"group_rights"`
	KeycloakGroupsRights map[string]Right `json:"keycloak_groups_rights"`
}

type Right struct {
	Read         bool `json:"read"`
	Write        bool `json:"write"`
	Execute      bool `json:"execute"`
	Administrate bool `json:"administrate"`
}

func permissionsToRights(permissions model.ResourcePermissions) *ResourcePermissions {
	result := ResourcePermissions{
		UserRights:           map[string]Right{},
		GroupRights:          map[string]Right{},
		KeycloakGroupsRights: map[string]Right{},
	}
	for user, perm := range permissions.UserPermissions {
		result.UserRights[user] = Right{
			Read:         perm.Read,
			Write:        perm.Write,
			Execute:      perm.Execute,
			Administrate: perm.Administrate,
		}
	}
	for group, perm := range permissions.RolePermissions {
		result.GroupRights[group] = Right{
			Read:         perm.Read,
			Write:        perm.Write,
			Execute:      perm.Execute,
			Administrate: perm.Administrate,
		}
	}
	for group, perm := range permissions.GroupPermissions {
		result.KeycloakGroupsRights[group] = Right{
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
		UserPermissions:  map[string]model.PermissionsMap{},
		GroupPermissions: map[string]model.PermissionsMap{},
	}
	if permissions != nil {
		for user, perm := range permissions.UserRights {
			result.UserPermissions[user] = model.PermissionsMap{
				Read:         perm.Read,
				Write:        perm.Write,
				Execute:      perm.Execute,
				Administrate: perm.Administrate,
			}
		}
		for group, perm := range permissions.GroupRights {
			result.GroupPermissions[group] = model.PermissionsMap{
				Read:         perm.Read,
				Write:        perm.Write,
				Execute:      perm.Execute,
				Administrate: perm.Administrate,
			}
		}
	}
	return result
}
