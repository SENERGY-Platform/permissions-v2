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

package model

import (
	"errors"
	"regexp"
	"strings"
)

type Topic struct {
	Id string `json:"id"`

	PublishToKafkaTopic string `json:"publish_to_kafka_topic"`

	EnsureKafkaTopicInit                bool `json:"ensure_kafka_topic_init"`
	EnsureKafkaTopicInitPartitionNumber int  `json:"ensure_kafka_topic_init_partition_number"`

	LastUpdateUnixTimestamp int64 `json:"last_update_unix_timestamp"` //should be ignored by the user; is set by db
}

func (this Topic) Validate() error {
	if this.Id == "" {
		return errors.New("id is required")
	}
	if strings.TrimSpace(this.PublishToKafkaTopic) != this.PublishToKafkaTopic {
		return errors.New("publish_to_kafka_topic contains space pre/suffix")
	}
	if this.PublishToKafkaTopic != "" && !regexp.MustCompile("^[a-zA-Z0-9\\._\\-]+$").MatchString(this.PublishToKafkaTopic) {
		return errors.New("kafka topic contains invalid characters")
	}
	return nil
}

func (this Topic) Equal(topic Topic) bool {
	if this.Id != topic.Id {
		return false
	}
	if this.PublishToKafkaTopic != topic.PublishToKafkaTopic {
		return false
	}
	if this.EnsureKafkaTopicInit != topic.EnsureKafkaTopicInit {
		return false
	}
	if this.EnsureKafkaTopicInitPartitionNumber != topic.EnsureKafkaTopicInitPartitionNumber {
		return false
	}
	return true
}
