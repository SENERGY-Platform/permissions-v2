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
	"testing"
	"time"
)

func TestLoadDuration(t *testing.T) {
	config := Config{}
	err := json.Unmarshal([]byte(`{"check_db_topic_changes_interval": "10m"}`), &config)
	if err != nil {
		t.Error(err)
		return
	}
	if config.CheckDbTopicChangesInterval.GetDuration() != time.Minute*10 {
		t.Error(config.CheckDbTopicChangesInterval.GetDuration())
		return
	}
}

func TestHandleEnvironmentVars(t *testing.T) {
	config := Config{}

	testEnv := func(key string) string {
		if key == fieldNameToEnvName("CheckDbTopicChangesInterval") {
			return "10s"
		}
		return ""
	}

	testEnvErr := func(key string) string {
		if key == fieldNameToEnvName("CheckDbTopicChangesInterval") {
			return "foo"
		}
		return ""
	}

	err := handleEnvironmentVars(&config, testEnv)
	if err != nil {
		t.Error(err)
		return
	}
	if config.CheckDbTopicChangesInterval.GetDuration() != 10*time.Second {
		t.Error(config.CheckDbTopicChangesInterval.GetDuration())
		return
	}

	err = handleEnvironmentVars(&config, testEnvErr)
	if err == nil {
		t.Error(err)
		return
	}
}
