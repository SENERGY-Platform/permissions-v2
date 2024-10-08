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

package tests

import (
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/permissions-v2/pkg"
	"github.com/SENERGY-Platform/permissions-v2/pkg/client"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	kafka2 "github.com/SENERGY-Platform/permissions-v2/pkg/controller/kafka"
	"github.com/SENERGY-Platform/permissions-v2/pkg/database"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/permissions-v2/pkg/tests/docker"
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
	"github.com/SENERGY-Platform/service-commons/pkg/kafka"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"
)

const GroupTestToken = `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjM1MzMzNjMsImlhdCI6MCwianRpIjoiMCIsImlzcyI6InRlc3QiLCJhdWQiOiJhY2NvdW50Iiwic3ViIjoiZ3JvdXB1c2VyIiwidHlwIjoiQmVhcmVyIiwiYXpwIjoiZnJvbnRlbmQiLCJub25jZSI6IjAiLCJzZXNzaW9uX3N0YXRlIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6W119LCJyZXNvdXJjZV9hY2Nlc3MiOnsiYWNjb3VudCI6eyJyb2xlcyI6W119fSwic2NvcGUiOiJvcGVuaWQgcHJvZmlsZSBlbWFpbCIsInNpZCI6IjAiLCJlbWFpbF92ZXJpZmllZCI6ZmFsc2UsInJvbGVzIjpbXSwiZ3JvdXBzIjpbIi90ZXN0MSIsIi90ZXN0MS90ZXN0MiIsIi90ZXN0MS90ZXN0MyJdLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiIiLCJnaXZlbl9uYW1lIjoiIiwibG9jYWxlIjoiZGUiLCJmYW1pbHlfbmFtZSI6IiJ9.tcuCCDEa0kohl1HN22GQfkaK4zo4vQtz7P0fM_sSLNs`
const GroupTestTokenGroup = "/test1"

func TestGroupInToken(t *testing.T) {
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

	err = pkg.Start(ctx, config)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(time.Second)

	c := client.New("http://localhost:" + config.Port)

	_, err, _ = c.SetTopic(client.InternalAdminToken, model.Topic{
		Id: "a",
	})
	if err != nil {
		t.Error(err)
		return
	}

	_, err, _ = c.SetPermission(client.InternalAdminToken, "a", "a1", model.ResourcePermissions{
		UserPermissions:  map[string]model.PermissionsMap{TestTokenUser: {Read: true, Write: true, Execute: true, Administrate: true}},
		GroupPermissions: map[string]model.PermissionsMap{GroupTestTokenGroup: {Read: true, Write: true, Execute: true, Administrate: true}},
		RolePermissions:  map[string]model.PermissionsMap{},
	})
	if err != nil {
		t.Error(err)
		return
	}

	_, err, _ = c.SetPermission(GroupTestToken, "a", "a1", model.ResourcePermissions{
		UserPermissions:  map[string]model.PermissionsMap{TestTokenUser: {Read: false, Write: false, Execute: false, Administrate: true}},
		GroupPermissions: map[string]model.PermissionsMap{GroupTestTokenGroup: {Read: true, Write: true, Execute: true, Administrate: true}},
		RolePermissions:  map[string]model.PermissionsMap{},
	})
	if err != nil {
		t.Error(err)
		return
	}

	access, err, _ := c.CheckPermission(GroupTestToken, "a", "a1", model.Administrate)
	if err != nil {
		t.Error(err)
		return
	}
	if !access {
		t.Error("access should be true")
		return
	}

	accessMap, err, _ := c.CheckMultiplePermissions(GroupTestToken, "a", []string{"a1"}, model.Administrate)
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(accessMap, map[string]bool{"a1": true}) {
		t.Error(accessMap)
		return
	}

	computed, err, _ := c.ListComputedPermissions(GroupTestToken, "a", []string{"a1"})
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(computed, []model.ComputedPermissions{
		{
			Id:             "a1",
			PermissionsMap: model.PermissionsMap{Read: true, Write: true, Execute: true, Administrate: true},
		},
	}) {
		t.Error(accessMap)
	}

	ids, err, _ := c.ListAccessibleResourceIds(GroupTestToken, "a", model.ListOptions{}, model.Administrate)
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(ids, []string{"a1"}) {
		t.Error(ids)
		return
	}

	list, err, _ := c.ListResourcesWithAdminPermission(GroupTestToken, "a", model.ListOptions{})
	if err != nil {
		t.Error(err)
		return
	}
	if len(list) != 1 || list[0].Id != "a1" {
		t.Error(list)
		return
	}

	_, err, _ = c.GetResource(GroupTestToken, "a", "a1")
	if err != nil {
		t.Error(err)
		return
	}

	err, _ = c.RemoveResource(GroupTestToken, "a", "a1")
	if err == nil {
		t.Error("only admins may delete")
		return
	}
}

