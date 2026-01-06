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
	"errors"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller/kafka"
	"github.com/SENERGY-Platform/permissions-v2/pkg/database"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/permissions-v2/pkg/tests/docker"
)

type MockProducerProvider struct {
}

type MockProducer struct {
	Err      error
	Produced map[string]map[string][]model.ResourcePermissions
}

func (this *MockProducer) GetProducer(config configuration.Config, topic model.Topic) (result kafka.Producer, err error) {
	return this, nil
}

func (this *MockProducer) Close() (err error) {
	return nil
}

func (this *MockProducer) SendPermissions(ctx context.Context, topic model.Topic, id string, permissions model.ResourcePermissions) (err error) {
	if this.Err != nil {
		return this.Err
	}
	if _, ok := this.Produced[topic.PublishToKafkaTopic]; !ok {
		this.Produced[topic.PublishToKafkaTopic] = map[string][]model.ResourcePermissions{}
	}
	this.Produced[topic.PublishToKafkaTopic][id] = append(this.Produced[topic.PublishToKafkaTopic][id], permissions)
	return nil
}

func TestRetryPublishOfUnsyncedResources(t *testing.T) {
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
	config.SyncAgeLimit.SetDuration(time.Second)

	dockerPort, _, err := docker.MongoDB(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.MongoUrl = "mongodb://localhost:" + dockerPort

	db, err := database.New(config)
	if err != nil {
		t.Error(err)
		return
	}

	producer := &MockProducer{
		Err:      errors.New("test error"),
		Produced: map[string]map[string][]model.ResourcePermissions{},
	}

	ctrl, err := NewWithDependencies(ctx, config, db, producer)
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("create topic", func(t *testing.T) {
		_, err, _ = ctrl.SetTopic(TestAdminToken, model.Topic{
			Id:                  "a",
			PublishToKafkaTopic: "b",
		})
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("set permissions", func(t *testing.T) {
		_, err, _ = ctrl.SetPermission(TestAdminToken, "a", "a1", model.ResourcePermissions{UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {Administrate: true}}})
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("check permissions", func(t *testing.T) {
		access, err, _ := ctrl.CheckPermission(TestToken, "a", "a1", model.Administrate)
		if err != nil {
			t.Error(err)
			return
		}
		if !access {
			t.Error("expected access")
		}
	})

	time.Sleep(2 * time.Second)

	t.Run("check unsynced", func(t *testing.T) {
		list, err := ctrl.db.ListUnsyncedResources(nil)
		if err != nil {
			t.Error(err)
			return
		}
		if len(list) != 1 {
			t.Error(list)
			return
		}
		if list[0].Id != "a1" {
			t.Error(list)
			return
		}
	})

	t.Run("retry", func(t *testing.T) {
		producer.Err = nil
		err = ctrl.RetryPublishOfUnsyncedResources()
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("check permissions", func(t *testing.T) {
		access, err, _ := ctrl.CheckPermission(TestToken, "a", "a1", model.Administrate)
		if err != nil {
			t.Error(err)
			return
		}
		if !access {
			t.Error("expected access")
		}
	})

	t.Run("check unsynced", func(t *testing.T) {
		list, err := ctrl.db.ListUnsyncedResources(nil)
		if err != nil {
			t.Error(err)
			return
		}
		if len(list) != 0 {
			t.Error(list)
			return
		}
	})

	t.Run("check published", func(t *testing.T) {
		if !reflect.DeepEqual(producer.Produced, map[string]map[string][]model.ResourcePermissions{
			"b": {
				"a1": {
					{
						UserPermissions:  map[string]model.PermissionsMap{TestTokenUser: {Administrate: true}},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
			},
		}) {

		}
	})

}

func TestPermissionsSetAndCheck(t *testing.T) {
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

	config.KafkaUrl, err = docker.Kafka(ctx, wg)
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
	ctrl, err := NewWithDependencies(ctx, config, db, kafka.NewKafkaProducerProvider())
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("init topics", func(t *testing.T) {
		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id: "topic1",
		})
		if err != nil {
			t.Error(err)
			return
		}
		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id: "topic2",
		})
		if err != nil {
			t.Error(err)
			return
		}
		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id: "topic3",
		})
		if err != nil {
			t.Error(err)
			return
		}

		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id:                   "topic4",
			PublishToKafkaTopic:  "topic4",
			EnsureKafkaTopicInit: true,
		})
		if err != nil {
			t.Error(err)
			return
		}

		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id:                   "topic5",
			PublishToKafkaTopic:  "topic5",
			EnsureKafkaTopicInit: true,
		})
		if err != nil {
			t.Error(err)
			return
		}

		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id:                   "topic6",
			PublishToKafkaTopic:  "topic6",
			EnsureKafkaTopicInit: true,
		})
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("check topics after init", func(t *testing.T) {
		t.Run("set permission", func(t *testing.T) {
			_, err, _ = ctrl.SetPermission(TestAdminToken, "topic1", "a1", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestAdminToken, "topic2", "b1", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestAdminToken, "topic3", "c1", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestAdminToken, "topic4", "d1", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestAdminToken, "topic5", "e1", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestAdminToken, "topic6", "f1", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			})
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
			Id: "topic2",
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
			Id: "topic3_2",
		})
		if err != nil {
			t.Error(err)
			return
		}

		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id:                   "topic4",
			PublishToKafkaTopic:  "topic4",
			EnsureKafkaTopicInit: true,
		})
		if err != nil {
			t.Error(err)
			return
		}

		err = db.SetTopic(ctrl.getTimeoutContext(), model.Topic{
			Id:                   "topic5",
			PublishToKafkaTopic:  "topic5",
			EnsureKafkaTopicInit: true,
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
			Id:                   "topic6_2",
			PublishToKafkaTopic:  "topic6_2",
			EnsureKafkaTopicInit: true,
		})
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("check topics after update", func(t *testing.T) {
		t.Run("set permission", func(t *testing.T) {
			_, err, _ = ctrl.SetPermission(TestAdminToken, "topic1", "a2", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestAdminToken, "topic2", "b2", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestAdminToken, "topic3_2", "c2", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestAdminToken, "topic3", "c2", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			})
			if err == nil {
				t.Error("expected error")
				return
			}
			_, err, _ = ctrl.SetPermission(TestAdminToken, "topic4", "d2", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestAdminToken, "topic5", "e2", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			})
			if err != nil {
				t.Error(err)
				return
			}
			_, err, _ = ctrl.SetPermission(TestAdminToken, "topic6", "f2", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			})
			if err == nil {
				t.Error("expected error")
				return
			}
			_, err, _ = ctrl.SetPermission(TestAdminToken, "topic6_2", "f2", model.ResourcePermissions{
				UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
			})
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

const TestAdminToken = `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjEwMDAwMDAwMDAsImlhdCI6MTAwMDAwMDAwMCwiYXV0aF90aW1lIjoxMDAwMDAwMDAwLCJpc3MiOiJpbnRlcm5hbCIsImF1ZCI6W10sInN1YiI6ImRkNjllYTBkLWY1NTMtNDMzNi04MGYzLTdmNDU2N2Y4NWM3YiIsInR5cCI6IkJlYXJlciIsImF6cCI6ImZyb250ZW5kIiwicmVhbG1fYWNjZXNzIjp7InJvbGVzIjpbImFkbWluIiwiZGV2ZWxvcGVyIiwidXNlciJdfSwicmVzb3VyY2VfYWNjZXNzIjp7Im1hc3Rlci1yZWFsbSI6eyJyb2xlcyI6W119LCJCYWNrZW5kLXJlYWxtIjp7InJvbGVzIjpbXX0sImFjY291bnQiOnsicm9sZXMiOltdfX0sInJvbGVzIjpbImFkbWluIiwiZGV2ZWxvcGVyIiwidXNlciJdLCJuYW1lIjoiU2VwbCBBZG1pbiIsInByZWZlcnJlZF91c2VybmFtZSI6InNlcGwiLCJnaXZlbl9uYW1lIjoiU2VwbCIsImxvY2FsZSI6ImVuIiwiZmFtaWx5X25hbWUiOiJBZG1pbiIsImVtYWlsIjoic2VwbEBzZXBsLmRlIn0.HZyG6n-BfpnaPAmcDoSEh0SadxUx-w4sEt2RVlQ9e5I`
