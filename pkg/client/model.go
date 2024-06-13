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

import "github.com/SENERGY-Platform/permissions-v2/pkg/model"

type ListOptions = model.ListOptions
type GetOptions = model.GetOptions
type SetPermissionOptions = model.SetPermissionOptions
type Topic = model.Topic

type Permission = model.Permission

const Read = model.Read
const Write = model.Write
const Administrate = model.Administrate
const Execute = model.Execute

type PermissionList = model.PermissionList
type GroupPermissions = model.GroupPermissions
type Resource = model.Resource
type ResourcePermissions = model.ResourcePermissions
type PermissionsMap = model.PermissionsMap
