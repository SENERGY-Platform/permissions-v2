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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/SENERGY-Platform/permissions-v2/pkg"
	"github.com/SENERGY-Platform/permissions-v2/pkg/client"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/permissions-v2/pkg/tests/docker"
	"github.com/SENERGY-Platform/service-commons/pkg/kafka"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAdminLoadFromPermissionSearch(t *testing.T) {
	t.Skip("permission-search is deprecated")
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

	_, openSearchIp, err := docker.OpenSearch(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}

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

	err = kafka.InitTopic(config.KafkaUrl, "locations")
	if err != nil {
		t.Error(err)
		return
	}

	_, searchIp, err := docker.PermissionSearch(ctx, wg, config.KafkaUrl, openSearchIp)
	if err != nil {
		t.Error(err)
		return
	}
	permissionSearchUrl := "http://" + searchIp + ":8080"

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

	apiUrl := "http://localhost:" + config.Port
	c := client.New(apiUrl)

	t.Run("create locations", func(t *testing.T) {

		err = kafka.NewConsumer(ctx, kafka.Config{
			KafkaUrl: config.KafkaUrl,
			Wg:       wg,
			Debug:    true,
			OnError: func(err error) {
				t.Error(err)
			},
		}, "locations", func(delivery []byte) error {
			t.Log("location consumed:", string(delivery))
			return nil
		})

		producer, err := kafka.NewProducer(ctx, kafka.Config{
			KafkaUrl: config.KafkaUrl,
			Debug:    true,
			Wg:       wg,
			OnError: func(err error) {
				t.Error(err)
			},
		}, "locations")
		if err != nil {
			t.Error(err)
			return
		}

		msgTmpl := `{
			"command": "PUT",
			"id": "%s",
			"owner": "%s",
			"location": {"name":"%s", "description": "", "image": "", "device_ids": [], "device_group_ids": []}
		}`

		err = producer.Produce("key", []byte(fmt.Sprintf(msgTmpl, "location:1", "owner1", "l1")))
		if err != nil {
			t.Error(err)
			return
		}
		err = producer.Produce("key", []byte(fmt.Sprintf(msgTmpl, "location:2", "owner2", "l2")))
		if err != nil {
			t.Error(err)
			return
		}
		err = producer.Produce("key", []byte(fmt.Sprintf(msgTmpl, "location:3", "owner3", "l3")))
		if err != nil {
			t.Error(err)
			return
		}
		err = producer.Produce("key", []byte(fmt.Sprintf(msgTmpl, "location:4", "owner4", "l4")))
		if err != nil {
			t.Error(err)
			return
		}
		err = producer.Produce("key", []byte(fmt.Sprintf(msgTmpl, "location:5", "owner5", "l5")))
		if err != nil {
			t.Error(err)
			return
		}
	})

	time.Sleep(10 * time.Second)

	t.Run("create topic", func(t *testing.T) {
		_, err, _ = c.SetTopic(client.InternalAdminToken, client.Topic{
			Id:                  "locations",
			PublishToKafkaTopic: "locations",
		})
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("dry-run load from permission search", func(t *testing.T) {
		//no client use because it is not supported
		msg := model.AdminLoadPermSearchRequest{
			PermissionSearchUrl: permissionSearchUrl,
			Token:               client.InternalAdminToken,
			TopicId:             "locations",
			OverwriteExisting:   false,
			DryRun:              true,
		}
		buf, err := json.Marshal(msg)
		if err != nil {
			t.Error(err)
			return
		}
		req, err := http.NewRequest(http.MethodPost, apiUrl+"/admin/load/permission-search", bytes.NewBuffer(buf))
		if err != nil {
			t.Error(err)
			return
		}
		req.Header.Set("Authorization", client.InternalAdminToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Error(resp.StatusCode)
			return
		}
		var count int = 0
		err = json.NewDecoder(resp.Body).Decode(&count)
		if err != nil {
			t.Error(err)
			return
		}
		if count <= 0 {
			t.Error(count)
		}
	})

	t.Run("check no locations after dry run", func(t *testing.T) {
		ids, err, _ := c.AdminListResourceIds(client.InternalAdminToken, "locations", client.ListOptions{})
		if err != nil {
			t.Error(err)
			return
		}
		if len(ids) != 0 {
			t.Error(len(ids))
			return
		}
	})

	t.Run("load from permission search", func(t *testing.T) {
		//no client use because it is not supported
		msg := model.AdminLoadPermSearchRequest{
			PermissionSearchUrl: permissionSearchUrl,
			Token:               client.InternalAdminToken,
			TopicId:             "locations",
			OverwriteExisting:   false,
			DryRun:              false,
		}
		buf, err := json.Marshal(msg)
		if err != nil {
			t.Error(err)
			return
		}
		req, err := http.NewRequest(http.MethodPost, apiUrl+"/admin/load/permission-search", bytes.NewBuffer(buf))
		if err != nil {
			t.Error(err)
			return
		}
		req.Header.Set("Authorization", client.InternalAdminToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Error(resp.StatusCode)
			return
		}
		var count int = 0
		err = json.NewDecoder(resp.Body).Decode(&count)
		if err != nil {
			t.Error(err)
			return
		}
		if count <= 0 {
			t.Error(count)
		}
	})

	t.Run("check locations after load", func(t *testing.T) {
		ids, err, _ := c.AdminListResourceIds(client.InternalAdminToken, "locations", client.ListOptions{})
		if err != nil {
			t.Error(err)
			return
		}
		if len(ids) != 5 {
			t.Error(len(ids))
			return
		}

		list, err, _ := c.ListResourcesWithAdminPermission(client.InternalAdminToken, "locations", client.ListOptions{})
		if err != nil {
			t.Error(err)
			return
		}
		expected := []model.Resource{
			{
				Id:      "location:1",
				TopicId: "locations",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{"owner1": {Read: true, Write: true, Execute: true, Administrate: true}},
					GroupPermissions: map[string]model.PermissionsMap{},
					RolePermissions:  map[string]model.PermissionsMap{"admin": {Read: true, Write: true, Execute: true, Administrate: true}},
				},
			},
			{
				Id:      "location:2",
				TopicId: "locations",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{"owner2": {Read: true, Write: true, Execute: true, Administrate: true}},
					GroupPermissions: map[string]model.PermissionsMap{},
					RolePermissions:  map[string]model.PermissionsMap{"admin": {Read: true, Write: true, Execute: true, Administrate: true}},
				},
			},
			{
				Id:      "location:3",
				TopicId: "locations",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{"owner3": {Read: true, Write: true, Execute: true, Administrate: true}},
					GroupPermissions: map[string]model.PermissionsMap{},
					RolePermissions:  map[string]model.PermissionsMap{"admin": {Read: true, Write: true, Execute: true, Administrate: true}},
				},
			},
			{
				Id:      "location:4",
				TopicId: "locations",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{"owner4": {Read: true, Write: true, Execute: true, Administrate: true}},
					GroupPermissions: map[string]model.PermissionsMap{},
					RolePermissions:  map[string]model.PermissionsMap{"admin": {Read: true, Write: true, Execute: true, Administrate: true}},
				},
			},
			{
				Id:      "location:5",
				TopicId: "locations",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{"owner5": {Read: true, Write: true, Execute: true, Administrate: true}},
					GroupPermissions: map[string]model.PermissionsMap{},
					RolePermissions:  map[string]model.PermissionsMap{"admin": {Read: true, Write: true, Execute: true, Administrate: true}},
				},
			},
		}
		slices.SortFunc(expected, func(a, b model.Resource) int {
			return strings.Compare(a.Id, b.Id)
		})
		slices.SortFunc(list, func(a, b model.Resource) int {
			return strings.Compare(a.Id, b.Id)
		})
		if !reflect.DeepEqual(list, expected) {
			t.Error(list)
			return
		}
	})

	t.Run("load without overwrite", func(t *testing.T) {
		//no client use because it is not supported
		msg := model.AdminLoadPermSearchRequest{
			PermissionSearchUrl: permissionSearchUrl,
			Token:               client.InternalAdminToken,
			TopicId:             "locations",
			OverwriteExisting:   false,
			DryRun:              false,
		}
		buf, err := json.Marshal(msg)
		if err != nil {
			t.Error(err)
			return
		}
		req, err := http.NewRequest(http.MethodPost, apiUrl+"/admin/load/permission-search", bytes.NewBuffer(buf))
		if err != nil {
			t.Error(err)
			return
		}
		req.Header.Set("Authorization", client.InternalAdminToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Error(resp.StatusCode)
			return
		}
		var count int = 0
		err = json.NewDecoder(resp.Body).Decode(&count)
		if err != nil {
			t.Error(err)
			return
		}
		if count > 0 {
			t.Error(count)
		}
	})

	t.Run("load with overwrite", func(t *testing.T) {
		//no client use because it is not supported
		msg := model.AdminLoadPermSearchRequest{
			PermissionSearchUrl: permissionSearchUrl,
			Token:               client.InternalAdminToken,
			TopicId:             "locations",
			OverwriteExisting:   true,
			DryRun:              false,
		}
		buf, err := json.Marshal(msg)
		if err != nil {
			t.Error(err)
			return
		}
		req, err := http.NewRequest(http.MethodPost, apiUrl+"/admin/load/permission-search", bytes.NewBuffer(buf))
		if err != nil {
			t.Error(err)
			return
		}
		req.Header.Set("Authorization", client.InternalAdminToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Error(resp.StatusCode)
			return
		}
		var count int = 0
		err = json.NewDecoder(resp.Body).Decode(&count)
		if err != nil {
			t.Error(err)
			return
		}
		if count <= 0 {
			t.Error(count)
		}
	})

	t.Run("check locations after load", func(t *testing.T) {
		ids, err, _ := c.AdminListResourceIds(client.InternalAdminToken, "locations", client.ListOptions{})
		if err != nil {
			t.Error(err)
			return
		}
		if len(ids) != 5 {
			t.Error(len(ids))
			return
		}

		list, err, _ := c.ListResourcesWithAdminPermission(client.InternalAdminToken, "locations", client.ListOptions{})
		if err != nil {
			t.Error(err)
			return
		}
		expected := []model.Resource{
			{
				Id:      "location:1",
				TopicId: "locations",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{"owner1": {Read: true, Write: true, Execute: true, Administrate: true}},
					GroupPermissions: map[string]model.PermissionsMap{},
					RolePermissions:  map[string]model.PermissionsMap{"admin": {Read: true, Write: true, Execute: true, Administrate: true}},
				},
			},
			{
				Id:      "location:2",
				TopicId: "locations",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{"owner2": {Read: true, Write: true, Execute: true, Administrate: true}},
					GroupPermissions: map[string]model.PermissionsMap{},
					RolePermissions:  map[string]model.PermissionsMap{"admin": {Read: true, Write: true, Execute: true, Administrate: true}},
				},
			},
			{
				Id:      "location:3",
				TopicId: "locations",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{"owner3": {Read: true, Write: true, Execute: true, Administrate: true}},
					GroupPermissions: map[string]model.PermissionsMap{},
					RolePermissions:  map[string]model.PermissionsMap{"admin": {Read: true, Write: true, Execute: true, Administrate: true}},
				},
			},
			{
				Id:      "location:4",
				TopicId: "locations",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{"owner4": {Read: true, Write: true, Execute: true, Administrate: true}},
					GroupPermissions: map[string]model.PermissionsMap{},
					RolePermissions:  map[string]model.PermissionsMap{"admin": {Read: true, Write: true, Execute: true, Administrate: true}},
				},
			},
			{
				Id:      "location:5",
				TopicId: "locations",
				ResourcePermissions: model.ResourcePermissions{
					UserPermissions:  map[string]model.PermissionsMap{"owner5": {Read: true, Write: true, Execute: true, Administrate: true}},
					GroupPermissions: map[string]model.PermissionsMap{},
					RolePermissions:  map[string]model.PermissionsMap{"admin": {Read: true, Write: true, Execute: true, Administrate: true}},
				},
			},
		}
		slices.SortFunc(expected, func(a, b model.Resource) int {
			return strings.Compare(a.Id, b.Id)
		})
		slices.SortFunc(list, func(a, b model.Resource) int {
			return strings.Compare(a.Id, b.Id)
		})
		if !reflect.DeepEqual(list, expected) {
			t.Error(list)
			return
		}
	})
}
