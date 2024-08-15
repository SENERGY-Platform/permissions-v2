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
	"errors"
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller/idmodifier"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
	"golang.org/x/exp/slices"
	"net/http"
)

func (this *Controller) CheckPermission(tokenStr string, topicId string, id string, permissions ...model.Permission) (access bool, err error, code int) {
	token, err := jwt.Parse(tokenStr)
	if err != nil {
		return false, err, http.StatusUnauthorized
	}
	return this.checkPermission(token, topicId, id, permissions...)
}

func (this *Controller) checkPermission(token jwt.Token, topicId string, id string, permissions ...model.Permission) (access bool, err error, code int) {
	access, err, code = this.CheckTopicDefaultPermission(token, topicId, permissions)
	if err != nil {
		if code >= 500 {
			return access, err, code
		} else {
			//we don't want to tell scrapers if a resource id exists or not
			return false, nil, http.StatusOK
		}
	}
	if access {
		return true, nil, http.StatusOK
	}

	pureId, _ := idmodifier.SplitModifier(id)
	access, err = this.db.CheckResourcePermissions(this.getTimeoutContext(), topicId, pureId, token.GetUserId(), token.GetRoles(), token.GetGroups(), permissions...)
	if err != nil {
		return access, err, http.StatusInternalServerError
	}
	return access, nil, http.StatusOK
}

func (this *Controller) CheckMultiplePermissions(tokenStr string, topicId string, ids []string, permissions ...model.Permission) (accessMap map[string]bool, err error, code int) {
	token, err := jwt.Parse(tokenStr)
	if err != nil {
		return accessMap, err, http.StatusUnauthorized
	}
	pureIdList := []string{}
	pureIdToIds := map[string][]string{}
	for _, id := range ids {
		pureId, _ := idmodifier.SplitModifier(id)
		pureIdList = append(pureIdList, pureId)
		pureIdToIds[pureId] = append(pureIdToIds[pureId], id)
	}
	var pureAccess map[string]bool

	access, err, code := this.CheckTopicDefaultPermission(token, topicId, permissions)
	if err != nil {
		return accessMap, err, code
	}
	if access {
		pureAccess, err = this.db.CheckMultipleResourcePermissions(this.getTimeoutContext(), topicId, pureIdList, token.GetUserId(), token.GetRoles(), token.GetGroups())
	} else {
		pureAccess, err = this.db.CheckMultipleResourcePermissions(this.getTimeoutContext(), topicId, pureIdList, token.GetUserId(), token.GetRoles(), token.GetGroups(), permissions...)
	}
	if err != nil {
		return accessMap, err, http.StatusInternalServerError
	}
	accessMap = map[string]bool{}
	for key, value := range pureAccess {
		for _, id := range pureIdToIds[key] {
			accessMap[id] = value
		}
	}
	return accessMap, err, http.StatusOK
}

func (this *Controller) ListComputedPermissions(tokenStr string, topicId string, ids []string) (result []model.ComputedPermissions, err error, code int) {
	token, err := jwt.Parse(tokenStr)
	if err != nil {
		return result, err, http.StatusUnauthorized
	}
	isAdmin := token.IsAdmin()
	if ids == nil {
		ids = []string{}
	}

	topic, exists, err := this.db.GetTopic(this.getTimeoutContext(), topicId)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	if !exists {
		return result, errors.New("unknown topic"), http.StatusNotFound
	}

	resources, err := this.db.AdminListResources(this.getTimeoutContext(), topicId, model.ListOptions{
		Ids: ids,
	})
	if err != nil {
		return result, err, http.StatusInternalServerError
	}

	done := map[string]bool{}
	for _, resource := range resources {
		if isAdmin {
			result = append(result, model.ComputedPermissions{
				Id: resource.Id,
				PermissionsMap: model.PermissionsMap{
					Read:         true,
					Write:        true,
					Execute:      true,
					Administrate: true,
				},
			})
		} else {
			result = append(result, model.ComputedPermissions{
				Id:             resource.Id,
				PermissionsMap: ComputePermissionsMap(token, resource, topic.DefaultPermissions),
			})
		}
		done[resource.Id] = true
	}
	for _, id := range ids {
		if !done[id] {
			result = append(result, model.ComputedPermissions{
				Id: id,
				PermissionsMap: model.PermissionsMap{
					Read:         false,
					Write:        false,
					Execute:      false,
					Administrate: false,
				},
			})
		}
	}
	return result, nil, http.StatusOK
}

