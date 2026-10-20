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

package configuration

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLoadDuration(t *testing.T) {
	config := Config{}
	err := json.Unmarshal([]byte(`{"sync_check_interval": "10m"}`), &config)
	if err != nil {
		t.Error(err)
		return
	}
	if config.SyncCheckInterval.GetDuration() != time.Minute*10 {
		t.Error(config.SyncCheckInterval.GetDuration())
		return
	}
}

func TestHandleEnvironmentVars(t *testing.T) {
	config := Config{}

	testEnv := func(key string) string {
		if key == fieldNameToEnvName("SyncCheckInterval") {
			return "10s"
		}
		return ""
	}

	testEnvErr := func(key string) string {
		if key == fieldNameToEnvName("SyncCheckInterval") {
			return "foo"
		}
		return ""
	}

	err := handleEnvironmentVars(&config, testEnv)
	if err != nil {
		t.Error(err)
		return
	}
	if config.SyncCheckInterval.GetDuration() != 10*time.Second {
		t.Error(config.SyncCheckInterval.GetDuration())
		return
	}

	err = handleEnvironmentVars(&config, testEnvErr)
	if err == nil {
		t.Error(err)
		return
	}
}

type mongoFields struct {
	Url, User, Password, AuthSource, Database, PermissionsCollection, TopicsCollection string
}

func mongoOf(c Config) mongoFields {
	return mongoFields{c.MongoUrl, c.MongoUser, c.MongoPassword, c.MongoAuthSource, c.MongoDatabase, c.MongoPermissionsCollection, c.MongoTopicsCollection}
}

