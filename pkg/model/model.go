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
	"reflect"
	"regexp"
	"slices"
	"strings"
)

const ClientVersion = "2" //increment on breaking changes to prevent old client access

type Topic struct {
	Id string `json:"id"`

	PublishToKafkaTopic string `json:"publish_to_kafka_topic"`

	EnsureKafkaTopicInit                bool `json:"ensure_kafka_topic_init"`
	EnsureKafkaTopicInitPartitionNumber int  `json:"ensure_kafka_topic_init_partition_number"`

	LastUpdateUnixTimestamp int64 `json:"last_update_unix_timestamp"` //should be ignored by the user; is set by db

	DefaultPermissions ResourcePermissions `json:"default_permissions"`
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

	if this.DefaultPermissions.UserPermissions == nil {
		this.DefaultPermissions.UserPermissions = map[string]PermissionsMap{}
	}
	if this.DefaultPermissions.GroupPermissions == nil {
		this.DefaultPermissions.GroupPermissions = map[string]PermissionsMap{}
	}
	if this.DefaultPermissions.RolePermissions == nil {
		this.DefaultPermissions.RolePermissions = map[string]PermissionsMap{}
	}

	if topic.DefaultPermissions.UserPermissions == nil {
		topic.DefaultPermissions.UserPermissions = map[string]PermissionsMap{}
	}
	if topic.DefaultPermissions.GroupPermissions == nil {
		topic.DefaultPermissions.GroupPermissions = map[string]PermissionsMap{}
	}
	if topic.DefaultPermissions.RolePermissions == nil {
		topic.DefaultPermissions.RolePermissions = map[string]PermissionsMap{}
	}

	if !reflect.DeepEqual(this.DefaultPermissions, topic.DefaultPermissions) {
		return false
	}
	return true
}

type AdminLoadPermSearchRequest struct {
	PermissionSearchUrl string `json:"permission_search_url"`
	Token               string `json:"token"`
	TopicId             string `json:"topic_id"`           //topic as used in permissions-v2
	OverwriteExisting   bool   `json:"overwrite_existing"` //false -> skip known elements; true -> force state of permission-search
	DryRun              bool   `json:"dry_run"`            //true -> log changes without executing them
}

type ImportExportOptions struct {
	IncludeTopicConfig bool     `json:"include_topic_config"`
	IncludePermissions bool     `json:"include_permissions"`
	FilterTopics       []string `json:"filter_topics,omitempty"`      //null->all; []->none
	FilterResourceId   []string `json:"filter_resource_id,omitempty"` //null->all; []->none
}

type ImportExport struct {
	Topics      []Topic    `json:"topics,omitempty"`
	Permissions []Resource `json:"permissions,omitempty"`
}

func (this *ImportExport) Sort() {
	slices.SortFunc(this.Topics, func(a, b Topic) int {
		return strings.Compare(a.Id, b.Id)
	})
	slices.SortFunc(this.Permissions, func(a, b Resource) int {
		comp := strings.Compare(a.TopicId, b.TopicId)
		if comp == 0 {
			comp = strings.Compare(a.Id, b.Id)
		}
		return comp
	})
}
