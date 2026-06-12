/*
 * Copyright 2025 InfAI (CC SES)
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
	"net/http"
	"slices"

	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
)

func (this *Controller) Export(token string, options model.ImportExportOptions) (result model.ImportExport, err error, code int) {
	return this.ExportContext(context.TODO(), token, options)
}

func (this *Controller) ExportContext(ctx context.Context, token string, options model.ImportExportOptions) (result model.ImportExport, err error, code int) {
	jwtToken, err := jwt.Parse(token)
	if err != nil {
		return result, err, http.StatusBadRequest
	}
	if !jwtToken.IsAdmin() {
		return result, errors.New("only admins may export"), http.StatusForbidden
	}
	result = model.ImportExport{}

	if options.IncludeTopicConfig {
		result.Topics, err = this.db.ListTopics(this.getTimeoutContext(ctx), model.ListOptions{Ids: options.FilterTopics})
		if err != nil {
			return result, err, http.StatusInternalServerError
		}
	}

	if options.IncludePermissions {
		topics := options.FilterTopics
		if topics == nil {
			list := result.Topics
			if !options.IncludeTopicConfig {
				list, err = this.db.ListTopics(this.getTimeoutContext(ctx), model.ListOptions{Ids: options.FilterTopics})
				if err != nil {
					return result, err, http.StatusInternalServerError
				}
			}
			for _, topic := range list {
				topics = append(topics, topic.Id)
			}
		}
		for _, topic := range topics {
			list, err := this.db.AdminListResources(this.getTimeoutContext(ctx), topic, model.ListOptions{Ids: options.FilterResourceId})
			if err != nil {
				return result, err, http.StatusInternalServerError
			}
			for _, resource := range list {
				result.Permissions = append(result.Permissions, resource)
			}
		}
	}

	result.Sort()
	return result, nil, http.StatusOK
}

func (this *Controller) Import(token string, importModel model.ImportExport, options model.ImportExportOptions) (err error, code int) {
	return this.ImportContext(context.TODO(), token, importModel, options)
}

func (this *Controller) ImportContext(ctx context.Context, token string, importModel model.ImportExport, options model.ImportExportOptions) (err error, code int) {
	jwtToken, err := jwt.Parse(token)
	if err != nil {
		return err, http.StatusBadRequest
	}
	if !jwtToken.IsAdmin() {
		return errors.New("only admins may import"), http.StatusForbidden
	}

	if options.IncludeTopicConfig {
		for _, topic := range importModel.Topics {
			if options.FilterTopics == nil || slices.Contains(options.FilterTopics, topic.Id) {
				_, err, code = this.SetTopicContext(ctx, token, topic)
				if err != nil {
					return err, code
				}
			}
		}
	}

	if options.IncludePermissions {
		topicCache := map[string]model.Topic{}

		for _, resource := range importModel.Permissions {
			if (options.FilterTopics == nil || slices.Contains(options.FilterTopics, resource.TopicId)) && (options.FilterResourceId == nil || slices.Contains(options.FilterResourceId, resource.Id)) {
				if !resource.Valid() {
					return fmt.Errorf("invalid resource topic=%v id=%v", resource.TopicId, resource.Id), http.StatusBadRequest
				}

				topic, ok := topicCache[resource.TopicId]
				if !ok {
					var exists bool
					topic, exists, err = this.db.GetTopic(this.getTimeoutContext(ctx), resource.TopicId)
					if err != nil {
						return err, http.StatusInternalServerError
					}
					if !exists {
						return fmt.Errorf("invalid resource topic=%v id=%v: topic does not exist", resource.TopicId, resource.Id), http.StatusBadRequest
					}
					topicCache[resource.TopicId] = topic
				}

				err = this.setPermission(ctx, topic, resource)
				if err != nil {
					return err, http.StatusInternalServerError
				}
			}
		}
	}

	return nil, http.StatusOK
}
