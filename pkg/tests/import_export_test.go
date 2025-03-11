/*
 * Copyright 2025 InfAI (CC SES)
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

package tests

import (
	"context"
	"github.com/SENERGY-Platform/permissions-v2/pkg"
	"github.com/SENERGY-Platform/permissions-v2/pkg/client"
	"github.com/SENERGY-Platform/permissions-v2/pkg/configuration"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/permissions-v2/pkg/tests/docker"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

func removeTimestamps(export client.ImportExport) client.ImportExport {
	for i, t := range export.Topics {
		t.LastUpdateUnixTimestamp = 0
		export.Topics[i] = t
	}
	return export
}

func TestImportExport(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var initRepo = func() (client.Client, error) {
		config, err := configuration.Load("../../config.json")
		if err != nil {
			t.Error(err)
			return nil, err
		}

		config.Debug = true
		config.DevNotifierUrl = ""

		dockerPort, _, err := docker.MongoDB(ctx, wg)
		if err != nil {
			t.Error(err)
			return nil, err
		}
		config.MongoUrl = "mongodb://localhost:" + dockerPort

		freePort, err := docker.GetFreePort()
		if err != nil {
			t.Error(err)
			return nil, err
		}
		config.Port = strconv.Itoa(freePort)

		err = pkg.Start(ctx, config)
		if err != nil {
			t.Error(err)
			return nil, err
		}

		time.Sleep(time.Second)

		return client.New("http://localhost:" + config.Port), nil
	}

	c, err := initRepo()
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("init topics", func(t *testing.T) {
		for _, topic := range []string{"t1", "t2", "t3"} {
			_, err, _ = c.SetTopic(client.InternalAdminToken, client.Topic{
				Id: topic,
			})
			if err != nil {
				t.Error(err)
				return
			}
		}
	})

	t.Run("init resources", func(t *testing.T) {
		for _, topic := range []string{"t1", "t2", "t3"} {
			for _, resource := range []string{"r1", "r2", "r3"} {
				_, err, _ = c.SetPermission(client.InternalAdminToken, topic, resource, client.ResourcePermissions{
					UserPermissions: map[string]model.PermissionsMap{
						"u" + topic + resource: {Read: true, Write: true, Execute: true, Administrate: true},
					},
				})
				if err != nil {
					t.Error(err)
					return
				}
			}
		}
	})

	t.Run("export topics", func(t *testing.T) {
		export, err, _ := c.Export(client.InternalAdminToken, client.ImportExportOptions{
			IncludeTopicConfig: true,
			IncludePermissions: false,
			FilterTopics:       nil,
			FilterResourceId:   nil,
		})
		if err != nil {
			t.Error(err)
			return
		}

		export = removeTimestamps(export)

		expected := model.ImportExport{
			Topics: []client.Topic{
				{Id: "t1"},
				{Id: "t2"},
				{Id: "t3"},
			},
			Permissions: nil,
		}
		expected.Sort()

		if !reflect.DeepEqual(export, expected) {
			t.Errorf("\ne:%#v\na:%#v\n", expected, export)
			return
		}
	})

	t.Run("export permissions", func(t *testing.T) {
		export, err, _ := c.Export(client.InternalAdminToken, client.ImportExportOptions{
			IncludeTopicConfig: false,
			IncludePermissions: true,
			FilterTopics:       nil,
			FilterResourceId:   nil,
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := model.ImportExport{
			Permissions: []client.Resource{
				{
					Id:      "r1",
					TopicId: "t1",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut1r1": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r2",
					TopicId: "t1",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut1r2": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r3",
					TopicId: "t1",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut1r3": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r1",
					TopicId: "t2",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut2r1": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r2",
					TopicId: "t2",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut2r2": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r3",
					TopicId: "t2",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut2r3": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r1",
					TopicId: "t3",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut3r1": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r2",
					TopicId: "t3",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut3r2": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r3",
					TopicId: "t3",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut3r3": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
			},
		}
		expected.Sort()

		if !reflect.DeepEqual(export, expected) {
			t.Errorf("\ne:%#v\na:%#v\n", expected, export)
			return
		}
	})

	t.Run("export t2", func(t *testing.T) {
		export, err, _ := c.Export(client.InternalAdminToken, client.ImportExportOptions{
			IncludeTopicConfig: true,
			IncludePermissions: true,
			FilterTopics:       []string{"t2"},
			FilterResourceId:   nil,
		})
		if err != nil {
			t.Error(err)
			return
		}
		export = removeTimestamps(export)
		expected := model.ImportExport{
			Topics: []client.Topic{
				{Id: "t2"},
			},
			Permissions: []client.Resource{
				{
					Id:      "r1",
					TopicId: "t2",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut2r1": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r2",
					TopicId: "t2",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut2r2": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r3",
					TopicId: "t2",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut2r3": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
			},
		}
		expected.Sort()

		if !reflect.DeepEqual(export, expected) {
			t.Errorf("\ne:%#v\na:%#v\n", expected, export)
			return
		}
	})

	t.Run("export permissions t2", func(t *testing.T) {
		export, err, _ := c.Export(client.InternalAdminToken, client.ImportExportOptions{
			IncludeTopicConfig: false,
			IncludePermissions: true,
			FilterTopics:       []string{"t2"},
			FilterResourceId:   nil,
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := model.ImportExport{
			Permissions: []client.Resource{
				{
					Id:      "r1",
					TopicId: "t2",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut2r1": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r2",
					TopicId: "t2",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut2r2": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r3",
					TopicId: "t2",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut2r3": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
			},
		}
		expected.Sort()

		if !reflect.DeepEqual(export, expected) {
			t.Errorf("\ne:%#v\na:%#v\n", expected, export)
			return
		}
	})

	t.Run("export permissions t2 r3", func(t *testing.T) {
		export, err, _ := c.Export(client.InternalAdminToken, client.ImportExportOptions{
			IncludeTopicConfig: false,
			IncludePermissions: true,
			FilterTopics:       []string{"t2"},
			FilterResourceId:   []string{"r3"},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := model.ImportExport{
			Permissions: []client.Resource{
				{
					Id:      "r3",
					TopicId: "t2",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut2r3": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
			},
		}
		expected.Sort()

		if !reflect.DeepEqual(export, expected) {
			t.Errorf("\ne:%#v\na:%#v\n", expected, export)
			return
		}
	})

	t.Run("export t2 r3", func(t *testing.T) {
		export, err, _ := c.Export(client.InternalAdminToken, client.ImportExportOptions{
			IncludeTopicConfig: true,
			IncludePermissions: true,
			FilterTopics:       []string{"t2"},
			FilterResourceId:   []string{"r3"},
		})
		if err != nil {
			t.Error(err)
			return
		}
		export = removeTimestamps(export)
		expected := model.ImportExport{
			Topics: []client.Topic{
				{Id: "t2"},
			},
			Permissions: []client.Resource{
				{
					Id:      "r3",
					TopicId: "t2",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut2r3": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
			},
		}
		expected.Sort()

		if !reflect.DeepEqual(export, expected) {
			t.Errorf("\ne:%#v\na:%#v\n", expected, export)
			return
		}
	})

	var storedExport model.ImportExport
	t.Run("export all", func(t *testing.T) {
		export, err, _ := c.Export(client.InternalAdminToken, client.ImportExportOptions{
			IncludeTopicConfig: true,
			IncludePermissions: true,
			FilterTopics:       nil,
			FilterResourceId:   nil,
		})
		if err != nil {
			t.Error(err)
			return
		}
		export = removeTimestamps(export)
		expected := model.ImportExport{
			Topics: []client.Topic{
				{Id: "t1"},
				{Id: "t2"},
				{Id: "t3"},
			},
			Permissions: []client.Resource{
				{
					Id:      "r1",
					TopicId: "t1",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut1r1": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r2",
					TopicId: "t1",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut1r2": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r3",
					TopicId: "t1",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut1r3": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r1",
					TopicId: "t2",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut2r1": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r2",
					TopicId: "t2",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut2r2": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r3",
					TopicId: "t2",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut2r3": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r1",
					TopicId: "t3",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut3r1": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r2",
					TopicId: "t3",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut3r2": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r3",
					TopicId: "t3",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut3r3": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
			},
		}
		expected.Sort()

		if !reflect.DeepEqual(export, expected) {
			t.Errorf("\ne:%#v\na:%#v\n", expected, export)
			return
		}
		storedExport = export
	})

	c2, err := initRepo()
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("import t2 r3", func(t *testing.T) {
		err, _ = c2.Import(client.InternalAdminToken, storedExport, client.ImportExportOptions{
			IncludeTopicConfig: true,
			IncludePermissions: true,
			FilterTopics:       []string{"t2"},
			FilterResourceId:   []string{"r3"},
		})
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("check export after partial import", func(t *testing.T) {
		export, err, _ := c2.Export(client.InternalAdminToken, client.ImportExportOptions{
			IncludeTopicConfig: true,
			IncludePermissions: true,
			FilterTopics:       nil,
			FilterResourceId:   nil,
		})
		if err != nil {
			t.Error(err)
			return
		}
		export = removeTimestamps(export)
		expected := model.ImportExport{
			Topics: []client.Topic{
				{Id: "t2"},
			},
			Permissions: []client.Resource{
				{
					Id:      "r3",
					TopicId: "t2",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut2r3": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
			},
		}
		expected.Sort()
		if !reflect.DeepEqual(export, expected) {
			t.Errorf("\ne:%#v\na:%#v\n", expected, export)
			return
		}
	})

	t.Run("import", func(t *testing.T) {
		err, _ = c2.Import(client.InternalAdminToken, storedExport, client.ImportExportOptions{
			IncludeTopicConfig: true,
			IncludePermissions: true,
			FilterTopics:       nil,
			FilterResourceId:   nil,
		})
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("check import", func(t *testing.T) {
		export, err, _ := c2.Export(client.InternalAdminToken, client.ImportExportOptions{
			IncludeTopicConfig: true,
			IncludePermissions: true,
			FilterTopics:       nil,
			FilterResourceId:   nil,
		})
		if err != nil {
			t.Error(err)
			return
		}
		export = removeTimestamps(export)
		expected := model.ImportExport{
			Topics: []client.Topic{
				{Id: "t1"},
				{Id: "t2"},
				{Id: "t3"},
			},
			Permissions: []client.Resource{
				{
					Id:      "r1",
					TopicId: "t1",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut1r1": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r2",
					TopicId: "t1",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut1r2": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r3",
					TopicId: "t1",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut1r3": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r1",
					TopicId: "t2",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut2r1": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r2",
					TopicId: "t2",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut2r2": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r3",
					TopicId: "t2",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut2r3": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r1",
					TopicId: "t3",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut3r1": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r2",
					TopicId: "t3",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut3r2": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
				{
					Id:      "r3",
					TopicId: "t3",
					ResourcePermissions: model.ResourcePermissions{
						UserPermissions: map[string]model.PermissionsMap{
							"ut3r3": {Read: true, Write: true, Execute: true, Administrate: true},
						},
						GroupPermissions: map[string]model.PermissionsMap{},
						RolePermissions:  map[string]model.PermissionsMap{},
					},
				},
			},
		}
		expected.Sort()

		if !reflect.DeepEqual(export, expected) {
			t.Errorf("\ne:%#v\na:%#v\n", expected, export)
			return
		}
	})
}
