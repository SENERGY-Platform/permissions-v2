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
	"time"
)

func (this *Controller) HandleResourceUpdate(topic model.Topic, id string, owner string) error {
	resource := model.Resource{
		Id:      id,
		TopicId: topic.Id,
		ResourcePermissions: model.ResourcePermissions{
			UserPermissions:  map[string]model.Permissions{owner: {Read: true, Write: true, Execute: true, Administrate: true}},
			GroupPermissions: map[string]model.Permissions{},
		},
	}
	for _, gr := range topic.InitialGroupRights {
		resource.GroupPermissions[gr.GroupName] = gr.Permissions
	}
	//init resource permissions; with time.Time{} and preventOlderUpdates=true we guarantee that no existing resource is overwritten
	_, err := this.db.SetResourcePermissions(this.getTimeoutContext(), resource, time.Time{}, true)
	return err
}

func (this *Controller) HandleResourceDelete(topic model.Topic, id string) error {
	return this.db.DeleteResource(this.getTimeoutContext(), topic.Id, id)
}

func (this *Controller) ListAccessibleResourceIds(tokenStr string, topicId string, permissions string, options model.ListOptions) (ids []string, err error, code int) {
	token, err := jwt.Parse(tokenStr)
	if err != nil {
		return ids, err, http.StatusUnauthorized
	}
	ids, err = this.db.ListResourceIdsByPermissions(this.getTimeoutContext(), topicId, token.GetUserId(), token.GetRoles(), permissions, options)
	if err != nil {
		code = http.StatusInternalServerError
	} else {
		code = http.StatusOK
	}
	return ids, err, code
}

func (this *Controller) ListResourcesWithAdminPermission(tokenStr string, topicId string, options model.ListOptions) (result []model.Resource, err error, code int) {
	token, err := jwt.Parse(tokenStr)
	if err != nil {
		return result, err, http.StatusUnauthorized
	}
	result, err = this.db.ListResourcesByPermissions(this.getTimeoutContext(), topicId, token.GetUserId(), token.GetRoles(), "a", options)
	if err != nil {
		code = http.StatusInternalServerError
	} else {
		code = http.StatusOK
	}
	return result, err, code
}

func (this *Controller) GetResource(tokenStr string, topicId string, id string) (result model.Resource, err error, code int) {
	token, err := jwt.Parse(tokenStr)
	if err != nil {
		return result, err, http.StatusUnauthorized
	}
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

func (this *Controller) SetPermission(tokenStr string, topicId string, id string, permissions model.ResourcePermissions, options model.SetPermissionOptions) (result model.ResourcePermissions, err error, code int) {
	token, err := jwt.Parse(tokenStr)
	if err != nil {
		return result, err, http.StatusUnauthorized
	}
	pureId, _ := idmodifier.SplitModifier(id)
	if !token.IsAdmin() {
		access, err, code := this.checkPermission(token, topicId, pureId, "a")
		if err != nil {
			return result, err, code
		}
		if !access {
			return result, errors.New("access denied"), http.StatusForbidden
		}
	}

	wait := func() error { return nil }

	err, code = func() (err error, code int) {
		this.topicsMux.RLock()
		defer this.topicsMux.RUnlock()
		wrapper, ok := this.topics[pureId]
		if !ok {
			return errors.New("unknown topic id"), http.StatusBadRequest
		}

		wait = this.optionalWait(options.Wait, wrapper.KafkaTopic, pureId)

		err = wrapper.SendPermissions(this.getTimeoutContext(), pureId, permissions)
		if err != nil {
			return err, http.StatusInternalServerError
		}
		return nil, http.StatusOK
	}()

	err = wait()
	if err != nil {
		return permissions, err, http.StatusInternalServerError
	}

	return permissions, err, code
}
