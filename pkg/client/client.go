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
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/SENERGY-Platform/gin-middleware/otelx"
	"github.com/SENERGY-Platform/permissions-v2/pkg/api"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller"
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller/kafka"
	"github.com/SENERGY-Platform/permissions-v2/pkg/database/mock"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
)

type Client interface {
	api.Controller
}

func New(serverUrl string) (client *ClientImpl) {
	return &ClientImpl{serverUrl: serverUrl}
}

func NewTestClient(ctx context.Context) (client *controller.Controller, err error) {
	return controller.NewWithDependencies(ctx,
		configuration.Config{},
		mock.New(),
		kafka.NewVoidProducerProvider())
}

type ClientImpl struct {
	serverUrl string
}

func (this *ClientImpl) ListTopics(token string, options ListOptions) (result []Topic, err error, code int) {
	return this.ListTopicsContext(context.TODO(), token, options)
}

func (this *ClientImpl) ListTopicsContext(ctx context.Context, token string, options ListOptions) (result []Topic, err error, code int) {
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
	return doWithContext[[]Topic](ctx, token, req)
}

func (this *ClientImpl) GetTopic(token string, id string) (result Topic, err error, code int) {
	return this.GetTopicContext(context.TODO(), token, id)
}

func (this *ClientImpl) GetTopicContext(ctx context.Context, token string, id string) (result Topic, err error, code int) {
	req, err := http.NewRequest(http.MethodGet, this.serverUrl+"/admin/topics/"+url.PathEscape(id), nil)
	if err != nil {
		return result, err, 0
	}
	return doWithContext[Topic](ctx, token, req)
}

func (this *ClientImpl) RemoveTopic(token string, id string) (err error, code int) {
	return this.RemoveTopicContext(context.TODO(), token, id)
}

func (this *ClientImpl) RemoveTopicContext(ctx context.Context, token string, id string) (err error, code int) {
	req, err := http.NewRequest(http.MethodDelete, this.serverUrl+"/admin/topics/"+url.PathEscape(id), nil)
	if err != nil {
		return err, 0
	}
	return doVoidWithContext(ctx, token, req)
}

func (this *ClientImpl) SetTopic(token string, topic Topic) (result Topic, err error, code int) {
	return this.SetTopicContext(context.TODO(), token, topic)
}

func (this *ClientImpl) SetTopicContext(ctx context.Context, token string, topic Topic) (result Topic, err error, code int) {
	body, err := json.Marshal(topic)
	if err != nil {
		return result, err, 0
	}
	req, err := http.NewRequest(http.MethodPut, this.serverUrl+"/admin/topics/"+url.PathEscape(topic.Id), bytes.NewBuffer(body))
	if err != nil {
		return result, err, 0
	}
	return doWithContext[Topic](ctx, token, req)
}

func (this *ClientImpl) CheckPermission(token string, topicId string, id string, permissions ...Permission) (access bool, err error, code int) {
	return this.CheckPermissionContext(context.TODO(), token, topicId, id, permissions...)
}

func (this *ClientImpl) CheckPermissionContext(ctx context.Context, token string, topicId string, id string, permissions ...Permission) (access bool, err error, code int) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/check/%v/%v?permissions=%v", this.serverUrl, url.PathEscape(topicId), url.PathEscape(id), PermissionList(permissions).Encode()), nil)
	if err != nil {
		return access, err, 0
	}
	return doWithContext[bool](ctx, token, req)
}

func (this *ClientImpl) CheckMultiplePermissions(token string, topicId string, ids []string, permissions ...Permission) (access map[string]bool, err error, code int) {
	return this.CheckMultiplePermissionsContext(context.TODO(), token, topicId, ids, permissions...)
}

func (this *ClientImpl) CheckMultiplePermissionsContext(ctx context.Context, token string, topicId string, ids []string, permissions ...Permission) (access map[string]bool, err error, code int) {
	query := url.Values{}
	query.Set("permissions", PermissionList(permissions).Encode())
	query.Set("ids", strings.Join(ids, ","))
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/check/%v?%v", this.serverUrl, url.PathEscape(topicId), query.Encode()), nil)
	if err != nil {
		return access, err, 0
	}
	return doWithContext[map[string]bool](ctx, token, req)
}

func (this *ClientImpl) ListComputedPermissions(token string, topicId string, ids []string) (result []model.ComputedPermissions, err error, code int) {
	return this.ListComputedPermissionsContext(context.TODO(), token, topicId, ids)
}

