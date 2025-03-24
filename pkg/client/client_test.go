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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
)

func TestClientInterface(t *testing.T) {
	//make sure, that the clients implement the client interface
	var c Client
	var err error
	c = New("") //*ClientImpl implements Client
	if c == nil {
		t.Error("client is nil")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, err = NewTestClient(ctx) //*controller.Controller implements Client
	if err != nil {
		t.Error(err)
		return
	}
	if c == nil {
		t.Error("client is nil")
		return
	}
}

func TestEmbed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, err := NewTestClient(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	requests := []string{}
	mux := sync.Mutex{}

	router := http.NewServeMux()

	router.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
		mux.Lock()
		defer mux.Unlock()
		requests = append(requests, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	})

	router.HandleFunc("GET /manage/{topic}", func(w http.ResponseWriter, r *http.Request) {
		mux.Lock()
		defer mux.Unlock()
		requests = append(requests, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	})

	routerWithEmbededClient := EmbedPermissionsClientIntoRouter(c, router, "/permissions/", nil)

	server := httptest.NewServer(routerWithEmbededClient)
	defer server.Close()

	httpClient := New(server.URL + "/permissions")

	_, err, _ = httpClient.SetTopic(InternalAdminToken, Topic{
		Id: "testtopic",
	})
	if err != nil {
		t.Error(err)
		return
	}
	topics, err, _ := httpClient.ListTopics(InternalAdminToken, ListOptions{})
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(topics, []Topic{{Id: "testtopic"}}) {
		t.Error(topics)
		return
	}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/permissions/admin/topics", nil)
	if err != nil {
		t.Error(err)
		return
	}
	req.Header.Set("Authorization", InternalAdminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Error(resp.StatusCode)
		return
	}
	err = json.NewDecoder(resp.Body).Decode(&topics)
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(topics, []Topic{{Id: "testtopic"}}) {
		t.Error(topics)
		return
	}

	resp, err = http.Get(server.URL + "/test")
	if err != nil {
		t.Error(err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		t.Error(resp.StatusCode)
		return
	}

	mux.Lock()
	defer mux.Unlock()
	if !reflect.DeepEqual(requests, []string{"/test"}) {
		t.Error(requests)
		return
	}
}
