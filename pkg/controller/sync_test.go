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
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller/com"
	"github.com/SENERGY-Platform/permissions-v2/pkg/database"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/permissions-v2/pkg/tests/docker"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestTopicSync(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	config.Debug = true
	config.DevNotifierUrl = ""

	_, zkIp, err := docker.Zookeeper(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.KafkaUrl = zkIp + ":2181"

	//kafka
	config.KafkaUrl, err = docker.Kafka(ctx, wg, config.KafkaUrl)
	if err != nil {
		t.Error(err)
		return
	}

	dockerPort, _, err := docker.MongoDB(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.MongoUrl = "mongodb://localhost:" + dockerPort

	freePort, err := docker.GetFreePort()
	if err != nil {
		t.Error(err)
		return
	}
	config.Port = strconv.Itoa(freePort)

	db, err := database.New(config)
	if err != nil {
		t.Error(err)
		return
	}
	ctrl, err := NewWithDependencies(ctx, config, db, com.NewKafkaComProvider())
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("init topics", func(t *testing.T) {
		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id:     "topic1",
			NoCqrs: true,
		})
		if err != nil {
			t.Error(err)
			return
		}
		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id:     "topic2",
			NoCqrs: true,
		})
		if err != nil {
			t.Error(err)
			return
		}
		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id:     "topic3",
			NoCqrs: true,
		})
		if err != nil {
			t.Error(err)
			return
		}

		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id:              "topic4",
			KafkaTopic:      "topic4",
			EnsureTopicInit: true,
		})
		if err != nil {
			t.Error(err)
			return
		}

		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id:              "topic5",
			KafkaTopic:      "topic5",
			EnsureTopicInit: true,
		})
		if err != nil {
			t.Error(err)
			return
		}

		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id:              "topic6",
			KafkaTopic:      "topic6",
			EnsureTopicInit: true,
		})
		if err != nil {
			t.Error(err)
			return
		}

		err = ctrl.refreshTopics()
		if err != nil {
			t.Error(err)
			return
		}
		time.Sleep(5 * time.Second)
	})

	t.Run("check topics after init", func(t *testing.T) {
		t.Run("check struct field", func(t *testing.T) {
			ctrl.topicsMux.Lock()
			defer ctrl.topicsMux.Unlock()
			if len(ctrl.topics) != 6 {
				t.Errorf("%#v\n", ctrl.topics)
				return
			}
			if topic, ok := ctrl.topics["topic1"]; !ok || topic.Id != "topic1" {
				t.Errorf("%#v\n", ctrl.topics)
				return
			}
			if topic, ok := ctrl.topics["topic2"]; !ok || topic.Id != "topic2" {
				t.Errorf("%#v\n", ctrl.topics)
				return
			}
			if topic, ok := ctrl.topics["topic3"]; !ok || topic.Id != "topic3" {
				t.Errorf("%#v\n", ctrl.topics)
				return
			}
			if topic, ok := ctrl.topics["topic4"]; !ok || topic.Id != "topic4" {
				t.Errorf("%#v\n", ctrl.topics)
				return
			}
			if topic, ok := ctrl.topics["topic5"]; !ok || topic.Id != "topic5" {
				t.Errorf("%#v\n", ctrl.topics)
				return
			}
			if topic, ok := ctrl.topics["topic6"]; !ok || topic.Id != "topic6" {
				t.Errorf("%#v\n", ctrl.topics)
				return
			}
		})
		t.Run("set permission", func(t *testing.T) {
			_, err, _ = ctrl.SetPermission(TestToken, "topic1", "a1", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			}, model.SetPermissionOptions{Wait: true})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestToken, "topic2", "b1", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			}, model.SetPermissionOptions{Wait: true})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestToken, "topic3", "c1", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			}, model.SetPermissionOptions{Wait: true})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestToken, "topic4", "d1", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			}, model.SetPermissionOptions{Wait: true})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestToken, "topic5", "e1", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			}, model.SetPermissionOptions{Wait: true})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestToken, "topic6", "f1", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			}, model.SetPermissionOptions{Wait: true})
			if err != nil {
				t.Error(err)
				return
			}
		})
		t.Run("check permission", func(t *testing.T) {
			access, err, _ := ctrl.CheckPermission(TestToken, "topic1", "a1", model.Read)
			if err != nil {
				t.Error(err)
				return
			}
			if !access {
				t.Error("access should be true")
				return
			}
			access, err, _ = ctrl.CheckPermission(TestToken, "topic2", "b1", model.Read)
			if err != nil {
				t.Error(err)
				return
			}
			if !access {
				t.Error("access should be true")
				return
			}
			access, err, _ = ctrl.CheckPermission(TestToken, "topic3", "c1", model.Read)
			if err != nil {
				t.Error(err)
				return
			}
			if !access {
				t.Error("access should be true")
				return
			}
			access, err, _ = ctrl.CheckPermission(TestToken, "topic4", "d1", model.Read)
			if err != nil {
				t.Error(err)
				return
			}
			if !access {
				t.Error("access should be true")
				return
			}
			access, err, _ = ctrl.CheckPermission(TestToken, "topic5", "e1", model.Read)
			if err != nil {
				t.Error(err)
				return
			}
			if !access {
				t.Error("access should be true")
				return
			}
			access, err, _ = ctrl.CheckPermission(TestToken, "topic6", "f1", model.Read)
			if err != nil {
				t.Error(err)
				return
			}
			if !access {
				t.Error("access should be true")
				return
			}
		})
	})

	t.Run("update topics", func(t *testing.T) {
		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id:     "topic2",
			NoCqrs: true,
			InitialGroupPermissions: []model.GroupPermissions{
				{
					GroupName:      "g1",
					PermissionsMap: model.PermissionsMap{Read: true, Write: true, Execute: true, Administrate: true},
				},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		err = db.DeleteTopic(ctrl.getTimeoutContext(), "topic3")
		if err != nil {
			t.Error(err)
			return
		}
		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id:     "topic3_2",
			NoCqrs: true,
		})
		if err != nil {
			t.Error(err)
			return
		}

		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id:              "topic4",
			KafkaTopic:      "topic4",
			EnsureTopicInit: true,
		})
		if err != nil {
			t.Error(err)
			return
		}

		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id:              "topic5",
			KafkaTopic:      "topic5",
			EnsureTopicInit: true,
			InitialGroupPermissions: []model.GroupPermissions{
				{
					GroupName:      "g1",
					PermissionsMap: model.PermissionsMap{Read: true, Write: true, Execute: true, Administrate: true},
				},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}

		err = db.DeleteTopic(ctrl.getTimeoutContext(), "topic6")
		if err != nil {
			t.Error(err)
			return
		}

		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id:              "topic6_2",
			KafkaTopic:      "topic6_2",
			EnsureTopicInit: true,
		})
		if err != nil {
			t.Error(err)
			return
		}

		err = ctrl.refreshTopics()
		if err != nil {
			t.Error(err)
			return
		}
		time.Sleep(5 * time.Second)
	})

	t.Run("check topics after update", func(t *testing.T) {
		t.Run("check struct field", func(t *testing.T) {
			ctrl.topicsMux.Lock()
			defer ctrl.topicsMux.Unlock()
			if len(ctrl.topics) != 6 {
				t.Errorf("%#v\n", ctrl.topics)
				return
			}
			if topic, ok := ctrl.topics["topic1"]; !ok || topic.Id != "topic1" {
				t.Errorf("%#v\n", ctrl.topics)
				return
			}
			if topic, ok := ctrl.topics["topic2"]; !ok || topic.Id != "topic2" {
				t.Errorf("%#v\n", ctrl.topics)
				return
			}
			if _, ok := ctrl.topics["topic3"]; ok {
				t.Errorf("%#v\n", ctrl.topics)
				return
			}
			if topic, ok := ctrl.topics["topic3_2"]; !ok || topic.Id != "topic3_2" {
				t.Errorf("%#v\n", ctrl.topics)
				return
			}

			if topic, ok := ctrl.topics["topic4"]; !ok || topic.Id != "topic4" {
				t.Errorf("%#v\n", ctrl.topics)
				return
			}
			if topic, ok := ctrl.topics["topic5"]; !ok || topic.Id != "topic5" {
				t.Errorf("%#v\n", ctrl.topics)
				return
			}
			if _, ok := ctrl.topics["topic6"]; ok {
				t.Errorf("%#v\n", ctrl.topics)
				return
			}
			if topic, ok := ctrl.topics["topic6_2"]; !ok || topic.Id != "topic6_2" {
				t.Errorf("%#v\n", ctrl.topics)
				return
			}
		})
		t.Run("set permission", func(t *testing.T) {
			_, err, _ = ctrl.SetPermission(TestToken, "topic1", "a2", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			}, model.SetPermissionOptions{Wait: true})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestToken, "topic2", "b2", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			}, model.SetPermissionOptions{Wait: true})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestToken, "topic3_2", "c2", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			}, model.SetPermissionOptions{Wait: true})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestToken, "topic3", "c2", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			}, model.SetPermissionOptions{Wait: true})
			if err == nil {
				t.Error("expected error")
				return
			}
			_, err, _ = ctrl.SetPermission(TestToken, "topic4", "d2", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			}, model.SetPermissionOptions{Wait: true})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestToken, "topic5", "e2", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			}, model.SetPermissionOptions{Wait: true})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestToken, "topic6", "f2", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			}, model.SetPermissionOptions{Wait: true})
			if err == nil {
				t.Error("expected error")
				return
			}
			_, err, _ = ctrl.SetPermission(TestToken, "topic6_2", "f2", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			}, model.SetPermissionOptions{Wait: true})
			if err != nil {
				t.Error(err)
				return
			}
		})
		t.Run("check permission", func(t *testing.T) {
			access, err, _ := ctrl.CheckPermission(TestToken, "topic1", "a2", model.Read)
			if err != nil {
				t.Error(err)
				return
			}
			if !access {
				t.Error("access should be true")
				return
			}
			access, err, _ = ctrl.CheckPermission(TestToken, "topic2", "b2", model.Read)
			if err != nil {
				t.Error(err)
				return
			}
			if !access {
				t.Error("access should be true")
				return
			}
			access, err, _ = ctrl.CheckPermission(TestToken, "topic3", "c2", model.Read)
			if err != nil {
				t.Error(err)
				return
			}
			if access {
				t.Error("expected no access")
				return
			}
			access, err, _ = ctrl.CheckPermission(TestToken, "topic3_2", "c2", model.Read)
			if err != nil {
				t.Error(err)
				return
			}
			if !access {
				t.Error("access should be true")
				return
			}
			access, err, _ = ctrl.CheckPermission(TestToken, "topic4", "d2", model.Read)
			if err != nil {
				t.Error(err)
				return
			}
			if !access {
				t.Error("access should be true")
				return
			}
			access, err, _ = ctrl.CheckPermission(TestToken, "topic5", "e2", model.Read)
			if err != nil {
				t.Error(err)
				return
			}
			if !access {
				t.Error("access should be true")
				return
			}
			access, err, _ = ctrl.CheckPermission(TestToken, "topic6", "f2", model.Read)
			if err != nil {
				t.Error(err)
				return
			}
			if access {
				t.Error("expected no access")
				return
			}
			access, err, _ = ctrl.CheckPermission(TestToken, "topic6_2", "f2", model.Read)
			if err != nil {
				t.Error(err)
				return
			}
			if !access {
				t.Error("access should be true")
				return
			}
		})
	})

}

