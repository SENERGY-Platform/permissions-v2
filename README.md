## Client

### Init

#### how to create a client
```
actualClient := client.New(permissionV2Url)
testClient, err := client.NewTestClient(ctx)
```

#### to manage permissions of a resource-kind/topic, it must first be registered: 
```
_, err, _ = actualClient.SetTopic(client.InternalAdminToken, Topic{
    Id: "testtopic",
})
```

this method should be called once on every startup.
the `InternalAdminToken` is a convenient token with admin rights, that can only be used internally. `InternalAdminToken` will be rejected by kong.

the `Topic` has the following properties:
```
type Topic struct {
	Id string `json:"id"`

	PublishToKafkaTopic string `json:"publish_to_kafka_topic"`

	EnsureKafkaTopicInit                bool `json:"ensure_kafka_topic_init"`
	EnsureKafkaTopicInitPartitionNumber int  `json:"ensure_kafka_topic_init_partition_number"`

	LastUpdateUnixTimestamp int64 `json:"last_update_unix_timestamp"` //should be ignored by the user; is set by db

	DefaultPermissions ResourcePermissions `json:"default_permissions"`
}

type ResourcePermissions struct {
	UserPermissions  map[string]PermissionsMap `json:"user_permissions"`
	GroupPermissions map[string]PermissionsMap `json:"group_permissions"`
	RolePermissions  map[string]PermissionsMap `json:"role_permissions"`
}

type PermissionsMap struct {
	Read         bool `json:"read"`
	Write        bool `json:"write"`
	Execute      bool `json:"execute"`
	Administrate bool `json:"administrate"`
}
```

- Id: mandatory, z.b. "devices"
- DefaultPermissions: what permissions does every resource of its kind/topic get
- PublishToKafkaTopic: optional, if != "" -> topic where cqrs commands are additionally published to
- EnsureKafkaTopicInit: optinal, should the PublishToKafkaTopic be initialized
- EnsureKafkaTopicInitPartitionNumber: how many partitions should a PublishToKafkaTopic get when EnsureKafkaTopicInit == true

### Usage

the most commonly used client methods:
- SetTopic
- SetPermission
- CheckPermission
- CheckMultiplePermissions
- ListAccessibleResourceIds
- AdminListResourceIds
- RemoveResource

### embed client in local api

if the resource service is supposed to receive permissions requests, the following command embeds the permissions-v2 api router
into the local router. in this case all requests that have the `/permissions/` prefix in the path will be forwarded to the matching client method. 
```
routerWithEmbededPermissionsHandling := client.EmbedPermissionsClientIntoRouter(client.New(config.PermissionsV2Url), router, "/permissions/")
```

to reflect this new endpoints in a swagger documentation the following generator can be used (example: lib/api/doc_gen/gen.go in github.com/SENERGY-Platform/device-repository )
```go
package main

import "github.com/SENERGY-Platform/permissions-v2/pkg/client"

//go:generate go run gen.go
//go:generate go install github.com/swaggo/swag/cmd/swag@latest
//go:generate swag init -o ../../../docs --parseDependency -d .. -g api.go

// generates lib/api/permissions.go
// which enables 'swag init --parseDependency -d ./lib/api -g api.go' to generate documentation for permissions endpoints
// which are added by 'permForward := client.New(config.PermissionsV2Url).EmbedPermissionsRequestForwarding("/permissions/", router)'
func main() {
	err := client.GenerateGoFileWithSwaggoCommentsForEmbededPermissionsClient("api", "permissions", "../generated_permissions.go")
	if err != nil {
		panic(err)
	}
}

```
this generator:
1. creates with `GenerateGoFileWithSwaggoCommentsForEmbededPermissionsClient()` a `generated_permissions.go` file with swaggo/swag comments
2. installs swaggo/swag
3. generates swagger docs

## OpenAPI
uses https://github.com/swaggo/swag

### generating
```
go generate ./...
```

### swagger ui
if the config variable UseSwaggerEndpoints is set to true, a swagger ui is accessible on /swagger/index.html (http://localhost:8080/swagger/index.html)