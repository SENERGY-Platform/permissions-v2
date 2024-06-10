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
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller"
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller/com"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/permissions-v2/pkg/tests/docker"
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
	"github.com/segmentio/kafka-go"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"
)

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

	err = pkg.Start(ctx, wg, config)
	if err != nil {
		t.Error(err)
		return
	}

	actualClient := client.New("http://localhost:" + config.Port)

	testClient, err := client.NewTestClient(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("with test client", RunTestsWithClient(config, testClient))

	t.Run("with actual client", RunTestsWithClient(config, actualClient))

}

const Userjwt = "Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICIzaUtabW9aUHpsMmRtQnBJdS1vSkY4ZVVUZHh4OUFIckVOcG5CcHM5SjYwIn0.eyJqdGkiOiJiOGUyNGZkNy1jNjJlLTRhNWQtOTQ4ZC1mZGI2ZWVkM2JmYzYiLCJleHAiOjE1MzA1MzIwMzIsIm5iZiI6MCwiaWF0IjoxNTMwNTI4NDMyLCJpc3MiOiJodHRwczovL2F1dGguc2VwbC5pbmZhaS5vcmcvYXV0aC9yZWFsbXMvbWFzdGVyIiwiYXVkIjoiZnJvbnRlbmQiLCJzdWIiOiJkZDY5ZWEwZC1mNTUzLTQzMzYtODBmMy03ZjQ1NjdmODVjN2IiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJmcm9udGVuZCIsIm5vbmNlIjoiMjJlMGVjZjgtZjhhMS00NDQ1LWFmMjctNGQ1M2JmNWQxOGI5IiwiYXV0aF90aW1lIjoxNTMwNTI4NDIzLCJzZXNzaW9uX3N0YXRlIjoiMWQ3NWE5ODQtNzM1OS00MWJlLTgxYjktNzMyZDgyNzRjMjNlIiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJjcmVhdGUtcmVhbG0iLCJhZG1pbiIsImRldmVsb3BlciIsInVtYV9hdXRob3JpemF0aW9uIiwidXNlciJdfSwicmVzb3VyY2VfYWNjZXNzIjp7Im1hc3Rlci1yZWFsbSI6eyJyb2xlcyI6WyJ2aWV3LWlkZW50aXR5LXByb3ZpZGVycyIsInZpZXctcmVhbG0iLCJtYW5hZ2UtaWRlbnRpdHktcHJvdmlkZXJzIiwiaW1wZXJzb25hdGlvbiIsImNyZWF0ZS1jbGllbnQiLCJtYW5hZ2UtdXNlcnMiLCJxdWVyeS1yZWFsbXMiLCJ2aWV3LWF1dGhvcml6YXRpb24iLCJxdWVyeS1jbGllbnRzIiwicXVlcnktdXNlcnMiLCJtYW5hZ2UtZXZlbnRzIiwibWFuYWdlLXJlYWxtIiwidmlldy1ldmVudHMiLCJ2aWV3LXVzZXJzIiwidmlldy1jbGllbnRzIiwibWFuYWdlLWF1dGhvcml6YXRpb24iLCJtYW5hZ2UtY2xpZW50cyIsInF1ZXJ5LWdyb3VwcyJdfSwiYWNjb3VudCI6eyJyb2xlcyI6WyJtYW5hZ2UtYWNjb3VudCIsIm1hbmFnZS1hY2NvdW50LWxpbmtzIiwidmlldy1wcm9maWxlIl19fSwicm9sZXMiOlsidW1hX2F1dGhvcml6YXRpb24iLCJhZG1pbiIsImNyZWF0ZS1yZWFsbSIsImRldmVsb3BlciIsInVzZXIiLCJvZmZsaW5lX2FjY2VzcyJdLCJuYW1lIjoiZGYgZGZmZmYiLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJzZXBsIiwiZ2l2ZW5fbmFtZSI6ImRmIiwiZmFtaWx5X25hbWUiOiJkZmZmZiIsImVtYWlsIjoic2VwbEBzZXBsLmRlIn0.eOwKV7vwRrWr8GlfCPFSq5WwR_p-_rSJURXCV1K7ClBY5jqKQkCsRL2V4YhkP1uS6ECeSxF7NNOLmElVLeFyAkvgSNOUkiuIWQpMTakNKynyRfH0SrdnPSTwK2V1s1i4VjoYdyZWXKNjeT2tUUX9eCyI5qOf_Dzcai5FhGCSUeKpV0ScUj5lKrn56aamlW9IdmbFJ4VwpQg2Y843Vc0TqpjK9n_uKwuRcQd9jkKHkbwWQ-wyJEbFWXHjQ6LnM84H0CQ2fgBqPPfpQDKjGSUNaCS-jtBcbsBAWQSICwol95BuOAqVFMucx56Wm-OyQOuoQ1jaLt2t-Uxtr-C9wKJWHQ"
const Userid = "dd69ea0d-f553-4336-80f3-7f4567f85c7b"

