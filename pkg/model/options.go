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
	"net/url"
	"strconv"
)

type ListOptions struct {
	Limit  int64 // 0 -> unlimited
	Offset int64
}

func ListOptionsFromQuery(q url.Values) (result ListOptions, err error) {
	limit := q.Get("limit")
	if limit != "" {
		result.Limit, err = strconv.ParseInt(limit, 10, 64)
		if err != nil {
			return result, err
		}
	}
	offset := q.Get("offset")
	if offset != "" {
		result.Offset, err = strconv.ParseInt(offset, 10, 64)
		if err != nil {
			return result, err
		}
	}
	return result, err
}

type GetOptions struct {
	CheckPermission bool
	UserId          string
	GroupIds        []string
	Permission      string
}

type SetPermissionOptions struct {
	Wait bool
}
