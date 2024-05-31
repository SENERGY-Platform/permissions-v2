/*
 * Copyright (c) 2022 InfAI (CC SES)
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
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/segmentio/kafka-go"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

func (this *Controller) newKafkaReader(topic model.Topic) (reader *kafka.Reader, err error) {
	if topic.EnsureTopicInit {
		err = initTopic(this.config.KafkaUrl, topic.EnsureTopicInitPartitionNumber, topic.KafkaTopic)
	}
	if err != nil {
		log.Println("ERROR: unable to create topic", err)
		return nil, err
	}
	consumerGroup := topic.KafkaConsumerGroup
	if consumerGroup == "" {
		consumerGroup = this.config.DefaultKafkaConsumerGroup
	}
	reader = kafka.NewReader(kafka.ReaderConfig{
		CommitInterval:         0, //synchronous commits
		Brokers:                []string{this.config.KafkaUrl},
		GroupID:                consumerGroup,
		Topic:                  topic.KafkaTopic,
		MaxWait:                1 * time.Second,
		Logger:                 log.New(io.Discard, "", 0),
		ErrorLogger:            log.New(os.Stdout, "[KAFKA-ERR] ", log.LstdFlags),
		WatchPartitionChanges:  true,
		PartitionWatchInterval: time.Minute,
	})
	go func() {
		defer log.Println("close consumer for topic ", topic)
		for {
			m, err := reader.FetchMessage(context.Background())
			if err == io.EOF || err == context.Canceled {
				return
			}
			if err != nil {
				this.handleReaderError(topic, fmt.Errorf("unable to fetch message: %w", err))
				return
			}

			err = retry(func() error {
				return this.handleReceivedCommand(topic, m)
			}, func(n int64) time.Duration {
				return time.Duration(n) * time.Second
			}, 10*time.Minute)

			if err != nil {
				this.handleReaderError(topic, fmt.Errorf("unable to handle message (no commit): %w", err))
			} else {
				timeout, _ := context.WithTimeout(context.Background(), 10*time.Second)
				err = reader.CommitMessages(timeout, m)
				if err != nil {
					this.handleReaderError(topic, fmt.Errorf("unable to commit consumption: %w", err))
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

func initTopic(bootstrapUrl string, partitionNumber int, topics ...string) (err error) {
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
