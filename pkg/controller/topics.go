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
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/segmentio/kafka-go"
)

type TopicWrapper struct {
	model.Topic
	debug  bool
	writer *kafka.Writer
	reader *kafka.Reader
}

func (this *Controller) newTopicWrapper(topic model.Topic) (TopicWrapper, error) {
	reader, err := this.newKafkaReader(topic)
	if err != nil {
		return TopicWrapper{}, err
	}
	return TopicWrapper{writer: this.newKafkaWriter(topic), reader: reader, debug: this.config.Debug, Topic: topic}, nil
}

func (this *TopicWrapper) Close() (err error) {
	err = this.writer.Close()
	if err != nil {
		return err
	}
	err = this.reader.Close()
	if err != nil {
		return err
	}
	return nil
}
