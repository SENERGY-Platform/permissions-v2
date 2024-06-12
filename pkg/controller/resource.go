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
	"log"
	"net/http"
	"time"
)

func (this *Controller) HandleResourceUpdate(topic model.Topic, id string, owner string) error {
	if this.config.Debug {
		log.Println("handle resource update command", topic.Id, id)
	}
	resource := model.Resource{
		Id:      id,
		TopicId: topic.Id,
		ResourcePermissions: model.ResourcePermissions{
			UserPermissions:  map[string]model.PermissionsMap{owner: {Read: true, Write: true, Execute: true, Administrate: true}},
			GroupPermissions: map[string]model.PermissionsMap{},
		},
	}
	for _, gr := range topic.InitialGroupPermissions {
		resource.GroupPermissions[gr.GroupName] = gr.PermissionsMap
	}
	//init resource permissions; with time.Time{} and preventOlderUpdates=true we guarantee that no existing resource is overwritten
	_, err := this.db.SetResourcePermissions(this.getTimeoutContext(), resource, time.Time{}, true)
	return err
}

func (this *Controller) HandleResourceDelete(topic model.Topic, id string) error {
	if this.config.Debug {
		log.Println("handle resource delete command", topic.Id, id)
	}
	return this.db.DeleteResource(this.getTimeoutContext(), topic.Id, id)
}

func (this *Controller) AdminListResourceIds(tokenStr string, topicId string, options model.ListOptions) (ids []string, err error, code int) {
	token, err := jwt.Parse(tokenStr)
	if err != nil {
		return ids, err, http.StatusUnauthorized
	}
	if !token.IsAdmin() {
		return ids, errors.New("only admins may use this method"), http.StatusUnauthorized
	}
	ids, err = this.db.AdminListResourceIds(this.getTimeoutContext(), topicId, options)
	if err != nil {
		code = http.StatusInternalServerError
	} else {
		code = http.StatusOK
	}
	return ids, err, code
}

func (this *Controller) ListAccessibleResourceIds(tokenStr string, topicId string, options model.ListOptions, permission ...model.Permission) (ids []string, err error, code int) {
	token, err := jwt.Parse(tokenStr)
	if err != nil {
		return ids, err, http.StatusUnauthorized
	}
	ids, err = this.db.ListResourceIdsByPermissions(this.getTimeoutContext(), topicId, token.GetUserId(), token.GetRoles(), options, permission...)
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
	result, err = this.db.ListResourcesByPermissions(this.getTimeoutContext(), topicId, token.GetUserId(), token.GetRoles(), options, model.Administrate)
	if err != nil {
		code = http.StatusInternalServerError
	} else {
		code = http.StatusOK
	}
	return result, err, code
}

func (this *Controller) RemoveResource(tokenStr string, topicId string, id string) (err error, code int) {
	token, err := jwt.Parse(tokenStr)
	if err != nil {
		return err, http.StatusUnauthorized
	}
	pureId, _ := idmodifier.SplitModifier(id)
	_, err = this.db.GetResource(this.getTimeoutContext(), topicId, pureId, model.GetOptions{
		CheckPermission: !token.IsAdmin(), //admins may access without stored permission
		UserId:          token.GetUserId(),
		GroupIds:        token.GetRoles(),
		Permissions:     model.PermissionList{model.Administrate},
	})
	if errors.Is(err, model.PermissionCheckFailed) {
		return errors.New("access denied"), http.StatusForbidden
	}
	if errors.Is(err, model.ErrNotFound) {
		return nil, http.StatusOK
	}
	if err != nil {
		return err, http.StatusInternalServerError
	}
	err = func() error {
		this.topicsMux.RLock()
		defer this.topicsMux.RUnlock()
		if !this.topics[topicId].NoCqrs {
			return errors.New("cqrs resources may only deleted by cqrs commands")
		}
		return nil
	}()
	if err != nil {
		return err, http.StatusBadRequest
	}

	err = this.db.DeleteResource(this.getTimeoutContext(), topicId, id)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
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
		Permissions:     model.PermissionList{model.Administrate},
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
	topic, exists, err := this.db.GetTopic(this.getTimeoutContext(), topicId)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	if !exists {
		return result, errors.New("topic does not exist"), http.StatusBadRequest
	}
	pureId, _ := idmodifier.SplitModifier(id)
	if !token.IsAdmin() {
		accessMap, err := this.db.CheckMultipleResourcePermissions(this.getTimeoutContext(), topicId, []string{pureId}, token.GetUserId(), token.GetRoles(), model.Administrate)
		if err != nil {
			return result, err, http.StatusInternalServerError
		}
		access, ok := accessMap[pureId]
		if !ok && topic.InitOnlyByCqrs {
			return result, errors.New("resource may only be initialized by resource cqrs command"), http.StatusForbidden
		}
		if ok && !access {
			return result, errors.New("access denied"), http.StatusForbidden
		}
	}

	if !permissions.Valid() {
		return result, errors.New("invalid permissions"), http.StatusBadRequest
	}

	wait := func() error { return nil }

	err, code = func() (err error, code int) {
		this.topicsMux.RLock()
		defer this.topicsMux.RUnlock()
		wrapper, ok := this.topics[topicId]
		if !ok {
			return errors.New("unknown topic id"), http.StatusBadRequest
		}

		wait = this.optionalWait(options.Wait && !topic.NoCqrs, wrapper.KafkaTopic, pureId)

		err = wrapper.SendPermissions(this.getTimeoutContext(), pureId, permissions)
		if err != nil {
			return err, http.StatusInternalServerError
		}
		return nil, http.StatusOK
	}()
	if err != nil {
		return permissions, err, code
	}

	err = wait()
	if err != nil {
		return permissions, err, http.StatusInternalServerError
	}

	return permissions, err, code
}