// has role user
const TestTokenUser = "testOwner"
const TestToken = `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIwOGM0N2E4OC0yYzc5LTQyMGYtODEwNC02NWJkOWViYmU0MWUiLCJleHAiOjE1NDY1MDcyMzMsIm5iZiI6MCwiaWF0IjoxNTQ2NTA3MTczLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0OjgwMDEvYXV0aC9yZWFsbXMvbWFzdGVyIiwiYXVkIjoiZnJvbnRlbmQiLCJzdWIiOiJ0ZXN0T3duZXIiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJmcm9udGVuZCIsIm5vbmNlIjoiOTJjNDNjOTUtNzViMC00NmNmLTgwYWUtNDVkZDk3M2I0YjdmIiwiYXV0aF90aW1lIjoxNTQ2NTA3MDA5LCJzZXNzaW9uX3N0YXRlIjoiNWRmOTI4ZjQtMDhmMC00ZWI5LTliNjAtM2EwYWUyMmVmYzczIiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJ1c2VyIl19LCJyZXNvdXJjZV9hY2Nlc3MiOnsibWFzdGVyLXJlYWxtIjp7InJvbGVzIjpbInZpZXctcmVhbG0iLCJ2aWV3LWlkZW50aXR5LXByb3ZpZGVycyIsIm1hbmFnZS1pZGVudGl0eS1wcm92aWRlcnMiLCJpbXBlcnNvbmF0aW9uIiwiY3JlYXRlLWNsaWVudCIsIm1hbmFnZS11c2VycyIsInF1ZXJ5LXJlYWxtcyIsInZpZXctYXV0aG9yaXphdGlvbiIsInF1ZXJ5LWNsaWVudHMiLCJxdWVyeS11c2VycyIsIm1hbmFnZS1ldmVudHMiLCJtYW5hZ2UtcmVhbG0iLCJ2aWV3LWV2ZW50cyIsInZpZXctdXNlcnMiLCJ2aWV3LWNsaWVudHMiLCJtYW5hZ2UtYXV0aG9yaXphdGlvbiIsIm1hbmFnZS1jbGllbnRzIiwicXVlcnktZ3JvdXBzIl19LCJhY2NvdW50Ijp7InJvbGVzIjpbIm1hbmFnZS1hY2NvdW50IiwibWFuYWdlLWFjY291bnQtbGlua3MiLCJ2aWV3LXByb2ZpbGUiXX19LCJyb2xlcyI6WyJ1c2VyIl19.ykpuOmlpzj75ecSI6cHbCATIeY4qpyut2hMc1a67Ycg`

const SecendOwnerTokenUser = "secondOwner"
const SecondOwnerToken = `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIwOGM0N2E4OC0yYzc5LTQyMGYtODEwNC02NWJkOWViYmU0MWUiLCJleHAiOjE1NDY1MDcyMzMsIm5iZiI6MCwiaWF0IjoxNTQ2NTA3MTczLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0OjgwMDEvYXV0aC9yZWFsbXMvbWFzdGVyIiwiYXVkIjoiZnJvbnRlbmQiLCJzdWIiOiJzZWNvbmRPd25lciIsInR5cCI6IkJlYXJlciIsImF6cCI6ImZyb250ZW5kIiwibm9uY2UiOiI5MmM0M2M5NS03NWIwLTQ2Y2YtODBhZS00NWRkOTczYjRiN2YiLCJhdXRoX3RpbWUiOjE1NDY1MDcwMDksInNlc3Npb25fc3RhdGUiOiI1ZGY5MjhmNC0wOGYwLTRlYjktOWI2MC0zYTBhZTIyZWZjNzMiLCJhY3IiOiIwIiwiYWxsb3dlZC1vcmlnaW5zIjpbIioiXSwicmVhbG1fYWNjZXNzIjp7InJvbGVzIjpbInVzZXIiXX0sInJlc291cmNlX2FjY2VzcyI6eyJtYXN0ZXItcmVhbG0iOnsicm9sZXMiOlsidmlldy1yZWFsbSIsInZpZXctaWRlbnRpdHktcHJvdmlkZXJzIiwibWFuYWdlLWlkZW50aXR5LXByb3ZpZGVycyIsImltcGVyc29uYXRpb24iLCJjcmVhdGUtY2xpZW50IiwibWFuYWdlLXVzZXJzIiwicXVlcnktcmVhbG1zIiwidmlldy1hdXRob3JpemF0aW9uIiwicXVlcnktY2xpZW50cyIsInF1ZXJ5LXVzZXJzIiwibWFuYWdlLWV2ZW50cyIsIm1hbmFnZS1yZWFsbSIsInZpZXctZXZlbnRzIiwidmlldy11c2VycyIsInZpZXctY2xpZW50cyIsIm1hbmFnZS1hdXRob3JpemF0aW9uIiwibWFuYWdlLWNsaWVudHMiLCJxdWVyeS1ncm91cHMiXX0sImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX0sInJvbGVzIjpbInVzZXIiXX0.cq8YeUuR0jSsXCEzp634fTzNbGkq_B8KbVrwBPgceJ4`

