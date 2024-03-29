// GENERATED BY THE COMMAND ABOVE; DO NOT EDIT
// This file was generated by swaggo/swag

package docs

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/alecthomas/template"
	"github.com/swaggo/swag"
)

var doc = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{.Description}}",
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
                "description": "List all current task IDs",
                "tags": [
                    "Exec"
                ],
                "summary": "List",
                "parameters": [
                    {
                        "type": "string",
                        "description": "sort by [start,expired,completed]",
                        "name": "sort",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/handlers.JSONResult"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/handlers.JSONResult"
                        }
                    }
                }
            },
            "post": {
                "description": "Execute command or script content",
                "tags": [
                    "Exec"
                ],
                "summary": "Exec",
                "parameters": [
                    {
                        "description": "scripts",
                        "name": "scripts",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/handlers.PostStruct"
                            }
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/handlers.JSONResult"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/handlers.JSONResult"
                        }
                    }
                }
            }
        },
        "/{id}": {
            "get": {
                "description": "Get the execution result",
                "tags": [
                    "Exec"
                ],
                "summary": "Get",
                "parameters": [
                    {
                        "type": "string",
                        "description": "id",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/handlers.JSONResult"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/handlers.JSONResult"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "handlers.JSONResult": {
            "type": "object",
            "properties": {
                "code": {
                    "type": "integer"
                },
                "data": {
                    "type": "object"
                },
                "message": {
                    "type": "string"
                }
            }
        },
        "handlers.PostStruct": {
            "type": "object",
            "required": [
                "command_content",
                "command_type"
            ],
            "properties": {
                "command_content": {
                    "type": "string",
                    "example": "ping baidu.com -c 10"
                },
                "command_type": {
                    "type": "string",
                    "example": "sh"
                },
                "depends_on": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "envs": {
                    "type": "object",
                    "additionalProperties": {
                        "type": "string"
                    },
                    "example": {
                        "env1": "value1",
                        "env2": "value2"
                    }
                },
                "name": {
                    "type": "string"
                },
                "timeout": {
                    "type": "string",
                    "example": "5m"
                }
            }
        }
    }
}`

type swaggerInfo struct {
	Version     string
	Host        string
	BasePath    string
	Schemes     []string
	Title       string
	Description string
}

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = swaggerInfo{
	Version:     "v1.0.0",
	Host:        "",
	BasePath:    "",
	Schemes:     []string{},
	Title:       "OSRemoteExecution API",
	Description: "This is a os remote executor orchestration script interface.",
}

type s struct{}

func (s *s) ReadDoc() string {
	sInfo := SwaggerInfo
	sInfo.Description = strings.Replace(sInfo.Description, "\n", "\\n", -1)

	t, err := template.New("swagger_info").Funcs(template.FuncMap{
		"marshal": func(v interface{}) string {
			a, _ := json.Marshal(v)
			return string(a)
		},
	}).Parse(doc)
	if err != nil {
		return doc
	}

	var tpl bytes.Buffer
	if err := t.Execute(&tpl, sInfo); err != nil {
		return doc
	}

	return tpl.String()
}

func init() {
	swag.Register(swag.Name, &s{})
}
