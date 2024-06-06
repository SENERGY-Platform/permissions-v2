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

package mongo

import (
	"context"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/permissions-v2/pkg/tests/docker"
	"sync"
	"testing"
	"time"
)

func TestEvents(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.Load("../../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	port, _, err := docker.MongoDB(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.MongoUrl = "mongodb://localhost:" + port

	db, err := New(config)
	if err != nil {
		t.Error(err)
		return
	}

	err = db.SetTopic(nil, model.Topic{
		Id:                             "devices",
		KafkaTopic:                     "devices",
		EnsureTopicInit:                false,
		EnsureTopicInitPartitionNumber: 0,
		KafkaConsumerGroup:             "",
		InitialGroupRights:             nil,
	})
	if err != nil {
		t.Error(err)
		return
	}

	err = db.SetTopic(nil, model.Topic{
		Id:                             "devices",
		KafkaTopic:                     "devices",
		EnsureTopicInit:                true,
		EnsureTopicInitPartitionNumber: 2,
		KafkaConsumerGroup:             "consumergroup1",
		InitialGroupRights:             []model.GroupRight{{GroupName: "user", Permissions: model.Permissions{Read: true, Write: true}}},
	})
	if err != nil {
		t.Error(err)
		return
	}

	err = db.DeleteTopic(nil, "devices")
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(5 * time.Second)
}