func TestOptionalPublish(t *testing.T) {
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

	err = pkg.Start(ctx, config)
	if err != nil {
		t.Error(err)
		return
	}

	consumed := []kafka2.Command{}
	err = kafka.NewMultiConsumer(ctx, kafka.Config{
		KafkaUrl: config.KafkaUrl,
		Wg:       wg,
		OnError: func(err error) {
			t.Error(err)
		},
	}, []string{"a", "b"}, func(delivery kafka.Message) error {
		if delivery.Topic != "a" {
			t.Error(delivery)
			return nil
		}
		cmd := kafka2.Command{}
		err = json.Unmarshal(delivery.Value, &cmd)
		if err != nil {
			t.Error(err)
			return nil
		}
		cmd.Owner = ""
		consumed = append(consumed, cmd)
		return nil
	})
	if err != nil {
		t.Error(err)
		return
	}

	c := client.New("http://localhost:" + config.Port)

	t.Run("create topics", func(t *testing.T) {
		_, err, _ = c.SetTopic(client.InternalAdminToken, model.Topic{
			Id:                   "a",
			PublishToKafkaTopic:  "a",
			EnsureKafkaTopicInit: true,
		})
		if err != nil {
			t.Error(err)
			return
		}

		_, err, _ = c.SetTopic(client.InternalAdminToken, model.Topic{
			Id: "b",
		})
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("send permissions", func(t *testing.T) {
		t.Run("init with kafka", func(t *testing.T) {
			_, err, _ = c.SetPermission(client.InternalAdminToken, "a", "a1", model.ResourcePermissions{UserPermissions: map[string]model.PermissionsMap{"user": {Read: true, Administrate: true}}})
			if err != nil {
				t.Error(err)
				return
			}
		})
		t.Run("init without kafka", func(t *testing.T) {
			_, err, _ = c.SetPermission(client.InternalAdminToken, "b", "b1", model.ResourcePermissions{UserPermissions: map[string]model.PermissionsMap{"user": {Read: true, Administrate: true}}})
			if err != nil {
				t.Error(err)
				return
			}
		})
		t.Run("update with kafka", func(t *testing.T) {
			_, err, _ = c.SetPermission(client.InternalAdminToken, "a", "a1", model.ResourcePermissions{UserPermissions: map[string]model.PermissionsMap{"user": {Read: true, Write: true, Administrate: true}}})
			if err != nil {
				t.Error(err)
				return
			}
		})

		t.Run("update without kafka", func(t *testing.T) {
			_, err, _ = c.SetPermission(client.InternalAdminToken, "b", "b1", model.ResourcePermissions{UserPermissions: map[string]model.PermissionsMap{"user": {Read: true, Write: true, Administrate: true}}})
			if err != nil {
				t.Error(err)
				return
			}
		})
	})

	t.Run("check synced marks", func(t *testing.T) {
		db, err := database.New(config)
		if err != nil {
			t.Error(err)
			return
		}
		list, err := db.ListUnsyncedResources(nil)
		if err != nil {
			t.Error(err)
			return
		}
		if len(list) > 0 {
			t.Error(list)
			return
		}
	})

	t.Run("check consumed", func(t *testing.T) {
		time.Sleep(time.Second)
		expected := []kafka2.Command{
			{
				Command: "RIGHTS",
				Id:      "a1",
				Rights: &kafka2.ResourcePermissions{
					UserRights: map[string]kafka2.Right{
						"user": {Read: true, Administrate: true},
					},
					GroupRights:          map[string]kafka2.Right{},
					KeycloakGroupsRights: map[string]kafka2.Right{},
				},
			},
			{
				Command: "RIGHTS",
				Id:      "a1",
				Rights: &kafka2.ResourcePermissions{
					UserRights: map[string]kafka2.Right{
						"user": {Read: true, Write: true, Administrate: true},
					},
					GroupRights:          map[string]kafka2.Right{},
					KeycloakGroupsRights: map[string]kafka2.Right{},
				},
			},
		}
		if !reflect.DeepEqual(consumed, expected) {
			t.Errorf("\n%v\n%v\n", expected, consumed)
		}
	})
}

func TestIntegration(t *testing.T) {
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

	err = pkg.Start(ctx, config)
	if err != nil {
		t.Error(err)
		return
	}

	forwardConfig := config
	freePort2, err := docker.GetFreePort()
	if err != nil {
		t.Error(err)
		return
	}
	forwardConfig.Port = strconv.Itoa(freePort2)
	forwardConfig.EditForward = "http://localhost:" + config.Port

	actualClient := client.New("http://localhost:" + config.Port)

	testClient, err := client.NewTestClient(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("with test client", RunTestsWithClient(config, testClient))

	t.Run("with actual client", RunTestsWithClient(config, actualClient))

}

func TestForward(t *testing.T) {
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

	err = pkg.Start(ctx, config)
	if err != nil {
		t.Error(err)
		return
	}

	forwardConfig := config
	freePort2, err := docker.GetFreePort()
	if err != nil {
		t.Error(err)
		return
	}
	forwardConfig.Port = strconv.Itoa(freePort2)
	forwardConfig.EditForward = "http://localhost:" + config.Port
	forwardConfig.KafkaUrl = ""
	err = pkg.Start(ctx, forwardConfig)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(time.Second)

	c := client.New("http://localhost:" + forwardConfig.Port)

	t.Run("with forward client", RunTestsWithClient(config, c))

}

// has role user
const TestTokenUser = "testOwner"
const TestToken = `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIwOGM0N2E4OC0yYzc5LTQyMGYtODEwNC02NWJkOWViYmU0MWUiLCJleHAiOjE1NDY1MDcyMzMsIm5iZiI6MCwiaWF0IjoxNTQ2NTA3MTczLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0OjgwMDEvYXV0aC9yZWFsbXMvbWFzdGVyIiwiYXVkIjoiZnJvbnRlbmQiLCJzdWIiOiJ0ZXN0T3duZXIiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJmcm9udGVuZCIsIm5vbmNlIjoiOTJjNDNjOTUtNzViMC00NmNmLTgwYWUtNDVkZDk3M2I0YjdmIiwiYXV0aF90aW1lIjoxNTQ2NTA3MDA5LCJzZXNzaW9uX3N0YXRlIjoiNWRmOTI4ZjQtMDhmMC00ZWI5LTliNjAtM2EwYWUyMmVmYzczIiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJ1c2VyIl19LCJyZXNvdXJjZV9hY2Nlc3MiOnsibWFzdGVyLXJlYWxtIjp7InJvbGVzIjpbInZpZXctcmVhbG0iLCJ2aWV3LWlkZW50aXR5LXByb3ZpZGVycyIsIm1hbmFnZS1pZGVudGl0eS1wcm92aWRlcnMiLCJpbXBlcnNvbmF0aW9uIiwiY3JlYXRlLWNsaWVudCIsIm1hbmFnZS11c2VycyIsInF1ZXJ5LXJlYWxtcyIsInZpZXctYXV0aG9yaXphdGlvbiIsInF1ZXJ5LWNsaWVudHMiLCJxdWVyeS11c2VycyIsIm1hbmFnZS1ldmVudHMiLCJtYW5hZ2UtcmVhbG0iLCJ2aWV3LWV2ZW50cyIsInZpZXctdXNlcnMiLCJ2aWV3LWNsaWVudHMiLCJtYW5hZ2UtYXV0aG9yaXphdGlvbiIsIm1hbmFnZS1jbGllbnRzIiwicXVlcnktZ3JvdXBzIl19LCJhY2NvdW50Ijp7InJvbGVzIjpbIm1hbmFnZS1hY2NvdW50IiwibWFuYWdlLWFjY291bnQtbGlua3MiLCJ2aWV3LXByb2ZpbGUiXX19LCJyb2xlcyI6WyJ1c2VyIl19.ykpuOmlpzj75ecSI6cHbCATIeY4qpyut2hMc1a67Ycg`

// has role user
const SecendOwnerTokenUser = "secondOwner"
const SecondOwnerToken = `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIwOGM0N2E4OC0yYzc5LTQyMGYtODEwNC02NWJkOWViYmU0MWUiLCJleHAiOjE1NDY1MDcyMzMsIm5iZiI6MCwiaWF0IjoxNTQ2NTA3MTczLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0OjgwMDEvYXV0aC9yZWFsbXMvbWFzdGVyIiwiYXVkIjoiZnJvbnRlbmQiLCJzdWIiOiJzZWNvbmRPd25lciIsInR5cCI6IkJlYXJlciIsImF6cCI6ImZyb250ZW5kIiwibm9uY2UiOiI5MmM0M2M5NS03NWIwLTQ2Y2YtODBhZS00NWRkOTczYjRiN2YiLCJhdXRoX3RpbWUiOjE1NDY1MDcwMDksInNlc3Npb25fc3RhdGUiOiI1ZGY5MjhmNC0wOGYwLTRlYjktOWI2MC0zYTBhZTIyZWZjNzMiLCJhY3IiOiIwIiwiYWxsb3dlZC1vcmlnaW5zIjpbIioiXSwicmVhbG1fYWNjZXNzIjp7InJvbGVzIjpbInVzZXIiXX0sInJlc291cmNlX2FjY2VzcyI6eyJtYXN0ZXItcmVhbG0iOnsicm9sZXMiOlsidmlldy1yZWFsbSIsInZpZXctaWRlbnRpdHktcHJvdmlkZXJzIiwibWFuYWdlLWlkZW50aXR5LXByb3ZpZGVycyIsImltcGVyc29uYXRpb24iLCJjcmVhdGUtY2xpZW50IiwibWFuYWdlLXVzZXJzIiwicXVlcnktcmVhbG1zIiwidmlldy1hdXRob3JpemF0aW9uIiwicXVlcnktY2xpZW50cyIsInF1ZXJ5LXVzZXJzIiwibWFuYWdlLWV2ZW50cyIsIm1hbmFnZS1yZWFsbSIsInZpZXctZXZlbnRzIiwidmlldy11c2VycyIsInZpZXctY2xpZW50cyIsIm1hbmFnZS1hdXRob3JpemF0aW9uIiwibWFuYWdlLWNsaWVudHMiLCJxdWVyeS1ncm91cHMiXX0sImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX0sInJvbGVzIjpbInVzZXIiXX0.cq8YeUuR0jSsXCEzp634fTzNbGkq_B8KbVrwBPgceJ4`

func RunTestsWithClient(config configuration.Config, c client.Client) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("cqrs topic crud", func(t *testing.T) {
			t.Run("create topic", func(t *testing.T) {
				result, err, _ := c.SetTopic(client.InternalAdminToken, model.Topic{
					Id:                   "devices",
					PublishToKafkaTopic:  "nopedevices",
					EnsureKafkaTopicInit: true,
				})
				if err != nil {
					t.Error(err)
					return
				}
				if result.Id != "devices" {
					t.Errorf("%#v\n", result)
					return
				}
			})
			t.Run("update topic", func(t *testing.T) {
				result, err, code := c.SetTopic(client.InternalAdminToken, model.Topic{
					Id:                   "devices",
					PublishToKafkaTopic:  "devices",
					EnsureKafkaTopicInit: true,
				})
				if err != nil {
					t.Error(err)
					return
				}
				if code != http.StatusOK {
					t.Error(code)
					return
				}
				if result.Id != "devices" {
					t.Errorf("%#v\n", result)
					return
				}
				if result.PublishToKafkaTopic != "devices" {
					t.Errorf("%#v\n", result)
					return
				}
			})
			t.Run("unchanged topic", func(t *testing.T) {
				result, err, code := c.SetTopic(client.InternalAdminToken, model.Topic{
					Id:                   "devices",
					PublishToKafkaTopic:  "devices",
					EnsureKafkaTopicInit: true,
				})
				if err != nil {
					t.Error(err)
					return
				}
				if code != http.StatusAccepted {
					t.Error(code)
					return
				}
				if result.Id != "devices" {
					t.Errorf("%#v\n", result)
					return
				}
				if result.PublishToKafkaTopic != "devices" {
					t.Errorf("%#v\n", result)
					return
				}
			})
			t.Run("get topic", func(t *testing.T) {
				result, err, _ := c.GetTopic(client.InternalAdminToken, "devices")
				if err != nil {
					t.Error(err)
					return
				}
				if result.Id != "devices" {
					t.Errorf("%#v\n", result)
					return
				}
			})
			t.Run("delete topic", func(t *testing.T) {
				temp, err, _ := c.SetTopic(client.InternalAdminToken, model.Topic{
					Id:                   "to_be_deleted",
					PublishToKafkaTopic:  "to_be_deleted",
					EnsureKafkaTopicInit: true,
				})
				if err != nil {
					t.Error(err)
					return
				}
				err, _ = c.RemoveTopic(client.InternalAdminToken, temp.Id)
				if err != nil {
					t.Error(err)
					return
				}
				list, err, _ := c.ListTopics(client.InternalAdminToken, model.ListOptions{})
				if err != nil {
					t.Error(err)
					return
				}
				if len(list) != 1 {
					t.Errorf("%#v\n", list)
					return
				}
				if !slices.ContainsFunc(list, func(topic model.Topic) bool {
					return topic.Id == "devices"
				}) {
					t.Errorf("%#v\n", list)
					return
				}
				if slices.ContainsFunc(list, func(topic model.Topic) bool {
					return topic.Id == "to_be_deleted"
				}) {
					t.Errorf("%#v\n", list)
					return
				}
			})
		})

		t.Run("none-cqrs topic crud", func(t *testing.T) {
			t.Run("create topic", func(t *testing.T) {
				result, err, _ := c.SetTopic(client.InternalAdminToken, model.Topic{
					Id: "foo",
				})
				if err != nil {
					t.Error(err)
					return
				}
				if result.Id != "foo" {
					t.Errorf("%#v\n", result)
					return
				}
			})
			t.Run("unchanged topic", func(t *testing.T) {
				result, err, code := c.SetTopic(client.InternalAdminToken, model.Topic{
					Id: "foo",
				})
				if err != nil {
					t.Error(err)
					return
				}
				if code != http.StatusAccepted {
					t.Error(code)
					return
				}
				if result.Id != "foo" {
					t.Errorf("%#v\n", result)
					return
				}
				if result.PublishToKafkaTopic != "" {
					t.Errorf("%#v\n", result)
					return
				}
			})
			t.Run("update topic", func(t *testing.T) {
				result, err, code := c.SetTopic(client.InternalAdminToken, model.Topic{
					Id:                  "foo",
					PublishToKafkaTopic: "foo",
				})
				if err != nil {
					t.Error(err)
					return
				}
				if code != http.StatusOK {
					t.Error(code)
					return
				}
				if result.Id != "foo" {
					t.Errorf("%#v\n", result)
					return
				}
				if result.PublishToKafkaTopic != "foo" {
					t.Errorf("%#v\n", result)
					return
				}
				result, err, code = c.SetTopic(client.InternalAdminToken, model.Topic{
					Id: "foo",
				})
				if err != nil {
					t.Error(err)
					return
				}
				if code != http.StatusOK {
					t.Error(code)
					return
				}
				if result.Id != "foo" {
					t.Errorf("%#v\n", result)
					return
				}
				if result.PublishToKafkaTopic != "" {
					t.Errorf("%#v\n", result)
					return
				}
			})
			t.Run("get topic", func(t *testing.T) {
				result, err, _ := c.GetTopic(client.InternalAdminToken, "foo")
				if err != nil {
					t.Error(err)
					return
				}
				if result.Id != "foo" {
					t.Errorf("%#v\n", result)
					return
				}
			})
			t.Run("delete topic", func(t *testing.T) {
				temp, err, _ := c.SetTopic(client.InternalAdminToken, model.Topic{
					Id:                   "to_be_deleted_2",
					PublishToKafkaTopic:  "to_be_deleted_2",
					EnsureKafkaTopicInit: true,
				})
				if err != nil {
					t.Error(err)
					return
				}
				err, _ = c.RemoveTopic(client.InternalAdminToken, temp.Id)
				if err != nil {
					t.Error(err)
					return
				}
				list, err, _ := c.ListTopics(client.InternalAdminToken, model.ListOptions{})
				if err != nil {
					t.Error(err)
					return
				}
				if len(list) != 2 {
					t.Errorf("%#v\n", list)
					return
				}
				if !slices.ContainsFunc(list, func(topic model.Topic) bool {
					return topic.Id == "devices"
				}) {
					t.Errorf("%#v\n", list)
					return
				}
				if !slices.ContainsFunc(list, func(topic model.Topic) bool {
					return topic.Id == "foo"
				}) {
					t.Errorf("%#v\n", list)
					return
				}
				if slices.ContainsFunc(list, func(topic model.Topic) bool {
					return topic.Id == "to_be_deleted"
				}) {
					t.Errorf("%#v\n", list)
					return
				}
			})
		})

		t.Run("try deleted topic", func(t *testing.T) {
			t.Run("try to_be_deleted", func(t *testing.T) {
				_, err, code := c.SetPermission(TestToken, "to_be_deleted", "nope", model.ResourcePermissions{UserPermissions: map[string]model.PermissionsMap{SecendOwnerTokenUser: {Read: true}, TestTokenUser: {true, true, true, true}}})
				if err == nil {
					t.Error("expect error")
					return
				}
				if code != http.StatusBadRequest && code != http.StatusNotFound {
					t.Error(code)
					return
				}

				access, err, _ := c.CheckPermission(SecondOwnerToken, "to_be_deleted", "2", model.Read)
				if err != nil {
					t.Error(err)
					return
				}
				if access {
					t.Error(access)
					return
				}

				_, err, code = c.GetResource(TestToken, "to_be_deleted", "nope")
				if err == nil {
					t.Error("expect error")
					return
				}
				if code != http.StatusNotFound {
					t.Error(code)
					return
				}
			})

			t.Run("try to_be_deleted_2", func(t *testing.T) {
				_, err, code := c.SetPermission(TestToken, "to_be_deleted_2", "nope", model.ResourcePermissions{UserPermissions: map[string]model.PermissionsMap{SecendOwnerTokenUser: {Read: true}, TestTokenUser: {true, true, true, true}}})
				if err == nil {
					t.Error("expect error")
					return
				}
				if code != http.StatusBadRequest && code != http.StatusNotFound {
					t.Error(code)
					return
				}

				access, err, _ := c.CheckPermission(SecondOwnerToken, "to_be_deleted_2", "2", model.Read)
				if err != nil {
					t.Error(err)
					return
				}
				if access {
					t.Error(access)
					return
				}

				_, err, code = c.GetResource(TestToken, "to_be_deleted_2", "nope")
				if err == nil {
					t.Error("expect error")
					return
				}
				if code != http.StatusNotFound {
					t.Error(code)
					return
				}
			})
		})

		t.Run("foo", RunTestsWithTopic(config, c, "foo"))
	}
}

func RunTestsWithTopic(config configuration.Config, c client.Client, topicId string) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("manage permissions", func(t *testing.T) {

			t.Run("initial permissions set", func(t *testing.T) {
				_, err, code := c.SetPermission(client.InternalAdminToken, topicId, "b", model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
					GroupPermissions: nil,
					RolePermissions:  nil,
				})
				if err != nil {
					t.Error(err)
					return
				}
				if code != http.StatusOK {
					t.Error(code)
					return
				}

				_, err, code = c.SetPermission(client.InternalAdminToken, topicId, "buseradmin", model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
					GroupPermissions: nil,
					RolePermissions:  map[string]model.PermissionsMap{"user": {true, true, true, true}},
				})
				if err != nil {
					t.Error(err)
					return
				}
				if code != http.StatusOK {
					t.Error(code)
					return
				}

				_, err, code = c.SetPermission(client.InternalAdminToken, topicId, "c", model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
					GroupPermissions: nil,
					RolePermissions:  nil,
				})
				if err != nil {
					t.Error(err)
					return
				}
				if code != http.StatusOK {
					t.Error(code)
					return
				}
			})

			t.Run("update permissions", func(t *testing.T) {
				_, err, code := c.SetPermission(client.InternalAdminToken, topicId, "a", model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}, SecendOwnerTokenUser: {true, true, true, true}},
					GroupPermissions: nil,
					RolePermissions:  nil,
				})
				if err != nil {
					t.Error(err)
					return
				}
				if code != http.StatusOK {
					t.Error(code)
					return
				}
				_, err, code = c.SetPermission(TestToken, topicId, "c", model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}, SecendOwnerTokenUser: {true, true, true, true}},
					GroupPermissions: nil,
					RolePermissions:  map[string]model.PermissionsMap{},
				})
				if err != nil {
					t.Error(err)
					return
				}
				if code != http.StatusOK {
					t.Error(code)
					return
				}
			})

			t.Run("admin list", func(t *testing.T) {
				list, err, _ := c.ListResourcesWithAdminPermission(TestToken, topicId, model.ListOptions{})
				if err != nil {
					t.Error(err)
					return
				}
				ids := []string{}
				for _, resource := range list {
					ids = append(ids, resource.Id)
				}
				if !reflect.DeepEqual(ids, []string{"a", "b", "buseradmin", "c"}) {
					t.Errorf("%#v\n", ids)
				}

				list, err, _ = c.ListResourcesWithAdminPermission(SecondOwnerToken, topicId, model.ListOptions{})
				if err != nil {
					t.Error(err)
					return
				}
				ids = []string{}
				for _, resource := range list {
					ids = append(ids, resource.Id)
				}
				if !reflect.DeepEqual(ids, []string{"a", "buseradmin", "c"}) {
					t.Errorf("%#v\n", ids)
				}
			})

			t.Run("permissions management auth", func(t *testing.T) {
				tests := map[string]map[string]bool{
					TestToken: {
						"a":          true,
						"b":          true,
						"buseradmin": true,
						"c":          true,
					},
					SecondOwnerToken: {
						"a":          true,
						"b":          false,
						"buseradmin": true,
						"c":          true,
					},
				}
				for user, sub := range tests {
					for id, access := range sub {
						token, _ := jwt.Parse(user)
						_, err, code := c.GetResource(user, topicId, id)
						if access {
							if err != nil {
								t.Error(token.GetUserId(), id, access, err)
								//return
							}
							if code != http.StatusOK {
								t.Error(token.GetUserId(), id, access, code)
								//return
							}
						} else {
							if err == nil {
								t.Error(token.GetUserId(), id, access, "expect error")
								//return
							}
							if code == http.StatusOK {
								t.Error(token.GetUserId(), id, access, code)
								//return
							}
						}
					}
				}

				for id, access := range tests[SecondOwnerToken] {
					user := SecondOwnerToken
					token, _ := jwt.Parse(user)
					_, err, code := c.SetPermission(user, topicId, id, model.ResourcePermissions{
						UserPermissions:  map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}, SecendOwnerTokenUser: {true, true, true, true}},
						GroupPermissions: nil,
						RolePermissions:  map[string]model.PermissionsMap{"user": {true, true, true, true}, "g2": {true, true, true, true}},
					})
					if access {
						if err != nil {
							t.Error(token.GetUserId(), id, access, err)
							return
						}
						if code != http.StatusOK {
							t.Error(token.GetUserId(), id, access, code)
							return
						}
					} else {
						if err == nil {
							t.Error(token.GetUserId(), id, access, "expect error")
							return
						}
						if code == http.StatusOK {
							t.Error(token.GetUserId(), id, access, code)
							return
						}
					}
				}

			})
		})

		t.Run("prevent admin less resource", func(t *testing.T) {
			_, err, _ := c.SetPermission(client.InternalAdminToken, topicId, "adminless", model.ResourcePermissions{
				UserPermissions:  map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}, SecendOwnerTokenUser: {true, true, true, false}},
				GroupPermissions: nil,
				RolePermissions:  map[string]model.PermissionsMap{"g2": {true, true, true, true}},
			})
			if err != nil {
				t.Error(err)
				return
			}

			_, err, _ = c.SetPermission(TestToken, topicId, "adminless", model.ResourcePermissions{
				UserPermissions:  map[string]model.PermissionsMap{SecendOwnerTokenUser: {true, true, true, false}},
				GroupPermissions: nil,
				RolePermissions:  map[string]model.PermissionsMap{"g2": {true, true, true, true}},
			})
			if err == nil {
				t.Error("expect error")
				return
			}

			_, err, _ = c.SetPermission(TestToken, topicId, "adminless", model.ResourcePermissions{
				UserPermissions:  map[string]model.PermissionsMap{SecendOwnerTokenUser: {true, true, true, true}},
				GroupPermissions: nil,
				RolePermissions:  map[string]model.PermissionsMap{"g2": {true, true, true, true}},
			})
			if err != nil {
				t.Error(err)
				return
			}
		})

		t.Run("check permissions", func(t *testing.T) {
			t.Run("init permissions", func(t *testing.T) {
				_, err, _ := c.SetPermission(client.InternalAdminToken, topicId, "1", model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
				})
				if err != nil {
					t.Error(err)
					return
				}
				_, err, _ = c.SetPermission(client.InternalAdminToken, topicId, "2", model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}},
					GroupPermissions: nil,
					RolePermissions:  map[string]model.PermissionsMap{"user": {true, true, true, true}},
				})
				if err != nil {
					t.Error(err)
					return
				}
				_, err, _ = c.SetPermission(client.InternalAdminToken, topicId, "3", model.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{TestTokenUser: {true, true, true, true}, SecendOwnerTokenUser: {true, true, true, true}},
				})
				if err != nil {
					t.Error(err)
					return
				}
			})

			t.Run("check", func(t *testing.T) {
				access, err, _ := c.CheckPermission(TestToken, topicId, "1", model.Read)
				if err != nil {
					t.Error(err)
					return
				}
				if !access {
					t.Error(access)
					return
				}
				access, err, _ = c.CheckPermission(TestToken, topicId, "2", model.Read)
				if err != nil {
					t.Error(err)
					return
				}
				if !access {
					t.Error(access)
					return
				}
				access, err, _ = c.CheckPermission(TestToken, topicId, "3", model.Read)
				if err != nil {
					t.Error(err)
					return
				}
				if !access {
					t.Error(access)
					return
				}
				access, err, _ = c.CheckPermission(TestToken, topicId, "2", model.Read)
				if err != nil {
					t.Error(err)
					return
				}
				if !access {
					t.Error(access)
					return
				}
				access, err, _ = c.CheckPermission(TestToken, topicId, "4", model.Read)
				if err != nil {
					t.Error(err)
					return
				}
				if access {
					t.Error(access)
					return
				}

				access, err, _ = c.CheckPermission(SecondOwnerToken, topicId, "1", model.Read)
				if err != nil {
					t.Error(err)
					return
				}
				if access {
					t.Error(access)
					return
				}
				access, err, _ = c.CheckPermission(SecondOwnerToken, topicId, "2", model.Read)
				if err != nil {
					t.Error(err)
					return
				}
				if !access {
					t.Error(access)
					return
				}
				access, err, _ = c.CheckPermission(SecondOwnerToken, topicId, "3", model.Read)
				if err != nil {
					t.Error(err)
					return
				}
				if !access {
					t.Error(access)
					return
				}
				access, err, _ = c.CheckPermission(SecondOwnerToken, topicId, "2", model.Read)
				if err != nil {
					t.Error(err)
					return
				}
				if !access {
					t.Error(access)
					return
				}
				access, err, _ = c.CheckPermission(SecondOwnerToken, topicId, "4", model.Read)
				if err != nil {
					t.Error(err)
					return
				}
				if access {
					t.Error(access)
					return
				}
			})

			t.Run("check multiple", func(t *testing.T) {
				access, err, _ := c.CheckMultiplePermissions(TestToken, topicId, []string{"1", "2", "3", "4"}, model.Read)
				if err != nil {
					t.Error(err)
					return
				}
				if !reflect.DeepEqual(access, map[string]bool{"1": true, "2": true, "3": true}) {
					t.Errorf("%#v\n", access)
					return
				}
				access, err, _ = c.CheckMultiplePermissions(SecondOwnerToken, topicId, []string{"1", "2", "3", "4"}, model.Read)
				if err != nil {
					t.Error(err)
					return
				}
				if !reflect.DeepEqual(access, map[string]bool{"1": false, "2": true, "3": true}) {
					t.Errorf("%#v\n", access)
					return
				}
			})

			t.Run("list", func(t *testing.T) {
				ids, err, _ := c.ListAccessibleResourceIds(TestToken, topicId, model.ListOptions{}, model.Read)
				if err != nil {
					t.Error(err)
					return
				}
				if !reflect.DeepEqual(ids, []string{"1", "2", "3", "a", "b", "buseradmin", "c"}) {
					t.Errorf("%#v\n", ids)
					return
				}
				ids, err, _ = c.ListAccessibleResourceIds(SecondOwnerToken, topicId, model.ListOptions{}, model.Read)
				if err != nil {
					t.Error(err)
					return
				}
				if !reflect.DeepEqual(ids, []string{"2", "3", "a", "adminless", "buseradmin", "c"}) {
					t.Errorf("%#v\n", ids)
					return
				}
			})

			t.Run("admin list", func(t *testing.T) {
				ids, err, _ := c.AdminListResourceIds(client.InternalAdminToken, topicId, model.ListOptions{})
				if err != nil {
					t.Error(err)
					return
				}
				if !reflect.DeepEqual(ids, []string{"1", "2", "3", "a", "adminless", "b", "buseradmin", "c"}) {
					t.Errorf("%#v\n", ids)
					return
				}
				_, err, _ = c.AdminListResourceIds(SecondOwnerToken, topicId, model.ListOptions{})
				if err == nil {
					t.Error("expect error")
					return
				}
			})

			t.Run("list computed permissions", func(t *testing.T) {
				t.Run("admin", func(t *testing.T) {
					ids := []string{"1", "2", "3", "a", "adminless", "b", "buseradmin", "c", "unknownFoobar"}
					actual, err, _ := c.ListComputedPermissions(client.InternalAdminToken, topicId, ids)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(actual, []model.ComputedPermissions{
						{Id: "1", PermissionsMap: model.PermissionsMap{Read: true, Write: true, Execute: true, Administrate: true}},
						{Id: "2", PermissionsMap: model.PermissionsMap{Read: true, Write: true, Execute: true, Administrate: true}},
						{Id: "3", PermissionsMap: model.PermissionsMap{Read: true, Write: true, Execute: true, Administrate: true}},
						{Id: "a", PermissionsMap: model.PermissionsMap{Read: true, Write: true, Execute: true, Administrate: true}},
						{Id: "adminless", PermissionsMap: model.PermissionsMap{Read: true, Write: true, Execute: true, Administrate: true}},
						{Id: "b", PermissionsMap: model.PermissionsMap{Read: true, Write: true, Execute: true, Administrate: true}},
						{Id: "buseradmin", PermissionsMap: model.PermissionsMap{Read: true, Write: true, Execute: true, Administrate: true}},
						{Id: "c", PermissionsMap: model.PermissionsMap{Read: true, Write: true, Execute: true, Administrate: true}},
						{Id: "unknownFoobar", PermissionsMap: model.PermissionsMap{Read: false, Write: false, Execute: false, Administrate: false}},
					}) {
						t.Errorf("%#v\n", actual)
						return
					}
				})

				t.Run("second owner", func(t *testing.T) {
					ids := []string{"1", "2", "3", "a", "adminless", "b", "buseradmin", "c", "unknownFoobar"}
					actual, err, _ := c.ListComputedPermissions(SecondOwnerToken, topicId, ids)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(actual, []model.ComputedPermissions{
						{Id: "1", PermissionsMap: model.PermissionsMap{Read: false, Write: false, Execute: false, Administrate: false}},
						{Id: "2", PermissionsMap: model.PermissionsMap{Read: true, Write: true, Execute: true, Administrate: true}},
						{Id: "3", PermissionsMap: model.PermissionsMap{Read: true, Write: true, Execute: true, Administrate: true}},
						{Id: "a", PermissionsMap: model.PermissionsMap{Read: true, Write: true, Execute: true, Administrate: true}},
						{Id: "adminless", PermissionsMap: model.PermissionsMap{Read: true, Write: true, Execute: true, Administrate: true}},
						{Id: "b", PermissionsMap: model.PermissionsMap{Read: false, Write: false, Execute: false, Administrate: false}},
						{Id: "buseradmin", PermissionsMap: model.PermissionsMap{Read: true, Write: true, Execute: true, Administrate: true}},
						{Id: "c", PermissionsMap: model.PermissionsMap{Read: true, Write: true, Execute: true, Administrate: true}},
						{Id: "unknownFoobar", PermissionsMap: model.PermissionsMap{Read: false, Write: false, Execute: false, Administrate: false}},
					}) {
						t.Errorf("%#v\n", actual)
						return
					}
				})

				t.Run("shorter id list", func(t *testing.T) {
					ids := []string{"1", "2"}
					actual, err, _ := c.ListComputedPermissions(SecondOwnerToken, topicId, ids)
					if err != nil {
						t.Error(err)
						return
					}
					if !reflect.DeepEqual(actual, []model.ComputedPermissions{
						{Id: "1", PermissionsMap: model.PermissionsMap{
							Read:         false,
							Write:        false,
							Execute:      false,
							Administrate: false,
						}},
						{Id: "2", PermissionsMap: model.PermissionsMap{
							Read:         true,
							Write:        true,
							Execute:      true,
							Administrate: true,
						}},
					}) {
						t.Errorf("%#v\n", actual)
						return
					}
				})

			})

			t.Run("check after resource delete", func(t *testing.T) {
				err, _ := c.RemoveResource(client.InternalAdminToken, topicId, "unknown")
				if err != nil {
					t.Error(err)
					return
				}
				err, _ = c.RemoveResource(client.InternalAdminToken, topicId, "1")
				if err != nil {
					t.Error(err)
					return
				}
				access, err, _ := c.CheckMultiplePermissions(TestToken, topicId, []string{"1", "2", "3", "4"}, model.Read)
				if err != nil {
					t.Error(err)
					return
				}
				if !reflect.DeepEqual(access, map[string]bool{"2": true, "3": true}) {
					t.Errorf("%#v\n", access)
					return
				}
			})

			t.Run("check after topic delete", func(t *testing.T) {
				err, _ := c.RemoveTopic(client.InternalAdminToken, topicId)
				if err != nil {
					t.Error(err)
					return
				}
				access, err, code := c.CheckMultiplePermissions(TestToken, topicId, []string{"1", "2", "3", "4"}, model.Read)
				if err == nil {
					t.Error("expect error")
					return
				}
				if code != http.StatusBadRequest && code != http.StatusNotFound {
					t.Error(code)
					return
				}
				if len(access) > 0 {
					t.Errorf("%#v\n", access)
					return
				}
				ids, err, code := c.ListAccessibleResourceIds(TestToken, topicId, model.ListOptions{}, model.Read)
				if err == nil {
					t.Error("expect error")
					return
				}
				if code != http.StatusBadRequest && code != http.StatusNotFound {
					t.Error(code)
					return
				}
				if len(ids) > 0 {
					t.Errorf("%#v\n", ids)
					return
				}

				_, err, code = c.SetPermission(TestToken, topicId, "nope", model.ResourcePermissions{UserPermissions: map[string]model.PermissionsMap{SecendOwnerTokenUser: {Read: true}, TestTokenUser: {true, true, true, true}}})
				if err == nil {
					t.Error("expect error")
					return
				}
				if code != http.StatusBadRequest && code != http.StatusNotFound {
					t.Error(code)
					return
				}

			})
		})
	}

}
