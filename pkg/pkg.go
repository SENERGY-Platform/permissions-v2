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

package pkg

import (
	"context"
	"github.com/SENERGY-Platform/permissions-v2/pkg/api"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/controller"
	"github.com/SENERGY-Platform/permissions-v2/pkg/database"
	"sync"
)

func Start(ctx context.Context, wg *sync.WaitGroup, config configuration.Config) error {
	db, err := database.New(config)
	if err != nil {
		return err
	}
	ctrl, err := controller.NewWithDependencies(ctx, config, db, config.EditForward != "" && config.EditForward != "-")
	if err != nil {
		return err
	}
	err = api.Start(ctx, config, ctrl)
	if err != nil {
		return err
	}
	return nil
}
