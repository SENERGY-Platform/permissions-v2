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
	"net/http"
)

func (this *Controller) ListAccessibleResourceIds(token jwt.Token, topicId string, permissions string, options model.ListOptions) (ids []string, err error, code int) {
	ids, err = this.db.ListResourceIdsByPermissions(this.getTimeoutContext(), topicId, token.GetUserId(), token.GetRoles(), permissions, options)
	if err != nil {
		code = http.StatusInternalServerError
	} else {
		code = http.StatusOK
	}
	return ids, err, code
}

func (this *Controller) ListResourcesWithAdminPermission(token jwt.Token, topicId string, options model.ListOptions) (result []model.Resource, err error, code int) {
	result, err = this.db.ListResourcesByPermissions(this.getTimeoutContext(), topicId, token.GetUserId(), token.GetRoles(), "a", options)
	if err != nil {
		code = http.StatusInternalServerError
	} else {
		code = http.StatusOK
	}
	return result, err, code
}

func (this *Controller) GetResource(token jwt.Token, topicId string, id string) (result model.Resource, err error, code int) {
	pureId, _ := idmodifier.SplitModifier(id)
	result, err = this.db.GetResource(this.getTimeoutContext(), topicId, pureId, model.GetOptions{
		CheckPermission: !token.IsAdmin(), //admins may access without stored permission
		UserId:          token.GetUserId(),
		GroupIds:        token.GetRoles(),
		Permission:      "a",
	})
	if errors.Is(err, model.PermissionCheckFailed) {
		return result, errors.New("access denied"), http.StatusForbidden
	}
	if errors.Is(err, model.ErrNotFound) {
		return result, err, http.StatusNotFound
	}
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	result.Id = id
	return result, nil, http.StatusOK
}

func (this *Controller) SetPermission(token jwt.Token, topicId string, id string, permissions model.ResourcePermissions) (result model.ResourcePermissions, err error, code int) {
	pureId, _ := idmodifier.SplitModifier(id)
	if !token.IsAdmin() {
		access, err, code := this.CheckPermission(token, topicId, pureId, "a")
		if err != nil {
			return result, err, code
		}
		if !access {
			return result, errors.New("access denied"), http.StatusForbidden
		}
	}

	this.topicsMux.RLock()
	defer this.topicsMux.RUnlock()
	wrapper, ok := this.topics[pureId]
	if !ok {
		return result, errors.New("unknown topic id"), http.StatusBadRequest
	}
	err = wrapper.SendPermissions(this.getTimeoutContext(), pureId, permissions)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	return permissions, nil, http.StatusOK
}
