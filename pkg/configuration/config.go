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
	"log/slog"
	"os"
	"reflect"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	struct_logger "github.com/SENERGY-Platform/go-service-base/struct-logger"
)

type Config struct {
	Port            string `json:"port"`
	Debug           bool   `json:"debug"`
	EnableSwaggerUi bool   `json:"enable_swagger_ui"`
	EditForward     string `json:"edit_forward"`
	DevNotifierUrl  string `json:"dev_notifier_url"`

	KafkaUrl string `json:"kafka_url"`

	MongoUrl                   string `json:"mongo_url"`
	MongoDatabase              string `json:"mongo_database"`
	MongoPermissionsCollection string `json:"mongo_permissions_collection"`
	MongoTopicsCollection      string `json:"mongo_topics_collection"`

	MigrateFromMongoUrl string `json:"migrate_from_mongo_url"`

	SyncCheckInterval Duration `json:"sync_check_interval"`
	SyncAgeLimit      Duration `json:"sync_age_limit"`

	UserManagementUrl string `json:"user_management_url"`

	ApiDocsProviderBaseUrl string `json:"api_docs_provider_base_url"`

	LogLevel string       `json:"log_level"`
	logger   *slog.Logger `json:"-"`
}

// loads config from json in location and used environment variables (e.g KafkaUrl --> KAFKA_URL)
func Load(location string) (config Config, err error) {
	file, err := os.Open(location)
	if err != nil {
		return config, err
	}
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return config, err
	}
	err = handleEnvironmentVars(&config, os.Getenv)
	return config, err
}

func (this *Config) GetLogger() *slog.Logger {
	if this.logger == nil {
		if this.Debug {
			this.LogLevel = "debug"
		}
		info, ok := debug.ReadBuildInfo()
		project := ""
		org := ""
		if ok {
			if parts := strings.Split(info.Main.Path, "/"); len(parts) > 2 {
				project = strings.Join(parts[2:], "/")
				org = strings.Join(parts[:2], "/")
			}
		}
		this.logger = struct_logger.New(
			struct_logger.Config{
				Handler:    struct_logger.JsonHandlerSelector,
				Level:      this.LogLevel,
				TimeFormat: time.RFC3339Nano,
				TimeUtc:    true,
				AddMeta:    true,
			},
			os.Stdout,
			org,
			project,
		)
	}
	return this.logger
}

var camel = regexp.MustCompile("(^[^A-Z]*|[A-Z]*)([A-Z][^A-Z]+|$)")

func fieldNameToEnvName(s string) string {
	var a []string
	for _, sub := range camel.FindAllStringSubmatch(s, -1) {
		if sub[1] != "" {
			a = append(a, sub[1])
		}
		if sub[2] != "" {
			a = append(a, sub[2])
		}
	}
	return strings.ToUpper(strings.Join(a, "_"))
}

// preparations for docker
func handleEnvironmentVars(config *Config, getEnv func(key string) string) (err error) {
	configValue := reflect.Indirect(reflect.ValueOf(config))
	configType := configValue.Type()
	for index := 0; index < configType.NumField(); index++ {
		fieldName := configType.Field(index).Name
		fieldConfig := configType.Field(index).Tag.Get("config")
		envName := fieldNameToEnvName(fieldName)
		envValue := getEnv(envName)
		if envValue != "" {
			loggedEnvValue := envValue
			if strings.Contains(fieldConfig, "secret") {
				loggedEnvValue = "***"
			}
			fmt.Println("use environment variable: ", envName, " = ", loggedEnvValue)
			if field := configValue.FieldByName(fieldName); field.Kind() == reflect.Struct && field.CanInterface() {
				fieldPtrInterface := field.Addr().Interface()
				setter, setterOk := fieldPtrInterface.(interface{ SetString(string) })
				errSetter, errSetterOk := fieldPtrInterface.(interface{ SetString(string) error })
				if setterOk {
					setter.SetString(envValue)
				}
				if errSetterOk {
					err = errSetter.SetString(envValue)
					if err != nil {
						return fmt.Errorf("invalid env variable %v=%v: %w", envName, envValue, err)
					}
				}
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Int64 || configValue.FieldByName(fieldName).Kind() == reflect.Int {
				i, err := strconv.ParseInt(envValue, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid env variable %v=%v: %w", envName, envValue, err)
				}
				configValue.FieldByName(fieldName).SetInt(i)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.String {
				configValue.FieldByName(fieldName).SetString(envValue)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Bool {
				b, err := strconv.ParseBool(envValue)
				if err != nil {
					return fmt.Errorf("invalid env variable %v=%v: %w", envName, envValue, err)
				}
				configValue.FieldByName(fieldName).SetBool(b)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Float64 {
				f, err := strconv.ParseFloat(envValue, 64)
				if err != nil {
					return fmt.Errorf("invalid env variable %v=%v: %w", envName, envValue, err)
				}
				configValue.FieldByName(fieldName).SetFloat(f)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Slice {
				val := []string{}
				for _, element := range strings.Split(envValue, ",") {
					val = append(val, strings.TrimSpace(element))
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(val))
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Map {
				value := map[string]string{}
				for _, element := range strings.Split(envValue, ",") {
					keyVal := strings.Split(element, ":")
					key := strings.TrimSpace(keyVal[0])
					val := strings.TrimSpace(keyVal[1])
					value[key] = val
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(value))
			}
		}
	}
	return nil
}

type Duration struct {
	dur time.Duration
}

func (this *Duration) GetDuration() time.Duration {
	return this.dur
}

func (this *Duration) SetDuration(dur time.Duration) {
	this.dur = dur
}

func (this *Duration) SetString(str string) error {
	if str == "" {
		return nil
	}
	duration, err := time.ParseDuration(str)
	if err != nil {
		return err
	}
	this.SetDuration(duration)
	return nil
}

func (this *Duration) UnmarshalJSON(bytes []byte) (err error) {
	var str string
	err = json.Unmarshal(bytes, &str)
	if err != nil {
		return err
	}
	return this.SetString(str)
}