func RunTestsWithClient(config configuration.Config, c client.Client) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("topic crud", func(t *testing.T) {
			t.Run("create topic", func(t *testing.T) {
				result, err, _ := c.SetTopic(client.InternalAdminToken, model.Topic{
					Id:                 "devices",
					KafkaTopic:         "nopedevices",
					EnsureTopicInit:    true,
					KafkaConsumerGroup: "",
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
					Id:                 "devices",
					KafkaTopic:         "devices",
					EnsureTopicInit:    true,
					KafkaConsumerGroup: "test_cg_1",
					InitialGroupRights: []model.GroupRight{
						{
							GroupName:   "g1",
							Permissions: model.Permissions{Read: true},
						},
					},
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
				if result.KafkaTopic != "devices" {
					t.Errorf("%#v\n", result)
					return
				}
				if result.KafkaConsumerGroup != "test_cg_1" {
					t.Errorf("%#v\n", result)
					return
				}
			})
			t.Run("unchanged topic", func(t *testing.T) {
				result, err, code := c.SetTopic(client.InternalAdminToken, model.Topic{
					Id:                 "devices",
					KafkaTopic:         "devices",
					EnsureTopicInit:    true,
					KafkaConsumerGroup: "test_cg_1",
					InitialGroupRights: []model.GroupRight{
						{
							GroupName:   "g1",
							Permissions: model.Permissions{Read: true},
						},
					},
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
				if result.KafkaTopic != "devices" {
					t.Errorf("%#v\n", result)
					return
				}
				if result.KafkaConsumerGroup != "test_cg_1" {
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
				if result.KafkaConsumerGroup != "test_cg_1" {
					t.Errorf("%#v\n", result)
					return
				}
			})
			t.Run("delete topic", func(t *testing.T) {
				temp, err, _ := c.SetTopic(client.InternalAdminToken, model.Topic{
					Id:                 "to_be_deleted",
					KafkaTopic:         "to_be_deleted",
					EnsureTopicInit:    true,
					KafkaConsumerGroup: "",
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
		t.Run("initial resource", func(t *testing.T) {
			if _, ok := c.(*controller.Controller); ok {
				t.Skip("skip for test client")
				return
			}
			writer := com.NewKafkaWriter(config, model.Topic{KafkaTopic: "devices"})
			buf, err := json.Marshal(com.Command{Command: "PUT", Id: "a", Owner: TestTokenUser})
			if err != nil {
				t.Error(err)
				return
			}
			err = writer.WriteMessages(context.Background(), kafka.Message{
				Key:   []byte("a"),
				Value: buf,
				Time:  time.Now(),
			})
			if err != nil {
				t.Error(err)
				return
			}

			time.Sleep(2 * time.Second)

			result, err, _ := c.GetResource(TestToken, "devices", "a")
			if err != nil {
				t.Error(err)
				return
			}

			if !reflect.DeepEqual(result, model.Resource{
				Id:      "a",
				TopicId: "devices",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions: map[string]model.Permissions{TestTokenUser: {
						Read:         true,
						Write:        true,
						Execute:      true,
						Administrate: true,
					}},
					GroupPermissions: map[string]model.Permissions{"g1": {Read: true}},
				},
			}) {
				t.Errorf("%#v\n", result)
			}

		})
		t.Run("topic sync interval", func(t *testing.T) {
			//TODO
			t.Skip("TODO")
		})
		t.Run("manage permissions", func(t *testing.T) {
			t.Run("try to_be_deleted topic", func(t *testing.T) {
				_, err, code := c.SetPermission(TestToken, "to_be_deleted", "nope", model.ResourcePermissions{UserPermissions: map[string]model.Permissions{SecendOwnerTokenUser: {Read: true}, TestTokenUser: {true, true, true, true}}}, model.SetPermissionOptions{Wait: true})
				if err == nil {
					t.Error("expect error")
					return
				}
				if code != http.StatusBadRequest {
					t.Error(code)
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

			t.Run("initial permissions set", func(t *testing.T) {
				_, err, code := c.SetPermission(TestToken, "devices", "b", model.ResourcePermissions{
					UserPermissions:  map[string]model.Permissions{TestTokenUser: {true, true, true, true}},
					GroupPermissions: nil,
				}, model.SetPermissionOptions{Wait: true})
				if err != nil {
					t.Error(err)
					return
				}
				if code != http.StatusOK {
					t.Error(code)
					return
				}

				_, err, code = c.SetPermission(TestToken, "devices", "buseradmin", model.ResourcePermissions{
					UserPermissions:  map[string]model.Permissions{TestTokenUser: {true, true, true, true}},
					GroupPermissions: map[string]model.Permissions{"user": {true, true, true, true}},
				}, model.SetPermissionOptions{Wait: true})
				if err != nil {
					t.Error(err)
					return
				}
				if code != http.StatusOK {
					t.Error(code)
					return
				}

				_, err, code = c.SetPermission(TestToken, "devices", "c", model.ResourcePermissions{
					UserPermissions:  map[string]model.Permissions{TestTokenUser: {true, true, true, true}},
					GroupPermissions: nil,
				}, model.SetPermissionOptions{Wait: true})
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
				_, err, code := c.SetPermission(TestToken, "devices", "a", model.ResourcePermissions{
					UserPermissions:  map[string]model.Permissions{TestTokenUser: {true, true, true, true}, SecendOwnerTokenUser: {true, true, true, true}},
					GroupPermissions: nil,
				}, model.SetPermissionOptions{Wait: true})
				if err != nil {
					t.Error(err)
					return
				}
				if code != http.StatusOK {
					t.Error(code)
					return
				}
				_, err, code = c.SetPermission(TestToken, "devices", "c", model.ResourcePermissions{
					UserPermissions:  map[string]model.Permissions{TestTokenUser: {true, true, true, true}, SecendOwnerTokenUser: {true, true, true, true}},
					GroupPermissions: map[string]model.Permissions{},
				}, model.SetPermissionOptions{Wait: true})
				if err != nil {
					t.Error(err)
					return
				}
				if code != http.StatusOK {
					t.Error(code)
					return
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
						_, err, code := c.GetResource(user, "devices", id)
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
					_, err, code := c.SetPermission(user, "devices", id, model.ResourcePermissions{
						UserPermissions:  map[string]model.Permissions{TestTokenUser: {true, true, true, true}, SecendOwnerTokenUser: {true, true, true, true}},
						GroupPermissions: map[string]model.Permissions{"user": {true, true, true, true}, "g2": {true, true, true, true}},
					}, model.SetPermissionOptions{Wait: true})
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
			_, err, _ := c.SetPermission(TestToken, "devices", "adminless", model.ResourcePermissions{
				UserPermissions:  map[string]model.Permissions{TestTokenUser: {true, true, true, true}, SecendOwnerTokenUser: {true, true, true, false}},
				GroupPermissions: map[string]model.Permissions{"g2": {true, true, true, true}},
			}, model.SetPermissionOptions{Wait: true})
			if err != nil {
				t.Error(err)
				return
			}

			_, err, _ = c.SetPermission(TestToken, "devices", "adminless", model.ResourcePermissions{
				UserPermissions:  map[string]model.Permissions{SecendOwnerTokenUser: {true, true, true, false}},
				GroupPermissions: map[string]model.Permissions{"g2": {true, true, true, true}},
			}, model.SetPermissionOptions{Wait: true})
			if err == nil {
				t.Error("expect error")
				return
			}

			_, err, _ = c.SetPermission(TestToken, "devices", "adminless", model.ResourcePermissions{
				UserPermissions:  map[string]model.Permissions{SecendOwnerTokenUser: {true, true, true, true}},
				GroupPermissions: map[string]model.Permissions{"g2": {true, true, true, true}},
			}, model.SetPermissionOptions{Wait: true})
			if err != nil {
				t.Error(err)
				return
			}
		})

		t.Run("check permissions", func(t *testing.T) {
			t.Run("init permissions", func(t *testing.T) {

			})
			t.Run("try to_be_deleted topic", func(t *testing.T) {
				//TODO
			})

			t.Run("check after topic delete", func(t *testing.T) {
				//TODO
			})
		})
	}
}
