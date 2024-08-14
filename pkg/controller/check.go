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
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller/idmodifier"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
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
	pureId, _ := idmodifier.SplitModifier(id)
	access, err = this.db.CheckResourcePermissions(this.getTimeoutContext(), topicId, pureId, token.GetUserId(), token.GetRoles(), token.GetGroups(), permissions...)
	if err != nil {
		code = http.StatusInternalServerError
	} else {
		code = http.StatusOK
	}
	return access, err, code
}

func (this *Controller) CheckMultiplePermissions(tokenStr string, topicId string, ids []string, permissions ...model.Permission) (access map[string]bool, err error, code int) {
	token, err := jwt.Parse(tokenStr)
	if err != nil {
		return access, err, http.StatusUnauthorized
	}
	pureIdList := []string{}
	pureIdToIds := map[string][]string{}
	for _, id := range ids {
		pureId, _ := idmodifier.SplitModifier(id)
		pureIdList = append(pureIdList, pureId)
		pureIdToIds[pureId] = append(pureIdToIds[pureId], id)
	}
	var pureAccess map[string]bool
	pureAccess, err = this.db.CheckMultipleResourcePermissions(this.getTimeoutContext(), topicId, pureIdList, token.GetUserId(), token.GetRoles(), token.GetGroups(), permissions...)
	if err != nil {
		code = http.StatusInternalServerError
	} else {
		code = http.StatusOK
	}
	access = map[string]bool{}
	for key, value := range pureAccess {
		for _, id := range pureIdToIds[key] {
			access[id] = value
		}
	}
	return access, err, code
}

func (this *Controller) ListComputedPermissions(tokenStr string, topic string, ids []string) (result []model.ComputedPermissions, err error, code int) {
	token, err := jwt.Parse(tokenStr)
	if err != nil {
		return result, err, http.StatusUnauthorized
	}
	if ids == nil {
		ids = []string{}
	}
	resources, err := this.db.AdminListResources(this.getTimeoutContext(), topic, model.ListOptions{
		Ids: ids,
	})
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	done := map[string]bool{}
	for _, resource := range resources {
		result = append(result, model.ComputedPermissions{
			Id:             resource.Id,
			PermissionsMap: ComputePermissionsMap(token, resource),
		})
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

func ComputePermissionsMap(token jwt.Token, resource model.Resource) (result model.PermissionsMap) {
	result = resource.UserPermissions[token.GetUserId()]
	for _, role := range token.GetRoles() {
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
