// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "license": {
            "name": "Apache 2.0",
            "url": "http://www.apache.org/licenses/LICENSE-2.0.html"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/": {
            "get": {
                "description": "checks health and reachability of the service",
                "tags": [
                    "health"
                ],
                "summary": "health check",
                "responses": {
                    "200": {
                        "description": "OK"
                    }
                }
            }
        },
        "/accessible/{topic}": {
            "get": {
                "description": "list accessible resource ids",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "accessible",
                    "resource"
                ],
                "summary": "list accessible resource ids",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Topic Id",
                        "name": "topic",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "checked rights in the form of 'rwxa', defaults to 'r'",
                        "name": "rights",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "limits size of result; 0 means unlimited",
                        "name": "limit",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "offset to be used in combination with limit",
                        "name": "offset",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "type": "string"
                            }
                        }
                    },
                    "400": {
                        "description": "Bad Request"
                    },
                    "401": {
                        "description": "Unauthorized"
                    },
                    "500": {
                        "description": "Internal Server Error"
                    }
                }
            }
        },
        "/admin/topics": {
            "get": {
                "description": "lists topics with their configuration, requesting user must be admin",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "topics"
                ],
                "summary": "lists topics with their configuration",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "limits size of result; 0 means unlimited",
                        "name": "limit",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "offset to be used in combination with limit",
                        "name": "offset",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/model.Topic"
                            }
                        }
                    },
                    "400": {
                        "description": "Bad Request"
                    },
                    "401": {
                        "description": "Unauthorized"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "500": {
                        "description": "Internal Server Error"
                    }
                }
            },
            "post": {
                "description": "set topic config, requesting user must be admin",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "topics"
                ],
                "summary": "set topic config",
                "parameters": [
                    {
                        "description": "Topic",
                        "name": "message",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/model.Topic"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/model.Topic"
                        }
                    },
                    "400": {
                        "description": "Bad Request"
                    },
                    "401": {
                        "description": "Unauthorized"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "404": {
                        "description": "Not Found"
                    },
                    "500": {
                        "description": "Internal Server Error"
                    }
                }
            }
        },
        "/admin/topics/{id}": {
            "get": {
                "description": "get topic config, requesting user must be admin",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "topics"
                ],
                "summary": "get topic config",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Topic Id",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/model.Topic"
                        }
                    },
                    "400": {
                        "description": "Bad Request"
                    },
                    "401": {
                        "description": "Unauthorized"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "404": {
                        "description": "Not Found"
                    },
                    "500": {
                        "description": "Internal Server Error"
                    }
                }
            },
            "put": {
                "description": "set topic config, requesting user must be admin",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "topics"
                ],
                "summary": "set topic config",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Topic Id",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "Topic",
                        "name": "message",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/model.Topic"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/model.Topic"
                        }
                    },
                    "400": {
                        "description": "Bad Request"
                    },
                    "401": {
                        "description": "Unauthorized"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "404": {
                        "description": "Not Found"
                    },
                    "500": {
                        "description": "Internal Server Error"
                    }
                }
            },
            "delete": {
                "description": "remove topic config, requesting user must be admin",
                "tags": [
                    "topics"
                ],
                "summary": "remove topic config",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Topic Id",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK"
                    },
                    "400": {
                        "description": "Bad Request"
                    },
                    "401": {
                        "description": "Unauthorized"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "404": {
                        "description": "Not Found"
                    },
                    "500": {
                        "description": "Internal Server Error"
                    }
                }
            }
        },
        "/check/{topic}": {
            "get": {
                "description": "check multiple permissions",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "check"
                ],
                "summary": "check multiple permissions",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Topic Id",
                        "name": "topic",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "Resource Ids, comma seperated",
                        "name": "ids",
                        "in": "query",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "checked rights in the form of 'rwxa', defaults to 'r'",
                        "name": "rights",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "boolean"
                            }
                        }
                    },
                    "400": {
                        "description": "Bad Request"
                    },
                    "401": {
                        "description": "Unauthorized"
                    },
                    "500": {
                        "description": "Internal Server Error"
                    }
                }
            }
        },
        "/check/{topic}/{id}": {
            "get": {
                "description": "check permission",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "check"
                ],
                "summary": "check permission",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Topic Id",
                        "name": "topic",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "Resource Id",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "checked rights in the form of 'rwxa', defaults to 'r'",
                        "name": "rights",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "boolean"
                        }
                    },
                    "400": {
                        "description": "Bad Request"
                    },
                    "401": {
                        "description": "Unauthorized"
                    },
                    "500": {
                        "description": "Internal Server Error"
                    }
                }
            }
        },
        "/health": {
            "get": {
                "description": "checks health and reachability of the service",
                "tags": [
                    "health"
                ],
                "summary": "health check",
                "responses": {
                    "200": {
                        "description": "OK"
                    }
                }
            }
        },
        "/manage/{topic}": {
            "get": {
                "description": "lists resources the user has admin rights to",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "manage",
                    "resource"
                ],
                "summary": "lists resources the user has admin rights to",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Topic Id",
                        "name": "topic",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "integer",
                        "description": "limits size of result; 0 means unlimited",
                        "name": "limit",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "offset to be used in combination with limit",
                        "name": "offset",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/model.Resource"
                            }
                        }
                    },
                    "400": {
                        "description": "Bad Request"
                    },
                    "401": {
                        "description": "Unauthorized"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "500": {
                        "description": "Internal Server Error"
                    }
                }
            }
        },
        "/manage/{topic}/{id}": {
            "get": {
                "description": "get resource, requesting user must have admin right",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "manage",
                    "resource"
                ],
                "summary": "get resource",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Topic Id",
                        "name": "topic",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "Resource Id",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/model.Resource"
                        }
                    },
                    "400": {
                        "description": "Bad Request"
                    },
                    "401": {
                        "description": "Unauthorized"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "500": {
                        "description": "Internal Server Error"
                    }
                }
            },
            "put": {
                "description": "get resource rights, requesting user must have admin right",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "manage",
                    "resource-rights"
                ],
                "summary": "set resource rights",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Topic Id",
                        "name": "topic",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "Resource Id",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "Topic",
                        "name": "message",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/model.ResourceRights"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/model.ResourceRights"
                        }
                    },
                    "400": {
                        "description": "Bad Request"
                    },
                    "401": {
                        "description": "Unauthorized"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "500": {
                        "description": "Internal Server Error"
                    }
                }
            }
        }
    },
    "definitions": {
        "model.GroupRight": {
            "type": "object",
            "properties": {
                "administrate": {
                    "type": "boolean"
                },
                "execute": {
                    "type": "boolean"
                },
                "group_name": {
                    "type": "string"
                },
                "read": {
                    "type": "boolean"
                },
                "write": {
                    "type": "boolean"
                }
            }
        },
        "model.Resource": {
            "type": "object",
            "properties": {
                "group_rights": {
                    "type": "object",
                    "additionalProperties": {
                        "$ref": "#/definitions/model.Right"
                    }
                },
                "id": {
                    "type": "string"
                },
                "topic_id": {
                    "type": "string"
                },
                "user_rights": {
                    "type": "object",
                    "additionalProperties": {
                        "$ref": "#/definitions/model.Right"
                    }
                }
            }
        },
        "model.ResourceRights": {
            "type": "object",
            "properties": {
                "group_rights": {
                    "type": "object",
                    "additionalProperties": {
                        "$ref": "#/definitions/model.Right"
                    }
                },
                "user_rights": {
                    "type": "object",
                    "additionalProperties": {
                        "$ref": "#/definitions/model.Right"
                    }
                }
            }
        },
        "model.Right": {
            "type": "object",
            "properties": {
                "administrate": {
                    "type": "boolean"
                },
                "execute": {
                    "type": "boolean"
                },
                "read": {
                    "type": "boolean"
                },
                "write": {
                    "type": "boolean"
                }
            }
        },
        "model.Topic": {
            "type": "object",
            "properties": {
                "id": {
                    "description": "at least one of Id and KafkaTopic must be set",
                    "type": "string"
                },
                "initial_group_rights": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/model.GroupRight"
                    }
                },
                "kafka_consumer_group": {
                    "description": "defaults to configured kafka consumer group",
                    "type": "string"
                },
                "kafka_topic": {
                    "description": "changeable, defaults to Id",
                    "type": "string"
                }
            }
        }
    },
    "securityDefinitions": {
        "Bearer": {
            "type": "apiKey",
            "name": "Authorization",
            "in": "header"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "0.1",
	Host:             "",
	BasePath:         "/",
	Schemes:          []string{},
	Title:            "Permissions API",
	Description:      "",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