func ComputePermissionsMap(token jwt.Token, resource model.Resource, defaultPerm model.ResourcePermissions) (result model.PermissionsMap) {
	result = resource.UserPermissions[token.GetUserId()]

	defaultForUser, ok := defaultPerm.UserPermissions[token.GetUserId()]
	if ok {
		if defaultForUser.Read {
			result.Read = true
		}
		if defaultForUser.Write {
			result.Write = true
		}
		if defaultForUser.Execute {
			result.Execute = true
		}
		if defaultForUser.Administrate {
			result.Administrate = true
		}
	}

	for _, role := range token.GetRoles() {
		defaultForRole, ok := defaultPerm.RolePermissions[role]
		if ok {
			if defaultForRole.Read {
				result.Read = true
			}
			if defaultForRole.Write {
				result.Write = true
			}
			if defaultForRole.Execute {
				result.Execute = true
			}
			if defaultForRole.Administrate {
				result.Administrate = true
			}
		}

		rolePermissions := resource.RolePermissions[role]
		if rolePermissions.Read {
			result.Read = true
		}
		if rolePermissions.Write {
			result.Write = true
		}
		if rolePermissions.Execute {
			result.Execute = true
		}
		if rolePermissions.Administrate {
			result.Administrate = true
		}
	}
	for _, group := range token.GetGroups() {
		defaultForGroup, ok := defaultPerm.GroupPermissions[group]
		if ok {
			if defaultForGroup.Read {
				result.Read = true
			}
			if defaultForGroup.Write {
				result.Write = true
			}
			if defaultForGroup.Execute {
				result.Execute = true
			}
			if defaultForGroup.Administrate {
				result.Administrate = true
			}
		}

		groupPermissions := resource.GroupPermissions[group]
		if groupPermissions.Read {
			result.Read = true
		}
		if groupPermissions.Write {
			result.Write = true
		}
		if groupPermissions.Execute {
			result.Execute = true
		}
		if groupPermissions.Administrate {
			result.Administrate = true
		}
	}
	return result
}

func (this *Controller) CheckTopicDefaultPermission(token jwt.Token, topicId string, permissions model.PermissionList) (access bool, err error, code int) {
	access = token.IsAdmin()
	if access {
		return true, nil, http.StatusOK
	}
	topic, exists, err := this.db.GetTopic(this.getTimeoutContext(), topicId)
	if err != nil {
		return access, err, http.StatusInternalServerError
	}
	if !exists {
		return access, errors.New("unknown topic"), http.StatusNotFound
	}
	access, err = this.checkTopicDefaultPermission(token, topic, permissions)
	if err != nil {
		return access, err, http.StatusInternalServerError
	}
	return access, nil, http.StatusOK
}

func (this *Controller) checkTopicDefaultPermission(token jwt.Token, topic model.Topic, permissions model.PermissionList) (access bool, err error) {
	access = token.IsAdmin()
	if access {
		return true, nil
	}
	user := token.GetUserId()
	groups := token.GetGroups()
	roles := token.GetRoles()
	for _, permission := range permissions {
		switch permission {
		case model.Read:
			if topic.DefaultPermissions.UserPermissions[user].Read {
				continue
			}
			if slices.ContainsFunc(roles, func(role string) bool {
				return topic.DefaultPermissions.RolePermissions[role].Read
			}) {
				continue
			}
			if slices.ContainsFunc(groups, func(group string) bool {
				return topic.DefaultPermissions.GroupPermissions[group].Read
			}) {
				continue
			}
			return false, nil
		case model.Administrate:
			if topic.DefaultPermissions.UserPermissions[user].Administrate {
				continue
			}
			if slices.ContainsFunc(roles, func(role string) bool {
				return topic.DefaultPermissions.RolePermissions[role].Administrate
			}) {
				continue
			}
			if slices.ContainsFunc(groups, func(group string) bool {
				return topic.DefaultPermissions.GroupPermissions[group].Administrate
			}) {
				continue
			}
			return false, nil
		case model.Write:
			if topic.DefaultPermissions.UserPermissions[user].Write {
				continue
			}
			if slices.ContainsFunc(roles, func(role string) bool {
				return topic.DefaultPermissions.RolePermissions[role].Write
			}) {
				continue
			}
			if slices.ContainsFunc(groups, func(group string) bool {
				return topic.DefaultPermissions.GroupPermissions[group].Write
			}) {
				continue
			}
			return false, nil
		case model.Execute:
			if topic.DefaultPermissions.UserPermissions[user].Execute {
				continue
			}
			if slices.ContainsFunc(roles, func(role string) bool {
				return topic.DefaultPermissions.RolePermissions[role].Execute
			}) {
				continue
			}
			if slices.ContainsFunc(groups, func(group string) bool {
				return topic.DefaultPermissions.GroupPermissions[group].Execute
			}) {
				continue
			}
			return false, nil
		}
	}
	return true, nil
}
