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

type Topic struct {
	//at least one of Id and KafkaTopic must be set
	Id         string `json:"id"`          //unchangeable, defaults to KafkaTopic
	KafkaTopic string `json:"kafka_topic"` //changeable, defaults to Id

	KafkaConsumerGroup string `json:"kafka_consumer_group"` //defaults to configured kafka consumer group

	InitialGroupRights []GroupRight `json:"initial_group_rights"`
}

type GroupRight struct {
	GroupName string `json:"group_name"`
	Right
}

type Resource struct {
	Id      string `json:"id"`
	TopicId string `json:"topic_id"`
	ResourceRights
}

type ResourceRights struct {
	UserRights  map[string]Right `json:"user_rights"`
	GroupRights map[string]Right `json:"group_rights"`
}

type Right struct {
	Read         bool `json:"read"`
	Write        bool `json:"write"`
	Execute      bool `json:"execute"`
	Administrate bool `json:"administrate"`
}