func (this *ClientImpl) ListComputedPermissionsContext(ctx context.Context, token string, topicId string, ids []string) (result []model.ComputedPermissions, err error, code int) {
	body, err := json.Marshal(ids)
	if err != nil {
		return result, err, 0
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%v/query/permissions/%v", this.serverUrl, url.PathEscape(topicId)), bytes.NewReader(body))
	if err != nil {
		return result, err, 0
	}
	return doWithContext[[]model.ComputedPermissions](ctx, token, req)
}

func (this *ClientImpl) AdminListResourceIds(token string, topicId string, options ListOptions) (ids []string, err error, code int) {
	return this.AdminListResourceIdsContext(context.TODO(), token, topicId, options)
}

func (this *ClientImpl) AdminListResourceIdsContext(ctx context.Context, token string, topicId string, options ListOptions) (ids []string, err error, code int) {
	query := url.Values{}
	if options.Limit > 0 {
		query.Set("limit", strconv.FormatInt(options.Limit, 10))
	}
	if options.Offset > 0 {
		query.Set("offset", strconv.FormatInt(options.Offset, 10))
	}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/admin/resources/%v?%v", this.serverUrl, url.PathEscape(topicId), query.Encode()), nil)
	if err != nil {
		return ids, err, 0
	}
	return doWithContext[[]string](ctx, token, req)
}

func (this *ClientImpl) ListAccessibleResourceIds(token string, topicId string, options ListOptions, permissions ...Permission) (ids []string, err error, code int) {
	return this.ListAccessibleResourceIdsContext(context.TODO(), token, topicId, options, permissions...)
}

func (this *ClientImpl) ListAccessibleResourceIdsContext(ctx context.Context, token string, topicId string, options ListOptions, permissions ...Permission) (ids []string, err error, code int) {
	query := url.Values{}
	query.Set("permissions", PermissionList(permissions).Encode())
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
	return doWithContext[[]string](ctx, token, req)
}

func (this *ClientImpl) ListResourcesWithAdminPermission(token string, topicId string, options ListOptions) (result []Resource, err error, code int) {
	return this.ListResourcesWithAdminPermissionContext(context.TODO(), token, topicId, options)
}

func (this *ClientImpl) ListResourcesWithAdminPermissionContext(ctx context.Context, token string, topicId string, options ListOptions) (result []Resource, err error, code int) {
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
	return doWithContext[[]Resource](ctx, token, req)
}

func (this *ClientImpl) GetResource(token string, topicId string, id string) (result Resource, err error, code int) {
	return this.GetResourceContext(context.TODO(), token, topicId, id)
}

func (this *ClientImpl) GetResourceContext(ctx context.Context, token string, topicId string, id string) (result Resource, err error, code int) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/manage/%v/%v", this.serverUrl, url.PathEscape(topicId), url.PathEscape(id)), nil)
	if err != nil {
		return result, err, 0
	}
	return doWithContext[Resource](ctx, token, req)
}

// RemoveResource removes a resource
// only admins may remove resources
func (this *ClientImpl) RemoveResource(token string, topicId string, id string) (err error, code int) {
	return this.RemoveResourceContext(context.TODO(), token, topicId, id)
}

func (this *ClientImpl) RemoveResourceContext(ctx context.Context, token string, topicId string, id string) (err error, code int) {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%v/manage/%v/%v", this.serverUrl, url.PathEscape(topicId), url.PathEscape(id)), nil)
	if err != nil {
		return err, 0
	}
	return doVoidWithContext(ctx, token, req)
}

// SetPermission sets the permissions of a resource.
// resource initialization needs to be done by an admin; user tokens may update their rights but may not create the initial resource
func (this *ClientImpl) SetPermission(token string, topicId string, id string, permissions ResourcePermissions) (result ResourcePermissions, err error, code int) {
	return this.SetPermissionContext(context.TODO(), token, topicId, id, permissions)
}

func (this *ClientImpl) SetPermissionContext(ctx context.Context, token string, topicId string, id string, permissions ResourcePermissions) (result ResourcePermissions, err error, code int) {
	body, err := json.Marshal(permissions)
	if err != nil {
		return result, err, 0
	}
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%v/manage/%v/%v", this.serverUrl, url.PathEscape(topicId), url.PathEscape(id)), bytes.NewReader(body))
	if err != nil {
		return result, err, 0
	}
	return doWithContext[ResourcePermissions](ctx, token, req)
}

// AdminLoadFromPermissionSearch is not supported by the client
// because this request should never be automated
func (this *ClientImpl) AdminLoadFromPermissionSearch(req model.AdminLoadPermSearchRequest) (updateCount int, err error, code int) {
	return this.AdminLoadFromPermissionSearchContext(context.TODO(), req)
}

func (this *ClientImpl) AdminLoadFromPermissionSearchContext(_ context.Context, req model.AdminLoadPermSearchRequest) (updateCount int, err error, code int) {
	panic("no client support: this request should never be automated")
}

func doWithContext[T any](ctx context.Context, token string, req *http.Request) (result T, err error, code int) {
	err = otelx.InjectContextToRequest(ctx, req)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	return do[T](token, req)
}

func doVoidWithContext(ctx context.Context, token string, req *http.Request) (err error, code int) {
	err = otelx.InjectContextToRequest(ctx, req)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	return doVoid(token, req)
}

func do[T any](token string, req *http.Request) (result T, err error, code int) {
	req.Header.Set("Authorization", token)

	//add version query param
	query := req.URL.Query()
	query.Set("version", ClientVersion)
	req.URL.RawQuery = query.Encode()

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

	//add version query param
	query := req.URL.Query()
	query.Set("version", ClientVersion)
	req.URL.RawQuery = query.Encode()

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
