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
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller/idmodifier"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
	"io"
	"log"
	"net/http"
	"slices"
	"time"
)

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
	access, err, code := this.CheckTopicDefaultPermission(token, topicId, permission)
	if err != nil {
		return ids, err, code
	}
	if access {
		ids, err = this.db.AdminListResourceIds(this.getTimeoutContext(), topicId, options)
	} else {
		ids, err = this.db.ListResourceIdsByPermissions(this.getTimeoutContext(), topicId, token.GetUserId(), token.GetRoles(), token.GetGroups(), options, permission...)
	}
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

	access, err, code := this.CheckTopicDefaultPermission(token, topicId, model.PermissionList{model.Administrate})
	if err != nil {
		return result, err, code
	}
	if access {
		result, err = this.db.AdminListResources(this.getTimeoutContext(), topicId, options)
	} else {
		result, err = this.db.ListResourcesByPermissions(this.getTimeoutContext(), topicId, token.GetUserId(), token.GetRoles(), token.GetGroups(), options, model.Administrate)
	}

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

	access := token.IsAdmin()
	if !access {
		return errors.New("access denied: only admin roles may delete resources"), http.StatusForbidden
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

	access, err, code := this.CheckTopicDefaultPermission(token, topicId, model.PermissionList{model.Administrate})
	if err != nil {
		return result, err, code
	}

	result, err = this.db.GetResource(this.getTimeoutContext(), topicId, pureId, model.GetOptions{
		CheckPermission: !access, //admins may access without stored permission
		UserId:          token.GetUserId(),
		RoleIds:         token.GetRoles(),
		GroupIds:        token.GetGroups(),
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

func (this *Controller) SetPermission(tokenStr string, topicId string, id string, permissions model.ResourcePermissions) (result model.ResourcePermissions, err error, code int) {
	token, err := jwt.Parse(tokenStr)
	if err != nil {
		return result, err, http.StatusUnauthorized
	}

	topic, exists, err := this.db.GetTopic(this.getTimeoutContext(), topicId)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	if !exists {
		return result, errors.New("unknown topic"), http.StatusNotFound
	}

	access, err := this.checkTopicDefaultPermission(token, topic, model.PermissionList{model.Administrate})
	if err != nil {
		return result, err, http.StatusInternalServerError
	}

	pureId, _ := idmodifier.SplitModifier(id)
	if !access {
		access, err := this.db.CheckResourcePermissions(this.getTimeoutContext(), topicId, pureId, token.GetUserId(), token.GetRoles(), token.GetGroups(), model.Administrate)
		if err != nil {
			return result, err, http.StatusInternalServerError
		}
		if !access {
			return result, errors.New("access denied"), http.StatusForbidden
		}
	}

	if !permissions.Valid() {
		return result, errors.New("invalid permissions"), http.StatusBadRequest
	}
	err, code = this.checkGroupMembership(token, topic.Id, id, permissions)
	if err != nil {
		return result, err, code
	}

	publish := topic.PublishToKafkaTopic != "" && topic.PublishToKafkaTopic != "-"

	err = this.db.SetResource(this.getTimeoutContext(), model.Resource{
		Id:                  pureId,
		TopicId:             topic.Id,
		ResourcePermissions: permissions,
	}, time.Now(), !publish)

	if err != nil {
		return result, err, http.StatusInternalServerError
	}

	if publish {
		err = this.publishPermission(topic, pureId, permissions)
		if err != nil {
			log.Println("WARNING: unable to publish permissions update to", topic.PublishToKafkaTopic)
			this.notifyError(fmt.Errorf("unable to publish permissions update to %v; publish will be retried", topic.PublishToKafkaTopic))
			return permissions, nil, http.StatusOK
		} else {
			err = this.db.MarkResourceAsSynced(this.getTimeoutContext(), topic.Id, pureId)
			if err != nil {
				log.Println("WARNING: unable to mark resource as synced", topic.Id, pureId)
			}
		}
	}

	return permissions, err, http.StatusOK
}

func (this *Controller) checkGroupMembership(token jwt.Token, topicId string, id string, permissions model.ResourcePermissions) (error, int) {
	if token.IsAdmin() {
		return nil, http.StatusOK
	}
	current, err := this.db.GetResource(this.getTimeoutContext(), topicId, id, model.GetOptions{})
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return err, http.StatusInternalServerError
	}
	if current.UserPermissions == nil {
		current.UserPermissions = map[string]model.PermissionsMap{}
	}
	if current.GroupPermissions == nil {
		current.GroupPermissions = map[string]model.PermissionsMap{}
	}
	addedUsers := []string{}
	addedGroups := []string{}
	for user, _ := range permissions.UserPermissions {
		if _, ok := current.UserPermissions[user]; !ok {
			addedUsers = append(addedUsers, user)
		}
	}
	for group, _ := range permissions.GroupPermissions {
		if _, ok := current.GroupPermissions[group]; !ok {
			addedGroups = append(addedGroups, group)
		}
	}
	for _, group := range addedGroups {
		if !token.HasGroup(group) {
			return fmt.Errorf("requesting user not in added group '%v'", group), http.StatusBadRequest
		}
	}
	if len(addedUsers) > 0 && this.config.UserManagementUrl != "" && this.config.UserManagementUrl != "-" {
		usersInSameGroup, err := this.getUsersInSameGroup(token)
		if err != nil {
			return err, http.StatusInternalServerError
		}
		for _, addedUser := range addedUsers {
			if !slices.ContainsFunc(usersInSameGroup, func(user User) bool {
				return user.Id == addedUser
			}) {
				return fmt.Errorf("added user '%v' nat in the same group as the requesting user", addedUser), http.StatusBadRequest
			}
		}
	}

	return nil, http.StatusOK
}

type User struct {
	Id   string `json:"id"`
	Name string `json:"username"`
}

func (this *Controller) getUsersInSameGroup(token jwt.Token) (users []User, err error) {
	req, err := http.NewRequest("GET", this.config.UserManagementUrl+"/user-list", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", token.Jwt())
	client := &http.Client{
		Timeout: time.Minute,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error in user-management /user-list request: %v %v", resp.StatusCode, string(msg))
	}
	err = json.NewDecoder(resp.Body).Decode(&users)
	if err != nil {
		return nil, fmt.Errorf("error while decoding user-management /user-list response: %w", err)
	}
	return users, nil
}
