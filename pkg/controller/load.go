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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

type PermissionSearchResponseElementRight struct {
	Read         bool `json:"read"`
	Write        bool `json:"write"`
	Execute      bool `json:"execute"`
	Administrate bool `json:"administrate"`
}

type PermissionSearchResponseElement struct {
	UserRights  map[string]PermissionSearchResponseElementRight `json:"user_rights"`
	GroupRights map[string]PermissionSearchResponseElementRight `json:"group_rights"`
	ResourceId  string                                          `json:"resource_id"`
	Creator     string                                          `json:"creator"`
}

func (this *Controller) AdminLoadFromPermissionSearch(req model.AdminLoadPermSearchRequest) (updateCount int, err error, code int) {
	updateCount = 0
	topic, exists, err := this.db.GetTopic(this.getTimeoutContext(), req.TopicId)
	if err != nil {
		return updateCount, err, http.StatusInternalServerError
	}
	if !exists {
		return updateCount, errors.New("unknown topic_id"), http.StatusBadRequest
	}
	if topic.PublishToKafkaTopic == "" {
		return updateCount, errors.New("no topic.publish_to_kafka_topic stored"), http.StatusBadRequest
	}
	if req.DryRun {
		log.Println("AdminLoadFromPermissionSearch Dry-Run Start")
		defer log.Println("AdminLoadFromPermissionSearch Dry-Run End")
	}
	limit := 100
	offset := 0
	for {
		r, err := http.NewRequest(http.MethodGet, req.PermissionSearchUrl+"/v3/export/"+url.PathEscape(topic.PublishToKafkaTopic)+"?limit="+strconv.Itoa(limit)+"&offset="+strconv.Itoa(offset), nil)
		if err != nil {
			return updateCount, err, http.StatusInternalServerError
		}
		r.Header.Set("Authorization", req.Token)
		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			return updateCount, err, http.StatusInternalServerError
		}
		defer resp.Body.Close()
		list := []PermissionSearchResponseElement{}
		err = json.NewDecoder(resp.Body).Decode(&list)
		if err != nil {
			return updateCount, err, http.StatusInternalServerError
		}
		for _, element := range list {
			updated, err := this.handlePermissionSearchExport(req, topic, element)
			if err != nil {
				return updateCount, err, http.StatusInternalServerError
			}
			if updated {
				updateCount++
			}
		}
		if len(list) < limit {
			return updateCount, nil, 200
		}
		offset = offset + limit
	}
}

func (this *Controller) handlePermissionSearchExport(req model.AdminLoadPermSearchRequest, topic model.Topic, element PermissionSearchResponseElement) (updated bool, err error) {
	ctx := this.getTimeoutContext()
	resource, err := this.db.GetResource(ctx, req.TopicId, element.ResourceId, model.GetOptions{})
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return false, err
	}
	if errors.Is(err, model.ErrNotFound) || req.OverwriteExisting {
		resource.Id = element.ResourceId
		resource.TopicId = req.TopicId
		if resource.UserPermissions == nil {
			resource.UserPermissions = map[string]model.PermissionsMap{}
		}
		if resource.GroupPermissions == nil {
			resource.GroupPermissions = map[string]model.PermissionsMap{}
		}
		if resource.RolePermissions == nil {
			resource.RolePermissions = map[string]model.PermissionsMap{}
		}
		for user, right := range element.UserRights {
			resource.UserPermissions[user] = model.PermissionsMap{
				Read:         right.Read,
				Write:        right.Write,
				Execute:      right.Execute,
				Administrate: right.Administrate,
			}
		}

		//group to role is a known semantic mismatch between permission-search and permissions-v2
		for role, right := range element.GroupRights {
			resource.RolePermissions[role] = model.PermissionsMap{
				Read:         right.Read,
				Write:        right.Write,
				Execute:      right.Execute,
				Administrate: right.Administrate,
			}
		}

		if req.DryRun {
			buf, _ := json.Marshal(resource)
			fmt.Println(string(buf))
			return true, nil //dry run is counted
		} else {
			err = this.setPermission(topic, resource)
			if err != nil {
				return true, err
			}
			return true, nil
		}

	}
	return false, nil
}
