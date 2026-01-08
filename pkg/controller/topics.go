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
	"fmt"
	"net/http"

	"github.com/SENERGY-Platform/developer-notifications/pkg/client"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
)

func (this *Controller) ListTopics(tokenStr string, options model.ListOptions) (result []model.Topic, err error, code int) {
	token, err := jwt.Parse(tokenStr)
	if err != nil {
		return result, err, http.StatusUnauthorized
	}
	if !token.IsAdmin() {
		return result, errors.New("only admins may manage topics"), http.StatusUnauthorized
	}
	timeout := this.getTimeoutContext()
	result, err = this.db.ListTopics(timeout, options)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	return result, nil, http.StatusOK
}

func (this *Controller) GetTopic(tokenStr string, id string) (result model.Topic, err error, code int) {
	token, err := jwt.Parse(tokenStr)
	if err != nil {
		return result, err, http.StatusUnauthorized
	}
	if !token.IsAdmin() {
		return result, errors.New("only admins may manage topics"), http.StatusUnauthorized
	}
	timeout := this.getTimeoutContext()
	var exists bool
	result, exists, err = this.db.GetTopic(timeout, id)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	if !exists {
		return result, errors.New("topic does not exist"), http.StatusNotFound
	}
	return result, nil, http.StatusOK
}

func (this *Controller) RemoveTopic(tokenStr string, id string) (err error, code int) {
	token, err := jwt.Parse(tokenStr)
	if err != nil {
		return err, http.StatusUnauthorized
	}
	if !token.IsAdmin() {
		return errors.New("only admins may manage topics"), http.StatusUnauthorized
	}

	err = this.notifier.SendMessage(client.Message{
		Sender: "github.com/SENERGY-Platform/permissions-v2",
		Title:  "PermissionsV2 Remove Topic Config",
		Tags:   []string{"permissions", "topic"},
		Body:   fmt.Sprintf("update topic config for %v", id),
	})
	if err != nil {
		this.config.GetLogger().Error("unable to send notification", "error", err)
	}

	timeout := this.getTimeoutContext()
	err = this.db.DeleteTopic(timeout, id)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func (this *Controller) SetTopic(tokenStr string, topic model.Topic) (result model.Topic, err error, code int) {
	token, err := jwt.Parse(tokenStr)
	if err != nil {
		return result, err, http.StatusUnauthorized
	}
	if !token.IsAdmin() {
		return result, errors.New("only admins may manage topics"), http.StatusUnauthorized
	}

	if topic.Id == "" {
		topic.Id = topic.PublishToKafkaTopic
	}

	err = topic.Validate()
	if err != nil {
		return result, fmt.Errorf("invalid topic: %w", err), http.StatusBadRequest
	}

	timeout := this.getTimeoutContext()
	old, exists, err := this.db.GetTopic(timeout, topic.Id)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	if exists && old.Equal(topic) {
		return old, nil, http.StatusAccepted
	}

	err = this.notifier.SendMessage(client.Message{
		Sender: "github.com/SENERGY-Platform/permissions-v2",
		Title:  "PermissionsV2 Update Topic Config",
		Tags:   []string{"permissions", "topic"},
		Body:   fmt.Sprintf("update topic config for %v %v", topic.Id, topic.PublishToKafkaTopic),
	})
	if err != nil {
		this.config.GetLogger().Error("unable to send notification", "error", err)
	}

	err = this.db.SetTopic(timeout, topic)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}

	return topic, nil, http.StatusOK
}
