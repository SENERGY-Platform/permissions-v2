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

package model

import (
	"fmt"
)

type Permission rune

const Read Permission = 'r'         //user may read the resource (metadata)  (e.g. read device name)
const Write Permission = 'w'        //user may write the resource (metadata)(e.g. rename device)
const Administrate Permission = 'a' // user may delete resource; user may change resource rights (e.g. delete device)
const Execute Permission = 'x'      //user may use the resource (e.g. cmd to device; read device data; read database)

type PermissionList []Permission

func PermissionListFromString(str string) (result PermissionList, err error) {
	for _, p := range str {
		switch Permission(p) {
		case Read, Write, Execute, Administrate:
			result = append(result, Permission(p))
		default:
			return result, fmt.Errorf("unknown permission '%v'", p)
		}
	}
	return result, nil
}

func (this PermissionList) Encode() string {
	return string(this)
}

type GroupPermissions struct {
	GroupName string `json:"group_name"`
	PermissionsMap
}

type Resource struct {
	Id      string `json:"id"`
	TopicId string `json:"topic_id"`
	ResourcePermissions
}

type ResourcePermissions struct {
	UserPermissions  map[string]PermissionsMap `json:"user_permissions"`
	GroupPermissions map[string]PermissionsMap `json:"group_permissions"`
	RolePermissions  map[string]PermissionsMap `json:"role_permissions"`
}

func (this ResourcePermissions) Valid() bool {
	//needs at least one admin user
	for _, r := range this.UserPermissions {
		if r.Administrate {
			return true
		}
	}
	return false
}

type PermissionsMap struct {
	Read         bool `json:"read"`
	Write        bool `json:"write"`
	Execute      bool `json:"execute"`
	Administrate bool `json:"administrate"`
}

type ComputedPermissions struct {
	Id string `json:"id"`
	PermissionsMap
}
