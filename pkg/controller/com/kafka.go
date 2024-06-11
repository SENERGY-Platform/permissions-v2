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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/segmentio/kafka-go"
	"io"
	"log"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

func NewKafkaComProvider() *KafkaComProvider {
	return &KafkaComProvider{}
}

type KafkaComProvider struct{}

type KafkaCom struct {
	config configuration.Config
	writer *kafka.Writer
	reader *kafka.Reader
}

func (this *KafkaComProvider) Get(config configuration.Config, topic model.Topic, readHandler ReadHandler) (result Com, err error) {
	if topic.NoCqrs {
		return NewBypassProvider().Get(config, topic, readHandler)
	}
	log.Println("init new com", topic.Id)
	if topic.EnsureTopicInit {
		err = InitKafkaTopic(config.KafkaUrl, topic.EnsureTopicInitPartitionNumber, topic.KafkaTopic)
		if err != nil {
			log.Println("WARNING: unable to create topic", topic.Id, topic.KafkaTopic, err)
		}
	}
	temp := &KafkaCom{config: config, writer: NewKafkaWriter(config, topic)}
	if !slices.Contains(config.DisabledTopicConsumers, topic.Id) && !slices.Contains(config.DisabledTopicConsumers, topic.KafkaTopic) {
		temp.reader, err = NewKafkaReader(config, topic, readHandler)
		if err != nil {
			temp.writer.Close()
			return nil, err
		}
	}
	return temp, nil
}

func (this *KafkaCom) Close() (err error) {
	if this.writer != nil {
		err = errors.Join(err, this.writer.Close())
	}
	if this.reader != nil {
		err = errors.Join(err, this.reader.Close())
	}
	return err
}

func (this *KafkaCom) SendPermissions(ctx context.Context, topic model.Topic, id string, permissions model.ResourcePermissions) (err error) {
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
		log.Println("produce:", id, topic.KafkaTopic, key, string(temp))
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
		Topic:       topic.KafkaTopic,
		MaxAttempts: 10,
		Logger:      logger,
		BatchSize:   1,
		Balancer:    &KeySeparationBalancer{SubBalancer: &kafka.Hash{}, Seperator: "/"},
	}
	return writer
}

func NewKafkaReader(config configuration.Config, topic model.Topic, handler ReadHandler) (reader *kafka.Reader, err error) {
	log.Println("new consumer for topic", topic.KafkaTopic)
	consumerGroup := topic.KafkaConsumerGroup
	if consumerGroup == "" {
		consumerGroup = config.DefaultKafkaConsumerGroup
	}
	reader = kafka.NewReader(kafka.ReaderConfig{
		CommitInterval:         0, //synchronous commits
		Brokers:                []string{config.KafkaUrl},
		GroupID:                consumerGroup,
		Topic:                  topic.KafkaTopic,
		MaxWait:                1 * time.Second,
		Logger:                 log.New(io.Discard, "", 0),
		ErrorLogger:            log.New(os.Stdout, "[KAFKA-ERR] ", log.LstdFlags),
		WatchPartitionChanges:  true,
		PartitionWatchInterval: time.Minute,
	})
	go func() {
		defer log.Println("close consumer for topic ", topic.KafkaTopic)
		for {
			m, err := reader.FetchMessage(context.Background())
			if err == io.EOF || err == context.Canceled {
				return
			}
			if err != nil {
				handler.HandleReaderError(topic, fmt.Errorf("unable to fetch message: %w", err))
				return
			}

			err = retry(func() error {
				return handleKafkaMessageAsCommand(handler, topic, m)
			}, func(n int64) time.Duration {
				return time.Duration(n) * time.Second
			}, 10*time.Minute)

			if err != nil {
				handler.HandleReaderError(topic, fmt.Errorf("unable to handle message (no commit): %w", err))
			} else {
				timeout, _ := context.WithTimeout(context.Background(), 10*time.Second)
				err = reader.CommitMessages(timeout, m)
				if err != nil {
					handler.HandleReaderError(topic, fmt.Errorf("unable to commit consumption: %w", err))
					return
				}
			}
		}
	}()
	return reader, nil
}

func retry(f func() error, waitProvider func(n int64) time.Duration, timeout time.Duration) (err error) {
	err = errors.New("initial")
	start := time.Now()
	for i := int64(1); err != nil && time.Since(start) < timeout; i++ {
		err = f()
		if err != nil {
			log.Println("ERROR: kafka listener error:", err)
			wait := waitProvider(i)
			if time.Since(start)+wait < timeout {
				log.Println("ERROR: retry after:", wait.String())
				time.Sleep(wait)
			} else {
				return err
			}
		}
	}
	return err
}

func handleKafkaMessageAsCommand(handler ReadHandler, topic model.Topic, m kafka.Message) error {
	cmd := Command{}
	err := json.Unmarshal(m.Value, &cmd)
	if err != nil {
		return err
	}
	switch cmd.Command {
	case "RIGHTS":
		return handler.HandleReceivedCommand(topic, model.Resource{
			Id:                  cmd.Id,
			TopicId:             topic.Id,
			ResourcePermissions: rightsToPermissions(cmd.Rights),
		}, m.Time)
	case "PUT", "POST":
		return handler.HandleResourceUpdate(topic, cmd.Id, cmd.Owner)
	case "DELETE":
		return handler.HandleResourceDelete(topic, cmd.Id)
	}
	return nil
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
