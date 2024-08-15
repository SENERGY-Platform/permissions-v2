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
type Topic = model.Topic

type Permission = model.Permission

const Read = model.Read                 //user may read the resource (metadata)  (e.g. read device name)
const Write = model.Write               //user may write the resource (metadata)(e.g. rename device)
const Administrate = model.Administrate // user may delete resource; user may change resource rights (e.g. delete device)
const Execute = model.Execute           //user may use the resource (e.g. cmd to device; read device data; read database)

type PermissionList = model.PermissionList
type GroupPermissions = model.GroupPermissions
type Resource = model.Resource
type ResourcePermissions = model.ResourcePermissions
type PermissionsMap = model.PermissionsMap

const ClientVersion = model.ClientVersion
