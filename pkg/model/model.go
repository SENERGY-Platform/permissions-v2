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
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
)

type Topic struct {
	//at least one of Id and KafkaTopic must be set
	Id     string `json:"id"`
	NoCqrs bool   `json:"no_cqrs"`

	KafkaTopic string `json:"kafka_topic"`

	EnsureTopicInit                bool `json:"ensure_topic_init"`
	EnsureTopicInitPartitionNumber int  `json:"ensure_topic_init_partition_number"`

	KafkaConsumerGroup string `json:"kafka_consumer_group"` //defaults to configured kafka consumer group

	InitialGroupRights []GroupRight `json:"initial_group_rights"`

	//if true the user may not set permissions for not existing resources; if false the user may
	//if true the initial resource must be created by cqrs
	InitOnlyByCqrs bool `json:"init_only_by_cqrs"`

	LastUpdateUnixTimestamp int64 `json:"last_update_unix_timestamp"` //should be ignored by the user; is set by db
}

func (this Topic) Validate() error {
	if this.Id == "" {
		return errors.New("id is required")
	}
	if this.KafkaTopic == "" && !this.NoCqrs {
		return errors.New("kafka topic is required")
	}
	if strings.TrimSpace(this.KafkaTopic) != this.KafkaTopic {
		return errors.New("kafka_topic contains space pre/suffix")
	}
	if this.KafkaTopic != "" && !regexp.MustCompile("^[a-zA-Z0-9\\._\\-]+$").MatchString(this.KafkaTopic) {
		return errors.New("kafka topic contains invalid characters")
	}
	if this.NoCqrs && this.InitOnlyByCqrs {
		return errors.New("init_only_by_cqrs can not be true if no_cqrs is true")
	}
	usedGroupName := map[string]bool{}
	for _, g := range this.InitialGroupRights {
		if _, ok := usedGroupName[g.GroupName]; ok {
			return fmt.Errorf("duplicated initial group name '%v'", g.GroupName)
		}
		usedGroupName[g.GroupName] = true
	}
	return nil
}

func (this Topic) Equal(topic Topic) bool {
	if this.Id != topic.Id {
		return false
	}
	if this.NoCqrs != topic.NoCqrs {
		return false
	}
	if this.KafkaTopic != topic.KafkaTopic {
		return false
	}
	if this.EnsureTopicInit != topic.EnsureTopicInit {
		return false
	}
	if this.EnsureTopicInitPartitionNumber != topic.EnsureTopicInitPartitionNumber {
		return false
	}
	if this.KafkaConsumerGroup != topic.KafkaConsumerGroup {
		return false
	}
	if this.InitOnlyByCqrs != topic.InitOnlyByCqrs {
		return false
	}
	slices.SortFunc(this.InitialGroupRights, func(a, b GroupRight) int {
		return strings.Compare(a.GroupName, b.GroupName)
	})
	slices.SortFunc(topic.InitialGroupRights, func(a, b GroupRight) int {
		return strings.Compare(a.GroupName, b.GroupName)
	})
	if !reflect.DeepEqual(this.InitialGroupRights, topic.InitialGroupRights) {
		return false
	}
	return true
}

type GroupRight struct {
	GroupName string `json:"group_name"`
	Permissions
}

type Resource struct {
	Id      string `json:"id"`
	TopicId string `json:"topic_id"`
	ResourcePermissions
}

type ResourcePermissions struct {
	UserPermissions  map[string]Permissions `json:"user_permissions"`
	GroupPermissions map[string]Permissions `json:"group_permissions"`
}

func (this ResourcePermissions) Valid() bool {
	//needs at least one admin user
	for _, r := range this.UserPermissions {
		if r.Administrate {
			return true
		}
	}
	return false
}

type Permissions struct {
	Read         bool `json:"read"`
	Write        bool `json:"write"`
	Execute      bool `json:"execute"`
	Administrate bool `json:"administrate"`
}
