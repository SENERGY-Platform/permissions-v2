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
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller/idmodifier"
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
	"net/http"
)

func (this *Controller) CheckPermission(token jwt.Token, topicId string, id string, permissions string) (access bool, err error, code int) {
	pureId, _ := idmodifier.SplitModifier(id)
	access, err = this.db.CheckResourcePermissions(this.getTimeoutContext(), topicId, pureId, token.GetUserId(), token.GetRoles(), permissions)
	if err != nil {
		code = http.StatusInternalServerError
	} else {
		code = http.StatusOK
	}
	return access, err, code
}

func (this *Controller) CheckMultiplePermissions(token jwt.Token, topicId string, ids []string, permissions string) (access map[string]bool, err error, code int) {
	pureIdList := []string{}
	pureIdToIds := map[string][]string{}
	for _, id := range ids {
		pureId, _ := idmodifier.SplitModifier(id)
		pureIdList = append(pureIdList, pureId)
		pureIdToIds[pureId] = append(pureIdToIds[pureId], id)
	}
	var pureAccess map[string]bool
	pureAccess, err = this.db.CheckMultipleResourcePermissions(this.getTimeoutContext(), topicId, pureIdList, token.GetUserId(), token.GetRoles(), permissions)
	if err != nil {
		code = http.StatusInternalServerError
	} else {
		code = http.StatusOK
	}
	access = map[string]bool{}
	for key, value := range pureAccess {
		for _, id := range pureIdToIds[key] {
			access[id] = value
		}
	}
	return access, err, code
}