const TestTokenUser = "testOwner"
const TestToken = `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIwOGM0N2E4OC0yYzc5LTQyMGYtODEwNC02NWJkOWViYmU0MWUiLCJleHAiOjE1NDY1MDcyMzMsIm5iZiI6MCwiaWF0IjoxNTQ2NTA3MTczLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0OjgwMDEvYXV0aC9yZWFsbXMvbWFzdGVyIiwiYXVkIjoiZnJvbnRlbmQiLCJzdWIiOiJ0ZXN0T3duZXIiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJmcm9udGVuZCIsIm5vbmNlIjoiOTJjNDNjOTUtNzViMC00NmNmLTgwYWUtNDVkZDk3M2I0YjdmIiwiYXV0aF90aW1lIjoxNTQ2NTA3MDA5LCJzZXNzaW9uX3N0YXRlIjoiNWRmOTI4ZjQtMDhmMC00ZWI5LTliNjAtM2EwYWUyMmVmYzczIiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJ1c2VyIl19LCJyZXNvdXJjZV9hY2Nlc3MiOnsibWFzdGVyLXJlYWxtIjp7InJvbGVzIjpbInZpZXctcmVhbG0iLCJ2aWV3LWlkZW50aXR5LXByb3ZpZGVycyIsIm1hbmFnZS1pZGVudGl0eS1wcm92aWRlcnMiLCJpbXBlcnNvbmF0aW9uIiwiY3JlYXRlLWNsaWVudCIsIm1hbmFnZS11c2VycyIsInF1ZXJ5LXJlYWxtcyIsInZpZXctYXV0aG9yaXphdGlvbiIsInF1ZXJ5LWNsaWVudHMiLCJxdWVyeS11c2VycyIsIm1hbmFnZS1ldmVudHMiLCJtYW5hZ2UtcmVhbG0iLCJ2aWV3LWV2ZW50cyIsInZpZXctdXNlcnMiLCJ2aWV3LWNsaWVudHMiLCJtYW5hZ2UtYXV0aG9yaXphdGlvbiIsIm1hbmFnZS1jbGllbnRzIiwicXVlcnktZ3JvdXBzIl19LCJhY2NvdW50Ijp7InJvbGVzIjpbIm1hbmFnZS1hY2NvdW50IiwibWFuYWdlLWFjY291bnQtbGlua3MiLCJ2aWV3LXByb2ZpbGUiXX19LCJyb2xlcyI6WyJ1c2VyIl19.ykpuOmlpzj75ecSI6cHbCATIeY4qpyut2hMc1a67Ycg`
