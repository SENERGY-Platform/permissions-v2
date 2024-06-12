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

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/SENERGY-Platform/permissions-v2/pkg/api"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller"
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller/com"
	"github.com/SENERGY-Platform/permissions-v2/pkg/database/mock"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Client interface {
	api.Controller
}

func New(serverUrl string) (client Client) {
	return &Impl{serverUrl: serverUrl}
}

func NewTestClient(ctx context.Context) (client Client, err error) {
	return controller.NewWithDependencies(ctx,
		configuration.Config{},
		mock.New(),
		com.NewBypassProvider())
}

type Impl struct {
	serverUrl string
}

func (this *Impl) ListTopics(token string, options model.ListOptions) (result []model.Topic, err error, code int) {
	query := url.Values{}
	if options.Limit > 0 {
		query.Set("limit", strconv.FormatInt(options.Limit, 10))
	}
	if options.Offset > 0 {
		query.Set("offset", strconv.FormatInt(options.Offset, 10))
	}
	req, err := http.NewRequest(http.MethodGet, this.serverUrl+"/admin/topics?"+query.Encode(), nil)
	if err != nil {
		return result, err, 0
	}
	return do[[]model.Topic](token, req)
}

func (this *Impl) GetTopic(token string, id string) (result model.Topic, err error, code int) {
	req, err := http.NewRequest(http.MethodGet, this.serverUrl+"/admin/topics/"+url.PathEscape(id), nil)
	if err != nil {
		return result, err, 0
	}
	return do[model.Topic](token, req)
}

func (this *Impl) RemoveTopic(token string, id string) (err error, code int) {
	req, err := http.NewRequest(http.MethodDelete, this.serverUrl+"/admin/topics/"+url.PathEscape(id), nil)
	if err != nil {
		return err, 0
	}
	return doVoid(token, req)
}

func (this *Impl) SetTopic(token string, topic model.Topic) (result model.Topic, err error, code int) {
	body, err := json.Marshal(topic)
	if err != nil {
		return result, err, 0
	}
	req, err := http.NewRequest(http.MethodPut, this.serverUrl+"/admin/topics/"+url.PathEscape(topic.Id), bytes.NewBuffer(body))
	if err != nil {
		return result, err, 0
	}
	return do[model.Topic](token, req)
}

func (this *Impl) CheckPermission(token string, topicId string, id string, permissions ...model.Permission) (access bool, err error, code int) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/check/%v/%v?permissions=%v", this.serverUrl, url.PathEscape(topicId), url.PathEscape(id), model.PermissionList(permissions).Encode()), nil)
	if err != nil {
		return access, err, 0
	}
	return do[bool](token, req)
}

func (this *Impl) CheckMultiplePermissions(token string, topicId string, ids []string, permissions ...model.Permission) (access map[string]bool, err error, code int) {
	query := url.Values{}
	query.Set("permissions", model.PermissionList(permissions).Encode())
	query.Set("ids", strings.Join(ids, ","))
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/check/%v?%v", this.serverUrl, url.PathEscape(topicId), query.Encode()), nil)
	if err != nil {
		return access, err, 0
	}
	return do[map[string]bool](token, req)
}

func (this *Impl) ListAccessibleResourceIds(token string, topicId string, options model.ListOptions, permissions ...model.Permission) (ids []string, err error, code int) {
	query := url.Values{}
	query.Set("permissions", model.PermissionList(permissions).Encode())
	if options.Limit > 0 {
		query.Set("limit", strconv.FormatInt(options.Limit, 10))
	}
	if options.Offset > 0 {
		query.Set("offset", strconv.FormatInt(options.Offset, 10))
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/accessible/%v?%v", this.serverUrl, url.PathEscape(topicId), query.Encode()), nil)
	if err != nil {
		return ids, err, 0
	}
	return do[[]string](token, req)
}

func (this *Impl) ListResourcesWithAdminPermission(token string, topicId string, options model.ListOptions) (result []model.Resource, err error, code int) {
	query := url.Values{}
	if options.Limit > 0 {
		query.Set("limit", strconv.FormatInt(options.Limit, 10))
	}
	if options.Offset > 0 {
		query.Set("offset", strconv.FormatInt(options.Offset, 10))
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/manage/%v?%v", this.serverUrl, url.PathEscape(topicId), query.Encode()), nil)
	if err != nil {
		return result, err, 0
	}
	return do[[]model.Resource](token, req)
}

func (this *Impl) GetResource(token string, topicId string, id string) (result model.Resource, err error, code int) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/manage/%v/%v", this.serverUrl, url.PathEscape(topicId), url.PathEscape(id)), nil)
	if err != nil {
		return result, err, 0
	}
	return do[model.Resource](token, req)
}

func (this *Impl) RemoveResource(token string, topicId string, id string) (err error, code int) {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%v/manage/%v/%v", this.serverUrl, url.PathEscape(topicId), url.PathEscape(id)), nil)
	if err != nil {
		return err, 0
	}
	return doVoid(token, req)
}

func (this *Impl) SetPermission(token string, topicId string, id string, permissions model.ResourcePermissions, options model.SetPermissionOptions) (result model.ResourcePermissions, err error, code int) {
	body, err := json.Marshal(permissions)
	if err != nil {
		return result, err, 0
	}
	query := ""
	if options.Wait {
		query = "?wait=true"
	}
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%v/manage/%v/%v%v", this.serverUrl, url.PathEscape(topicId), url.PathEscape(id), query), bytes.NewReader(body))
	if err != nil {
		return result, err, 0
	}
	return do[model.ResourcePermissions](token, req)
}

func do[T any](token string, req *http.Request) (result T, err error, code int) {
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		temp, _ := io.ReadAll(resp.Body) //read error response end ensure that resp.Body is read to EOF
		return result, fmt.Errorf("unexpected statuscode %v: %v", resp.StatusCode, string(temp)), resp.StatusCode
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		_, _ = io.ReadAll(resp.Body) //ensure resp.Body is read to EOF
		return result, err, http.StatusInternalServerError
	}
	return result, nil, resp.StatusCode
}

func doVoid(token string, req *http.Request) (err error, code int) {
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		temp, _ := io.ReadAll(resp.Body) //read error response end ensure that resp.Body is read to EOF
		return fmt.Errorf("unexpected statuscode %v: %v", resp.StatusCode, string(temp)), resp.StatusCode
	}
	return nil, resp.StatusCode
}
