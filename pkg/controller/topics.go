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
	"context"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/developer-notifications/pkg/client"
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller/com"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
	"log"
	"net/http"
	"slices"
	"time"
)

type TopicHandler struct {
	model.Topic
	com com.Com
}

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
		log.Println("ERROR: unable to send notification", err)
	}

	timeout := this.getTimeoutContext()
	err = this.db.DeleteTopic(timeout, id)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	err = this.stopTopicHandling(id, true)
	if err != nil {
		err = fmt.Errorf("unable to stop topic handling %v: %w", id, err)
		log.Println("ERROR:", err)
		this.notifyError(err)
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
		topic.Id = topic.KafkaTopic
	}
	if topic.KafkaTopic == "" && !topic.NoCqrs {
		topic.KafkaTopic = topic.Id
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
		Body:   fmt.Sprintf("update topic config for %v %v", topic.Id, topic.KafkaTopic),
	})
	if err != nil {
		log.Println("ERROR: unable to send notification", err)
	}

	err = this.db.SetTopic(timeout, topic)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}

	err = this.updateTopicHandling(topic, true)
	if err != nil {
		err = fmt.Errorf("unable to update topic handling %v %v: %w", topic.Id, topic.KafkaTopic, err)
		log.Println("ERROR:", err)
		this.notifyError(err)
		log.Println("try to reset old topic", old.Id, old.KafkaTopic, this.db.SetTopic(timeout, old), this.updateTopicHandling(old, true))
		return result, err, http.StatusInternalServerError
	}

	return topic, nil, http.StatusOK
}

func (this *Controller) startTopicUpdateWatcher(ctx context.Context) {
	dur := this.config.CheckDbTopicChangesInterval.GetDuration()
	if dur == 0 {
		return
	}
	ticker := time.NewTicker(dur)
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Println("refresh topics:", this.refreshTopics())
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

func (this *Controller) refreshTopics() error {
	dbTopics, err := this.db.ListTopics(this.getTimeoutContext(), model.ListOptions{})
	if err != nil {
		err = fmt.Errorf("unable to refresh topics: %w", err)
		this.notifyError(err)
		log.Println("ERROR: ListTopics(): %w", err)
		return err
	}
	updates := []model.Topic{}
	deletes := []string{}

	this.topicsMux.Lock()
	defer this.topicsMux.Unlock()
	if this.topics == nil {
		this.topics = map[string]TopicHandler{}
	}
	for _, topic := range dbTopics {
		if this.topics[topic.Id].LastUpdateUnixTimestamp < topic.LastUpdateUnixTimestamp { //if topic is not in this.topics LastUpdateUnixTimestamp will be initialized as 0 --> new topic wins
			updates = append(updates, topic)
		}
	}
	for id, _ := range this.topics {
		if !slices.ContainsFunc(dbTopics, func(topic model.Topic) bool {
			return topic.Id == id
		}) {
			deletes = append(deletes, id)
		}
	}

	for _, topic := range updates {
		err = this.updateTopicHandling(topic, false)
		if err != nil {
			err = fmt.Errorf("unable to update topic %v %v: %w", topic.Id, topic.KafkaTopic, err)
			log.Println("ERROR:", err)
			this.notifyError(err)
			return err
		}
	}
	for _, topicId := range deletes {
		err = this.stopTopicHandling(topicId, false)
		if err != nil {
			err = fmt.Errorf("unable to stop topic %v: %w", topicId, err)
			log.Println("ERROR:", err)
			this.notifyError(err)
			return err
		}
	}
	return nil
}

func (this *Controller) updateTopicHandling(topic model.Topic, lock bool) error {
	if lock {
		this.topicsMux.Lock()
		defer this.topicsMux.Unlock()
	}
	err := this.stopTopicHandling(topic.Id, false)
	if err != nil {
		return fmt.Errorf("unable to stop topic: %w", err)
	}
	wrapper, err := this.newTopicWrapper(topic)
	if err != nil {
		return fmt.Errorf("unable start topic kafka handling: %w", err)
	}
	this.topics[topic.Id] = wrapper
	return nil
}

func (this *Controller) stopTopicHandling(id string, lock bool) error {
	if lock {
		this.topicsMux.Lock()
		defer this.topicsMux.Unlock()
	}
	topic, ok := this.topics[id]
	if !ok {
		return nil
	}
	delete(this.topics, id)
	err := topic.Close()
	if err != nil {
		return err
	}
	return nil
}

func (this *Controller) newTopicWrapper(topic model.Topic) (result TopicHandler, err error) {
	if this.config.DisableCom {
		return result, errors.New("com is disabled")
	}
	c, err := this.com.Get(this.config, topic, this)
	if err != nil {
		return result, err
	}
	return TopicHandler{com: c, Topic: topic}, nil
}

func (this *TopicHandler) Close() (err error) {
	return this.com.Close()
}

func (this *TopicHandler) SendPermissions(ctx context.Context, id string, permissions model.ResourcePermissions) (err error) {
	return this.com.SendPermissions(ctx, this.Topic, id, permissions)
}