// clearMongoEnv empties the variables for this test; the loader ignores empty values.
func clearMongoEnv(t *testing.T) {
	for _, k := range []string{"MONGO_URL", "MONGO_USER", "MONGO_PASSWORD", "MONGO_AUTH_SOURCE", "MONGO_DATABASE", "MONGO_PERMISSIONS_COLLECTION", "MONGO_TOPICS_COLLECTION", "MIGRATE_FROM_MONGO_URL"} {
		t.Setenv(k, "")
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad_RepoConfigMongoDefaults(t *testing.T) {
	clearMongoEnv(t)
	cfg, err := Load("../../config.json")
	if err != nil {
		t.Fatal(err)
	}
	want := mongoFields{Url: "mongodb://localhost:27017", AuthSource: "admin", Database: "permissions", PermissionsCollection: "permissions", TopicsCollection: "topics"}
	if got := mongoOf(cfg); got != want {
		t.Errorf("mongo = %+v, want %+v", got, want)
	}
}

func TestLoad_MongoDefaultsWhenFileOmitsThem(t *testing.T) {
	clearMongoEnv(t)
	cfg, err := Load(writeConfig(t, `{}`))
	if err != nil {
		t.Fatal(err)
	}
	want := mongoFields{Url: "mongodb://localhost:27017", AuthSource: "admin", Database: "permissions"}
	if got := mongoOf(cfg); got != want {
		t.Errorf("mongo = %+v, want %+v", got, want)
	}
}

func TestLoad_MongoEnvNames(t *testing.T) {
	clearMongoEnv(t)
	t.Setenv("MONGO_URL", "mongodb://mongo-0.mongo:27017,mongo-1.mongo:27017/?replicaSet=rs0")
	t.Setenv("MONGO_USER", "permissions")
	t.Setenv("MONGO_PASSWORD", "p@ss:w/rd")
	t.Setenv("MONGO_AUTH_SOURCE", "users")
	t.Setenv("MONGO_DATABASE", "permissions_test")
	var cfg Config
	captureStdout(t, func() {
		var err error
		if cfg, err = Load("../../config.json"); err != nil {
			t.Error(err)
		}
	})
	want := mongoFields{
		Url:                   "mongodb://mongo-0.mongo:27017,mongo-1.mongo:27017/?replicaSet=rs0",
		User:                  "permissions",
		Password:              "p@ss:w/rd",
		AuthSource:            "users",
		Database:              "permissions_test",
		PermissionsCollection: "permissions",
		TopicsCollection:      "topics",
	}
	if got := mongoOf(cfg); got != want {
		t.Errorf("mongo = %+v, want %+v", got, want)
	}
}

func TestLoad_MongoConfigFile(t *testing.T) {
	clearMongoEnv(t)
	cfg, err := Load(writeConfig(t, `{"mongo_url": "mongodb://file:27017", "mongo_user": "u", "mongo_password": "s3cr3t", "mongo_auth_source": "a", "mongo_database": "d", "mongo_permissions_collection": "p", "mongo_topics_collection": "t"}`))
	if err != nil {
		t.Fatal(err)
	}
	want := mongoFields{Url: "mongodb://file:27017", User: "u", Password: "s3cr3t", AuthSource: "a", Database: "d", PermissionsCollection: "p", TopicsCollection: "t"}
	if got := mongoOf(cfg); got != want {
		t.Errorf("mongo = %+v, want %+v", got, want)
	}
}

// The loader prints every environment variable it applies.
func TestLoad_EnvPrintMasksMongoPassword(t *testing.T) {
	clearMongoEnv(t)
	t.Setenv("MONGO_USER", "permissions")
	t.Setenv("MONGO_PASSWORD", "s3cr3t-pw")
	out := captureStdout(t, func() {
		if _, err := Load("../../config.json"); err != nil {
			t.Error(err)
		}
	})
	if strings.Contains(out, "s3cr3t-pw") {
		t.Errorf("printed environment leaks the password: %s", out)
	}
	if !strings.Contains(out, "MONGO_PASSWORD  =  ***") {
		t.Errorf("expected the password variable to be printed masked, got: %s", out)
	}
	if !strings.Contains(out, "MONGO_USER  =  permissions") {
		t.Errorf("expected the applied variables to be printed, got: %s", out)
	}
}

func TestConfigFormattingMasksMongoPassword(t *testing.T) {
	clearMongoEnv(t)
	t.Setenv("MONGO_PASSWORD", "s3cr3t-pw")
	var cfg Config
	captureStdout(t, func() {
		var err error
		if cfg, err = Load("../../config.json"); err != nil {
			t.Error(err)
		}
	})
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	bPtr, err := json.Marshal(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	outputs := map[string]string{
		"json":         string(b),
		"json pointer": string(bPtr),
		"%v":           fmt.Sprintf("%v", cfg),
		"%+v":          fmt.Sprintf("%+v", cfg),
		"%#v":          fmt.Sprintf("%#v", cfg),
		"%s":           fmt.Sprintf("%s", cfg),
		"%v pointer":   fmt.Sprintf("%v", &cfg),
		"%+v pointer":  fmt.Sprintf("%+v", &cfg),
		"String":       cfg.String(),
		"GoString":     cfg.GoString(),
		"Sprint":       fmt.Sprint(cfg),
	}
	for name, s := range outputs {
		if strings.Contains(s, "s3cr3t-pw") {
			t.Errorf("%s leaks the password: %s", name, s)
		}
		if !strings.Contains(s, "***") || !strings.Contains(s, "mongodb://localhost:27017") {
			t.Errorf("%s does not show the masked config: %s", name, s)
		}
	}
	if !strings.Contains(string(b), `"mongo_password":"***"`) {
		t.Errorf("json does not show the password as masked: %s", b)
	}
	if cfg.MongoPassword != "s3cr3t-pw" {
		t.Errorf("masking changed the loaded password to %q", cfg.MongoPassword)
	}
}

func TestConfigFormattingKeepsEmptyPasswordEmpty(t *testing.T) {
	b, err := json.Marshal(Config{MongoUrl: "mongodb://localhost:27017"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"mongo_password":""`) {
		t.Errorf("an empty password should stay empty: %s", b)
	}
}

func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	defer func() { os.Stdout = orig }()
	f()
	_ = w.Close()
	return <-done
}

func TestGroupGrantersSetString(t *testing.T) {
	cases := []struct {
		in   string
		want map[string][]string
	}{
		{"/shared-resources:user-1", map[string][]string{"/shared-resources": {"user-1"}}},
		{" /shared-resources : user-1 , /shared-resources:user-2,/shared-resources:user-1,/other:user-3", map[string][]string{"/shared-resources": {"user-1", "user-2"}, "/other": {"user-3"}}},
		{"/admins-only:", map[string][]string{"/admins-only": {}}},
		{"/a:b:user-1", map[string][]string{"/a:b": {"user-1"}}},
	}
	for _, c := range cases {
		g := GroupGranters{}
		if err := g.SetString(c.in); err != nil {
			t.Errorf("%q: %v", c.in, err)
			continue
		}
		if !reflect.DeepEqual(g.Granters, c.want) {
			t.Errorf("%q: got %#v, want %#v", c.in, g.Granters, c.want)
		}
	}
	for _, in := range []string{"", "/shared-resources", ":user-1", " :user-1", "/shared-resources:user-1,", "/shared-resources:user-1,,/other:user-2"} {
		g := GroupGranters{Granters: map[string][]string{"/keep": {"user-1"}}}
		if err := g.SetString(in); err == nil {
			t.Errorf("%q: expected error, got %#v", in, g.Granters)
		}
		if !reflect.DeepEqual(g.Granters, map[string][]string{"/keep": {"user-1"}}) {
			t.Errorf("%q: a rejected value changed the granters to %#v", in, g.Granters)
		}
	}
}

func TestGroupGrantersJson(t *testing.T) {
	cfg := Config{}
	err := json.Unmarshal([]byte(`{"restricted_group_granters": {"/shared-resources": ["user-1", " user-2 ", "user-1"], "/admins-only": null, "/empty": []}}`), &cfg)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{"/shared-resources": {"user-1", "user-2"}, "/admins-only": {}, "/empty": {}}
	if !reflect.DeepEqual(cfg.RestrictedGroupGranters.Granters, want) {
		t.Errorf("got %#v, want %#v", cfg.RestrictedGroupGranters.Granters, want)
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"restricted_group_granters":{"/admins-only":[],"/empty":[],"/shared-resources":["user-1","user-2"]}`) {
		t.Errorf("config json does not show the granters as map: %s", b)
	}
	roundTrip := GroupGranters{}
	b, err = json.Marshal(cfg.RestrictedGroupGranters)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(roundTrip.Granters, want) {
		t.Errorf("round trip through %s gave %#v", b, roundTrip.Granters)
	}

	for _, in := range []string{`{"": ["user-1"]}`, `{"/g": [""]}`, `{"/g": [" "]}`, `{"/g": "user-1"}`, `["/g"]`} {
		g := GroupGranters{}
		if err := json.Unmarshal([]byte(in), &g); err == nil {
			t.Errorf("%s: expected error, got %#v", in, g.Granters)
		}
	}
}

func TestLoad_RestrictedGroupGranters(t *testing.T) {
	clearMongoEnv(t)
	t.Setenv("RESTRICTED_GROUP_GRANTERS", "")
	cfg, err := Load("../../config.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.RestrictedGroupGranters.Granters) != 0 {
		t.Errorf("repo config must leave the restriction off, got %#v", cfg.RestrictedGroupGranters.Granters)
	}

	if name := fieldNameToEnvName("RestrictedGroupGranters"); name != "RESTRICTED_GROUP_GRANTERS" {
		t.Errorf("env name = %v", name)
	}
	t.Setenv("RESTRICTED_GROUP_GRANTERS", "/shared-resources:user-1,/shared-resources:user-2")
	captureStdout(t, func() {
		cfg, err = Load(writeConfig(t, `{"restricted_group_granters": {"/from-file": ["user-3"]}}`))
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{"/shared-resources": {"user-1", "user-2"}}
	if !reflect.DeepEqual(cfg.RestrictedGroupGranters.Granters, want) {
		t.Errorf("env should replace the file value: got %#v, want %#v", cfg.RestrictedGroupGranters.Granters, want)
	}

	t.Setenv("RESTRICTED_GROUP_GRANTERS", "/shared-resources")
	captureStdout(t, func() {
		_, err = Load("../../config.json")
	})
	if err == nil {
		t.Error("expected an invalid env value to fail loading")
	}
}
