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

package mock

import (
	"context"
	"errors"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

func New() *Mock {
	return &Mock{}
}

type Mock struct {
	resources []ResourceWithTime
	topics    []model.Topic
	mux       sync.Mutex
}

type ResourceWithTime struct {
	model.Resource
	time time.Time
}

func (this *Mock) MarkResourceAsSynced(ctx context.Context, topicId string, id string) error {
	return nil
}

func (this *Mock) ListUnsyncedResources(ctx context.Context) ([]model.Resource, error) {
	return []model.Resource{}, nil
}

func (this *Mock) DeleteResource(ctx context.Context, topicId string, id string) error {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.resources = slices.DeleteFunc(this.resources, func(element ResourceWithTime) bool {
		return element.Id == id && element.TopicId == topicId
	})
	return nil
}

func (this *Mock) SetResource(ctx context.Context, r model.Resource, t time.Time, synced bool) (err error) {
	this.mux.Lock()
	defer this.mux.Unlock()
	for i, element := range this.resources {
		if element.Id == r.Id && element.TopicId == r.TopicId {
			this.resources[i] = ResourceWithTime{
				Resource: r,
				time:     t,
			}
			return nil
		}
	}
	this.resources = append(this.resources, ResourceWithTime{
		Resource: r,
		time:     t,
	})
	return nil
}

func (this *Mock) ListResourcesByPermissions(ctx context.Context, topicId string, userId string, roleIds []string, groupIds []string, options model.ListOptions, permissions ...model.Permission) (result []model.Resource, err error) {
	this.mux.Lock()
	defer this.mux.Unlock()
	for _, element := range this.resources {
		if element.TopicId == topicId && checkPerms(element, userId, roleIds, groupIds, permissions...) {
			result = append(result, element.Resource)
		}
	}
	slices.SortFunc(result, func(a, b model.Resource) int {
		return strings.Compare(a.Id, b.Id)
	})
	return limitOffset(result, options.Limit, options.Offset), nil
}

func limitOffset[T any](list []T, limit int64, offset int64) (result []T) {
	result = list
	if offset > 0 {
		if offset > int64(len(result)) {
			result = []T{}
		} else {
			result = result[offset:]
		}
	}
	if limit > 0 {
		if limit < int64(len(result)) {
			result = result[:limit]
		}
	}
	return result
}

func checkPerms(element ResourceWithTime, user string, roles []string, groups []string, permissions ...model.Permission) bool {
	for _, p := range permissions {
		if !checkPerm(element, user, roles, groups, p) {
			return false
		}
	}
	return true
}

func checkPerm(element ResourceWithTime, user string, roles []string, groups []string, permission model.Permission) bool {
	switch permission {
	case model.Read:
		if element.UserPermissions[user].Read {
			return true
		}
		for _, g := range groups {
			if element.GroupPermissions[g].Read {
				return true
			}
		}
		for _, g := range roles {
			if element.RolePermissions[g].Read {
				return true
			}
		}
	case model.Write:
		if element.UserPermissions[user].Write {
			return true
		}
		for _, g := range groups {
			if element.GroupPermissions[g].Write {
				return true
			}
		}
		for _, g := range roles {
			if element.RolePermissions[g].Write {
				return true
			}
		}
	case model.Execute:
		if element.UserPermissions[user].Execute {
			return true
		}
		for _, g := range groups {
			if element.GroupPermissions[g].Execute {
				return true
			}
		}
		for _, g := range roles {
			if element.RolePermissions[g].Execute {
				return true
			}
		}
	case model.Administrate:
		if element.UserPermissions[user].Administrate {
			return true
		}
		for _, g := range groups {
			if element.GroupPermissions[g].Administrate {
				return true
			}
		}
		for _, g := range roles {
			if element.RolePermissions[g].Administrate {
				return true
			}
		}
	}
	return false
}

func (this *Mock) AdminListResourceIds(ctx context.Context, topicId string, listOptions model.ListOptions) (result []string, err error) {
	this.mux.Lock()
	defer this.mux.Unlock()
	for _, element := range this.resources {
		if element.TopicId == topicId {
			result = append(result, element.Id)
		}
	}
	sort.Strings(result)
	return limitOffset(result, listOptions.Limit, listOptions.Offset), nil
}

func (this *Mock) AdminListResources(ctx context.Context, topicId string, listOptions model.ListOptions) (result []model.Resource, err error) {
	this.mux.Lock()
	defer this.mux.Unlock()
	for _, element := range this.resources {
		if element.TopicId == topicId && (listOptions.Ids == nil || slices.Contains(listOptions.Ids, element.Id)) {
			result = append(result, element.Resource)
		}
	}
	slices.SortFunc(result, func(a, b model.Resource) int {
		return strings.Compare(a.Id, b.Id)
	})
	return limitOffset(result, listOptions.Limit, listOptions.Offset), nil
}

func (this *Mock) ListResourceIdsByPermissions(ctx context.Context, topicId string, userId string, roleIds []string, groupIds []string, options model.ListOptions, permissions ...model.Permission) (result []string, err error) {
	list, err := this.ListResourcesByPermissions(ctx, topicId, userId, roleIds, groupIds, options, permissions...)
	if err != nil {
		return nil, err
	}
	for _, e := range list {
		result = append(result, e.Id)
	}
	return result, nil
}

func (this *Mock) GetResource(ctx context.Context, topicId string, id string, options model.GetOptions) (resource model.Resource, err error) {
	this.mux.Lock()
	defer this.mux.Unlock()
	for _, element := range this.resources {
		if element.TopicId == topicId && element.Id == id {
			if options.CheckPermission && !checkPerms(element, options.UserId, options.RoleIds, options.GroupIds, options.Permissions...) {
				return resource, model.PermissionCheckFailed
			}
			return element.Resource, nil
		}
	}
	return resource, model.ErrNotFound
}

func (this *Mock) CheckMultipleResourcePermissions(ctx context.Context, topicId string, ids []string, userId string, roleIds []string, groupIds []string, permissions ...model.Permission) (result map[string]bool, err error) {
	result = map[string]bool{}
	for _, id := range ids {
		_, err = this.GetResource(ctx, topicId, id, model.GetOptions{
			CheckPermission: true,
			UserId:          userId,
			RoleIds:         roleIds,
			GroupIds:        groupIds,
			Permissions:     permissions,
		})
		if errors.Is(err, model.PermissionCheckFailed) {
			result[id] = false
			err = nil
			continue
		}
		if errors.Is(err, model.ErrNotFound) {
			err = nil
			continue
		}
		if err != nil {
			return result, err
		}
		result[id] = true
	}
	return result, nil
}

func (this *Mock) CheckResourcePermissions(ctx context.Context, topicId string, id string, userId string, roleIds []string, groupIds []string, permissions ...model.Permission) (result bool, err error) {
	_, err = this.GetResource(ctx, topicId, id, model.GetOptions{
		CheckPermission: true,
		UserId:          userId,
		RoleIds:         roleIds,
		GroupIds:        groupIds,
		Permissions:     permissions,
	})
	if errors.Is(err, model.ErrNotFound) || errors.Is(err, model.PermissionCheckFailed) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (this *Mock) SetTopic(ctx context.Context, topic model.Topic) error {
	this.mux.Lock()
	defer this.mux.Unlock()
	for i, element := range this.topics {
		if element.Id == topic.Id {
			this.topics[i] = topic
			return nil
		}
	}
	this.topics = append(this.topics, topic)
	return nil
}

func (this *Mock) GetTopic(ctx context.Context, id string) (result model.Topic, exists bool, err error) {
	for _, element := range this.topics {
		if element.Id == id {
			return element, true, nil
		}
	}
	return result, false, nil
}

func (this *Mock) ListTopics(ctx context.Context, options model.ListOptions) (result []model.Topic, err error) {
	this.mux.Lock()
	defer this.mux.Unlock()
	return limitOffset(this.topics, options.Limit, options.Offset), nil
}

func (this *Mock) DeleteTopic(ctx context.Context, id string) error {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.topics = slices.DeleteFunc(this.topics, func(topic model.Topic) bool {
		return topic.Id == id
	})
	this.resources = slices.DeleteFunc(this.resources, func(element ResourceWithTime) bool {
		return element.TopicId == id
	})
	return nil
}
