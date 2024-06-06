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

package api

import (
	"encoding/json"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
	"log"
	"net/http"
)

func init() {
	endpoints = append(endpoints, &TopicsEndpoints{})
}

type TopicsEndpoints struct{}

// ListTopics godoc
// @Summary      lists topics with their configuration
// @Description  lists topics with their configuration, requesting user must be admin
// @Tags         topics
// @Param        limit query integer false "limits size of result; 0 means unlimited"
// @Param        offset query integer false "offset to be used in combination with limit"
// @Produce      json
// @Success      200 {array}  model.Topic
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      500
// @Router       /admin/topics [get]
func (this *TopicsEndpoints) ListTopics(config configuration.Config, router *http.ServeMux, ctrl Controller) {
	router.HandleFunc("GET /admin/topics", func(w http.ResponseWriter, req *http.Request) {
		token, err := jwt.GetParsedToken(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !token.IsAdmin() {
			http.Error(w, "only admins may manage topics", http.StatusForbidden)
			return
		}
		listOptions, err := model.ListOptionsFromQuery(req.URL.Query())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result, err, code := ctrl.ListTopics(token, listOptions)
		if err != nil {
			http.Error(w, err.Error(), code)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(result)
		if err != nil {
			log.Println("ERROR: unable to encode response", err)
		}
	})
}

// GetTopic godoc
// @Summary      get topic config
// @Description  get topic config, requesting user must be admin
// @Tags         topics
// @Param        id path string true "Topic Id"
// @Produce      json
// @Success      200 {object}  model.Topic
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /admin/topics/{id} [get]
func (this *TopicsEndpoints) GetTopic(config configuration.Config, router *http.ServeMux, ctrl Controller) {
	router.HandleFunc("GET /admin/topics/{id}", func(w http.ResponseWriter, req *http.Request) {
		token, err := jwt.GetParsedToken(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !token.IsAdmin() {
			http.Error(w, "only admins may manage topics", http.StatusForbidden)
			return
		}
		id := req.PathValue("id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		result, err, code := ctrl.GetTopic(token, id)
		if err != nil {
			http.Error(w, err.Error(), code)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(result)
		if err != nil {
			log.Println("ERROR: unable to encode response", err)
		}
	})
}

// SetTopic godoc
// @Summary      set topic config
// @Description  set topic config, requesting user must be admin
// @Tags         topics
// @Accept       json
// @Produce      json
// @Param        id path string true "Topic Id"
// @Param        message body model.Topic true "Topic"
// @Success      200 {object}  model.Topic
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /admin/topics/{id} [put]
func (this *TopicsEndpoints) SetTopic(config configuration.Config, router *http.ServeMux, ctrl Controller) {
	router.HandleFunc("PUT /admin/topics/{id}", func(w http.ResponseWriter, req *http.Request) {
		token, err := jwt.GetParsedToken(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !token.IsAdmin() {
			http.Error(w, "only admins may manage topics", http.StatusForbidden)
			return
		}
		id := req.PathValue("id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		topic := model.Topic{}
		err = json.NewDecoder(req.Body).Decode(&topic)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if topic.Id == "" {
			topic.Id = id
		}
		if topic.Id != id {
			http.Error(w, "topic id mismatch", http.StatusBadRequest)
			return
		}

		result, err, code := ctrl.SetTopic(token, topic)
		if err != nil {
			http.Error(w, err.Error(), code)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(result)
		if err != nil {
			log.Println("ERROR: unable to encode response", err)
		}
	})
}

// SetTopicByPost godoc
// @Summary      set topic config
// @Description  set topic config, requesting user must be admin
// @Tags         topics
// @Accept       json
// @Produce      json
// @Param        message body model.Topic true "Topic"
// @Success      200 {object}  model.Topic
// @Success      202 {object}  model.Topic
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /admin/topics [POST]
func (this *TopicsEndpoints) SetTopicByPost(config configuration.Config, router *http.ServeMux, ctrl Controller) {
	router.HandleFunc("POST /admin/topics", func(w http.ResponseWriter, req *http.Request) {
		token, err := jwt.GetParsedToken(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !token.IsAdmin() {
			http.Error(w, "only admins may manage topics", http.StatusForbidden)
			return
		}

		topic := model.Topic{}
		err = json.NewDecoder(req.Body).Decode(&topic)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		result, err, code := ctrl.SetTopic(token, topic)
		if err != nil {
			http.Error(w, err.Error(), code)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(result)
		if err != nil {
			log.Println("ERROR: unable to encode response", err)
		}
	})
}

// DeleteTopic godoc
// @Summary      remove topic config
// @Description  remove topic config, requesting user must be admin
// @Tags         topics
// @Param        id path string true "Topic Id"
// @Success      200
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /admin/topics/{id} [delete]
func (this *TopicsEndpoints) DeleteTopic(config configuration.Config, router *http.ServeMux, ctrl Controller) {
	router.HandleFunc("DELETE /admin/topics/{id}", func(w http.ResponseWriter, req *http.Request) {
		token, err := jwt.GetParsedToken(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !token.IsAdmin() {
			http.Error(w, "only admins may manage topics", http.StatusForbidden)
			return
		}
		id := req.PathValue("id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		err, code := ctrl.RemoveTopic(token, id)
		if err != nil {
			http.Error(w, err.Error(), code)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
