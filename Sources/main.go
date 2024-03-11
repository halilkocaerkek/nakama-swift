/*
 * Copyright © 2024 The Satori Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/template"
	"unicode"
)

const codeTemplate string = `/* Code generated by codegen/main.go. DO NOT EDIT. */

import Foundation

/// An Error generated for HTTPURLResponse that don't return a success status.
public final class ApiResponseError: Error, Decodable {
    /// The gRPC status code of the response.
	public let grpcStatusCode: Int
    
    /// The message of the response.
    public let message: String

    /// The http status code of the response.
	public var statusCode: Int?
	
    private enum CodingKeys: String, CodingKey {
        case grpcStatusCode = "code"
        case message
    }

    public init(grpcStatusCode: Int, message: String) {
        self.grpcStatusCode = grpcStatusCode
        self.message = message
    }

	public  var description: String {
		return "ApiResponseError(StatusCode=\(statusCode ?? 0), Message='\(message)', GrpcStatusCode=\(grpcStatusCode))"
	}
}

struct EmptyResponse: Codable {
    init() {}
}

{{- range $defname, $definition := .Definitions }}
{{- $classname := $defname | title }}

{{- if isRefToEnum $defname }}

/// {{ $definition.Title }}
public enum {{ $classname }}
{
    {{- range $idx, $enum := $definition.Enum }}
    /// <summary>
    /// {{ (index (splitEnumDescription $definition.Description) $idx) }}
    /// </summary>
    {{ $enum }} = {{ $idx }},
    {{- end }}
}
{{- else }}

/// {{ (descriptionOrTitle $definition.Description $definition.Title) | stripNewlines }}
public protocol {{ $classname }}Protocol: Codable {
    {{- range $propname, $property := $definition.Properties }}
    {{- $fieldname := $propname }}
    {{- if eq $fieldname "default" }}{{ $fieldname = "default_" }}{{ end }}

    /// {{ (descriptionOrTitle $property.Description $property.Title) | stripNewlines }}
    {{- if eq $property.Type "integer"}}
    var {{ $fieldname }}: Int { get }
    {{- else if eq $property.Type "number" }}
    var {{ $fieldname }}: Double { get }
    {{- else if eq $property.Type "boolean" }}
    var {{ $fieldname }}: Bool? { get }
    {{- else if eq $property.Type "string"}}
    var {{ $fieldname }}: String { get }
    {{- else if eq $property.Type "array"}}
        {{- if eq $property.Items.Type "string"}}
    var {{ $fieldname }}: [String] { get }
        {{- else if eq $property.Items.Type "integer"}}
    var {{ $fieldname }}: [Int] { get }
        {{- else if eq $property.Items.Type "number"}}
    var {{ $fieldname }}: [Double] { get }
        {{- else if eq $property.Items.Type "boolean"}}
    var {{ $fieldname }}: [Bool] { get }
        {{- else}}
    var {{ $fieldname }}: [{{ $property.Items.Ref | cleanRef }}]? { get }
        {{- end }}
    {{- else if eq $property.Type "object"}}
        {{- if eq $property.AdditionalProperties.Type "string" }}
            {{- if eq $property.AdditionalProperties.Format "int64" }}
    var {{ $fieldname }}: [String: Int]? { get }
            {{- else }}
    var {{ $fieldname }}: [String: String]? { get }
            {{- end }}
        {{- else if eq $property.AdditionalProperties.Type "integer"}}
    var {{ $fieldname }}: [String: Int]? { get }
        {{- else if eq $property.AdditionalProperties.Type "number"}}
    var ? Double]? { get }
        {{- else if eq $property.AdditionalProperties.Type "boolean"}}
    var {{ $fieldname }}: [String: Bool]? { get }
        {{- else }}
    var {{ $fieldname }}: [String: {{$property.AdditionalProperties.Ref | cleanRef}}]? { get }
        {{- end}}
    {{- else if isRefToEnum (cleanRef $property.Ref) }}
    var {{ $fieldname }}: {{ $property.Ref | cleanRef }} { get }
    {{- else }}
    var {{ $fieldname }}: {{ $property.Ref | cleanRef }} { get }
    {{- end }}
    {{- end }}
}

public class {{ $classname }}: {{ $classname }}Protocol
{
    {{- range $propname, $property := $definition.Properties }}
    {{- $fieldname := $propname }}
    {{- $attrDataName := $propname | camelToSnake }}
    {{- if eq $fieldname "default" }}{{ $fieldname = "default_" }}{{ end }}

    {{- if eq $property.Type "integer" }}
    public var {{ $fieldname }}: Int
    {{- else if eq $property.Type "number" }}
    public var {{ $fieldname }}: Double
    {{- else if eq $property.Type "boolean" }}
    public var {{ $fieldname }}: Bool?
    {{- else if eq $property.Type "string" }}
    public var {{ $fieldname }}: String
    {{- else if eq $property.Type "array" }}
        {{- if eq $property.Items.Type "string" }}
    public var {{ $fieldname }}:[String]
        {{- else if eq $property.Items.Type "integer" }}
    public var {{ $fieldname }}: [Int]
        {{- else if eq $property.Items.Type "number" }}
    public var {{ $fieldname }}: [Double]
        {{- else if eq $property.Items.Type "boolean" }}
    public var {{ $fieldname }}: [Bool]
        {{- else}}
    public var {{ $fieldname }}: [{{ $property.Items.Ref | cleanRef }}]? = []
        {{- end }}
    {{- else if eq $property.Type "object"}}
        {{- if eq $property.AdditionalProperties.Type "string"}}
            {{- if eq $property.AdditionalProperties.Format "int64" }}
    public var {{ $fieldname }}: [String: Int]? = [:]
    {{- else }}
    public var {{ $fieldname }}: [String: String]? = [:]
    {{- end }}
    {{- else if eq $property.AdditionalProperties.Type "integer"}}
    public var {{ $fieldname }}: [String: Int]? = [:]
    {{- else if eq $property.AdditionalProperties.Type "number"}}
    public var {{ $fieldname }}: [String: Double]? = [:]
    {{- else if eq $property.AdditionalProperties.Type "boolean"}}
    public var {{ $fieldname }}: [String: Bool]? = [:]
    {{- else}}
    public var {{ $fieldname }}: [String: {{$property.AdditionalProperties.Ref | cleanRef}}]? = [:]
    {{- end}}
    {{- else if isRefToEnum (cleanRef $property.Ref) }}
    public var {{ $property.Ref | cleanRef }} {{ $fieldname }}
    {{- else }}
    public var {{ $fieldname }}: {{ $property.Ref | cleanRef }}
    {{- end }}
    {{- end }}

    private enum CodingKeys: String, CodingKey {
        {{- range $fieldname, $property := $definition.Properties }}
        {{- $propname := $fieldname }}
        {{- if eq $fieldname "default" }}{{ $fieldname = "default_" }}{{ end }}
        {{- if eq $propname "refreshToken" }}{{ $propname = "refresh_token" }}{{ end }}
        case {{ $fieldname }} = "{{ $propname }}"
        {{- end }}
    }
    
    init(
        {{- $first := true -}}
        {{- range $propname, $property := $definition.Properties }}
        {{- if eq $propname "default" }}{{ $propname = "default_" }}{{ end }}
        {{- if $first }}{{- $first = false }}{{- else }}, {{- end }}
        {{- $fieldname := $propname }}
        {{- $attrDataName := $propname | camelToSnake }}
        {{- if eq $property.Type "integer" }}
        {{ $fieldname }}: Int
        {{- else if eq $property.Type "number" }}
        {{ $fieldname }}: Double
        {{- else if eq $property.Type "boolean" }}
        {{ $fieldname }}: Bool
        {{- else if eq $property.Type "string" }}
        {{ $fieldname }}: String
        {{- else if eq $property.Type "array" }}
            {{- if eq $property.Items.Type "string" }}
        {{ $fieldname }}:[String]
            {{- else if eq $property.Items.Type "integer" }}
        {{ $fieldname }}: [Int]
            {{- else if eq $property.Items.Type "number" }}
        {{ $fieldname }}: [Double]
            {{- else if eq $property.Items.Type "boolean" }}
        {{ $fieldname }}: [Bool]
            {{- else}}
        {{ $fieldname }}: [{{ $property.Items.Ref | cleanRef }}] = []
            {{- end }}
        {{- else if eq $property.Type "object"}}
            {{- if eq $property.AdditionalProperties.Type "string"}}
                {{- if eq $property.AdditionalProperties.Format "int64" }}
        {{ $fieldname }}: [String: Int] = [:]
        {{- else }}
        {{ $fieldname }}: [String: String] = [:]
        {{- end }}
        {{- else if eq $property.AdditionalProperties.Type "integer"}}
        {{ $fieldname }}: [String: Int] = [:]
        {{- else if eq $property.AdditionalProperties.Type "number"}}
        {{ $fieldname }}: [String: Double] = [:]
        {{- else if eq $property.AdditionalProperties.Type "boolean"}}
        {{ $fieldname }}: [String: Bool] = [:]
        {{- else}}
        {{ $fieldname }}: [String: {{$property.AdditionalProperties.Ref | cleanRef}}] = [:]
        {{- end}}
        {{- else if isRefToEnum (cleanRef $property.Ref) }}
        {{ $property.Ref | cleanRef }} {{ $fieldname }}
        {{- else }}
        {{ $fieldname }}: {{ $property.Ref | cleanRef }}
        {{- end }}
        {{- end }}
    ) {
        {{- range $fieldname, $property := $definition.Properties }}
        {{- if eq $fieldname "default" }}{{ $fieldname = "default_" }}{{ end }}
        self.{{ $fieldname }} = {{ $fieldname }}
        {{- end }}
    }

    var debugDescription: String {
        return "{{- range $fieldname, $property := $definition.Properties }}{{- if eq $fieldname "default" }}{{ $fieldname | snakeToCamel }}: \({{ $fieldname | snakeToCamel }}_) {{- else }}{{ $fieldname | snakeToCamel }}: \({{ $fieldname | snakeToCamel }}) {{- end }}{{- end }}"
    }
}
{{- end }}


{{- end }}

/// The low level client for the {{ .Namespace }} API.
class ApiClient
{
    public let httpAdapter: HttpAdapterProtocol
    public let timeout: Int

    private(set) var baseUri: URL

    public init(baseUri: URL, httpAdapter: HttpAdapterProtocol, timeout: Int = 10)
    {
        self.baseUri = baseUri
        self.httpAdapter = httpAdapter
        self.timeout = timeout
    }

    {{- range $url, $path := .Paths }}
    {{- range $method, $operation := $path}}

    /// {{ $operation.Summary | stripNewlines }}
    public func {{ $operation.OperationId | stripOperationPrefix | snakeToPascal }}(

    {{- $isPreviousParam := false}}

    {{- if $operation.Security }}
        {{- range $idx, $security := $operation.Security}}
            {{- range $key, $value := $security}}
                {{- if or (eq $key "BasicAuth") (eq $key "HttpKeyAuth") }}
        basicAuthUsername: String,
        basicAuthPassword: String
                    {{- $isPreviousParam = true}}
                {{- else if (eq $key "BearerJwt") }}
                    {{- $isPreviousParam = true}}
        bearerToken: String,
                {{- end }}
            {{- end }}
        {{- end }}
    {{- else }}
        {{- $isPreviousParam = true}}
        bearerToken: String
    {{- end }}

    {{- range $parameter := $operation.Parameters }}

    {{- if eq $isPreviousParam true}},{{- end}}
    {{- if eq $parameter.In "path" }}
        {{ $parameter.Name }}: {{ $parameter.Type | camelToPascal }}{{- if not $parameter.Required }}?{{- end }}
    {{- else if eq $parameter.In "body" }}
        {{- if eq $parameter.Schema.Type "string" }}
        string{{- if not $parameter.Required }}?{{- end }} {{ $parameter.Name }}
        {{- else }}
        {{ $parameter.Name }}: {{ $parameter.Schema.Ref | cleanRef }}{{- if not $parameter.Required }}?{{- end }}
        {{- end }}
    {{- else if eq $parameter.Type "array"}}
        {{ $parameter.Name | snakeToCamel }}: [{{ $parameter.Items.Type | camelToPascal }}]
    {{- else if eq $parameter.Type "object"}}
        {{- if eq $parameter.AdditionalProperties.Type "string"}}
    {{ $parameter.Name }}: [String : String]
        {{- else if eq $parameter.Items.Type "integer"}}
    {{ $parameter.Name }}: [String : Int]
        {{- else if eq $parameter.Items.Type "boolean"}}
    {{ $parameter.Name }}: [String : Int]
        {{- else}}
    {{ $parameter.Name }}: [String : {{ $parameter.Items.Type }}] 
        {{- end}}
    {{- else if eq $parameter.Type "integer" }}
        {{ $parameter.Name }}: Int?
    {{- else if eq $parameter.Type "boolean" }}
        {{ $parameter.Name }}: Bool?
    {{- else if eq $parameter.Type "string" }}
        {{ $parameter.Name }}: String?
    {{- else }}
        {{ $parameter.Type }} {{ $parameter.Name }}
    {{- end }}
    {{- $isPreviousParam = true}}
{{- end }}) async throws -> {{- if $operation.Responses.Ok.Schema.Ref }} {{ $operation.Responses.Ok.Schema.Ref | cleanRef }}{{- else }} Void {{- end }} {
        {{- range $parameter := $operation.Parameters }}
        {{- if $parameter.Required }}
        {{- end }}
    {{- end }}

        var urlComponents = URLComponents()
        urlComponents.scheme = baseUri.scheme
        urlComponents.host = baseUri.host
        urlComponents.path = "{{- $url }}"

        {{- range $parameter := $operation.Parameters }}
        {{- $camelToSnake := $parameter.Name | camelToSnake }}
        {{- if eq $parameter.In "path" }}
        urlComponents.path.append({{ $parameter.Name }}.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed)!)
        {{- end }}
    {{- end }}

        var queryItems = [URLQueryItem]()
        {{- range $parameter := $operation.Parameters }}
        {{- $camelToSnake := $parameter.Name | camelToSnake }}
        {{- if eq $parameter.In "query"}}
            {{- if eq $parameter.Type "integer" }}
        if let {{ $parameter.Name }} {
            queryItems.append(URLQueryItem(name: "{{- $camelToSnake }}", value: "\({{ $parameter.Name }})"))
        }
            {{- else if eq $parameter.Type "string" }}
        if let {{ $parameter.Name }} {
            queryItems.append(URLQueryItem(name: "{{- $camelToSnake }}", value: {{ $parameter.Name }}.lowercased()))
        }
            {{- else if eq $parameter.Type "boolean" }}
        if let {{ $parameter.Name }} {
            queryItems.append(URLQueryItem(name: "{{- $camelToSnake }}", value: "\({{ $parameter.Name }})".addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed)))
        }
            {{- else if eq $parameter.Type "array" }}
        for param in {{ $parameter.Name | snakeToCamel }} {
            {{- if eq $parameter.Items.Type "string" }}
            queryItems.append(URLQueryItem(name: "{{- $camelToSnake }}", value: param))
                {{- else }}
            queryItems.append(URLQueryItem(name: "{{- $camelToSnake }}", value: param.description))
                {{- end }}
        }
            {{- else }}
        {{ $parameter }}
            {{- end }}
        {{- end }}
    {{- end }}
        urlComponents.queryItems = queryItems
        guard let url = urlComponents.url else {
            throw SatoriError.invalidURL
        }

        let method = "{{- $method | uppercase }}"
        var headers: [String: String] = [:]

        {{- if $operation.Security }}
            {{- range $idx, $security := $operation.Security }}
                {{- range $key, $value := $security }}
                    {{- if or (eq $key "BasicAuth") (eq $key "HttpKeyAuth")}}
        if !basicAuthUsername.isEmpty {
            if let credentials = "\(basicAuthUsername):\(basicAuthPassword)".data(using: .utf8)?.base64EncodedString() {
                var header = "Basic \(credentials)"
                headers["Authorization"] = header
            }
        }
                    {{- else if (eq $key "BearerJwt") }}
        if !bearerToken.isEmpty {
            var header = "Bearer \(bearerToken)"
            headers["Authorization"] = header
        }
                    {{- end }}
                {{- end }}
            {{- end }}
        {{- else }}
        var header = "Bearer \(bearerToken)"
        headers["Authorization"] = header
        {{- end }}

        var content: Data? = nil
        {{- range $parameter := $operation.Parameters }}
        {{- if eq $parameter.In "body" }}
        let encoder = JSONEncoder()
        do {
            content = try encoder.encode({{ $parameter.Name }})
        } catch {
            print("Error encoding body: \(error)")
        }
        {{- end }}
        {{- end }}

        {{- if $operation.Responses.Ok.Schema.Ref }}
        var response: {{ $operation.Responses.Ok.Schema.Ref | cleanRef }} = try await httpAdapter.sendAsync(method: method, uri: url, headers: headers, body: content, timeoutSec: timeout)
        return response
        {{- else }}
        let _: EmptyResponse = try await httpAdapter.sendAsync(method: method, uri: url, headers: headers, body: content, timeoutSec: timeout)
        {{- end }}
    }
    {{- end }}
{{- end }}
}
`

func convertRefToClassName(input string) (className string) {
	cleanRef := strings.TrimPrefix(input, "#/definitions/")
	className = strings.Title(cleanRef)
	return
}

// camelToSnake converts a camel or Pascal case string into snake case.
func camelToSnake(input string) (output string) {
	for k, v := range input {
		if unicode.IsUpper(v) {
			formatString := "%c"

			if k != 0 {
				formatString = "_" + formatString
			}

			output += fmt.Sprintf(formatString, unicode.ToLower(v))
		} else {
			output += string(v)
		}
	}

	return
}

func snakeToCamel(input string) (snakeToCamel string) {
	isToUpper := false
	for k, v := range input {
		if k == 0 {
			snakeToCamel = strings.ToLower(string(input[0]))
		} else {
			if isToUpper {
				snakeToCamel += strings.ToUpper(string(v))
				isToUpper = false
			} else {
				if v == '_' {
					isToUpper = true
				} else {
					snakeToCamel += string(v)
				}
			}
		}

	}
	return
}

func snakeToPascal(input string) (output string) {
	isToUpper := false
	for k, v := range input {
		if k == 0 {
			output = strings.ToUpper(string(input[0]))
		} else {
			if isToUpper {
				output += strings.ToUpper(string(v))
				isToUpper = false
			} else {
				if v == '_' {
					isToUpper = true
				} else {
					output += string(v)
				}
			}
		}
	}
	return
}

func isPropertyEnum(string) (output string) {
	return
}

// pascalToCamel converts a Pascal case string to a camel case string.
func pascalToCamel(input string) (camelCase string) {
	if input == "" {
		return ""
	}

	camelCase = strings.ToLower(string(input[0]))
	camelCase += string(input[1:])
	return camelCase
}

func splitEnumDescription(description string) (output []string) {
	return strings.Split(description, "\n")
}

func stripNewlines(input string) string {
	return strings.Replace(input, "\n", " ", -1)
}

func stripOperationPrefix(input string) string {
	return strings.Replace(input, "Nakama_", "", 1)
}

func descriptionOrTitle(description string, title string) string {
	if description != "" {
		return description
	}

	return title
}

// camelToPascal converts a string from camel case to Pascal case.
func camelToPascal(camelCase string) (pascalCase string) {

	if len(camelCase) <= 0 {
		return ""
	}

	pascalCase = strings.ToUpper(string(camelCase[0])) + camelCase[1:]
	return
}

func main() {
	// Argument flags
	var output = flag.String("output", "", "The output for generated code.")
	flag.Parse()

	inputs := flag.Args()
	if len(inputs) < 1 {
		fmt.Printf("No input file found: %s\n\n", inputs)
		fmt.Println("openapi-gen [flags] inputs...")
		flag.PrintDefaults()
		return
	}

	inputFile := inputs[0]
	content, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Unable to read file: %s\n", err)
		return
	}

	var namespace (string) = ""

	if len(inputs) > 1 {
		if len(inputs[1]) <= 0 {
			fmt.Println("Empty Namespace provided.")
			return
		}

		namespace = inputs[1]
	}

	var schema *Schema
	if err := json.Unmarshal(content, &schema); err != nil {
		fmt.Printf("Unable to decode input file %s : %s\n", inputFile, err)
		return
	}
	schema.Namespace = namespace

	generateBodyDefinitionFromSchema(schema)

	fmap := template.FuncMap{
		"snakeToCamel": snakeToCamel,
		"camelToSnake": camelToSnake,
		"cleanRef":     convertRefToClassName,
		"isRefToEnum": func(ref string) bool {
			// swagger schema definition keys have inconsistent casing
			var camelOk bool
			var pascalOk bool
			var enums []string

			asCamel := pascalToCamel(ref)
			if _, camelOk = schema.Definitions[asCamel]; camelOk {
				enums = schema.Definitions[asCamel].Enum
			}

			asPascal := camelToPascal(ref)
			if _, pascalOk = schema.Definitions[asPascal]; pascalOk {
				enums = schema.Definitions[asPascal].Enum
			}

			if !pascalOk && !camelOk {
				fmt.Printf("no definition found: %v", ref)
				return false
			}

			return len(enums) > 0
		},
		"pascalToCamel":        pascalToCamel,
		"snakeToPascal":        snakeToPascal,
		"stripNewlines":        stripNewlines,
		"title":                strings.Title,
		"uppercase":            strings.ToUpper,
		"camelToPascal":        camelToPascal,
		"splitEnumDescription": splitEnumDescription,
		"stripOperationPrefix": stripOperationPrefix,
		"descriptionOrTitle":   descriptionOrTitle,
	}

	tmpl, err := template.New(inputFile).Funcs(fmap).Parse(codeTemplate)
	if err != nil {
		panic(err)
	}

	if len(*output) < 1 {
		tmpl.Execute(os.Stdout, schema)
		return
	}

	f, err := os.Create(*output)
	if err != nil {
		fmt.Printf("Unable to create file: %s\n", err)
		return
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	tmpl.Execute(writer, schema)
	writer.Flush()
}

type Schema struct {
	Namespace string
	Paths     map[string]map[string]struct {
		Summary     string
		OperationId string
		Responses   struct {
			Ok struct {
				Schema struct {
					Ref string `json:"$ref"`
				}
			} `json:"200"`
		}
		Parameters []struct {
			Name     string
			In       string
			Required bool
			Type     string   // used with primitives
			Items    struct { // used with type "array"
				Type string
			}
			Format string       // used with type "boolean"
			Schema ObjectSchema `json:"schema"`
		}
		Security []map[string][]struct {
		}
	}
	Definitions map[string]ObjectDefinition
}

type ObjectSchema struct {
	Type       string
	Ref        string `json:"$ref"`
	Properties map[string]struct {
		Type        string
		Description string
	}
	Description string
}

type ObjectDefinition struct {
	Properties map[string]ObjectProperty

	Enum        []string
	Description string
	// used only by enums
	Title string
}

type ObjectProperty struct {
	Type                 string
	Ref                  string `json:"$ref"` // used with object
	Items                Items
	AdditionalProperties AdditionalProperties
	Format               string // used with type "boolean"
	Description          string
	Title                string // used by enums
}

type Items struct {
	Type string
	Ref  string `json:"$ref"`
}

type AdditionalProperties struct {
	Type   string // used with type "map"
	Format string // used with type "map"
	Ref    string `json:"$ref"` // used with object
}

func generateBodyDefinitionFromSchema(s *Schema) {
	// Needed because of this change: https://github.com/grpc-ecosystem/grpc-gateway/issues/1670
	for _, def := range s.Paths {
		if verb, ok := def["put"]; ok {
			for idx, param := range verb.Parameters {
				if param.In == "body" && param.Name == "body" && param.Schema.Ref == "" {
					objectName := "Api" + strings.TrimPrefix(verb.OperationId, fmt.Sprintf("%s_", s.Namespace)) + "Request"
					param.Schema.Ref = "#/definitions/" + objectName
					def["put"].Parameters[idx] = param

					properties := make(map[string]ObjectProperty)
					for key, p := range param.Schema.Properties {
						properties[key] = ObjectProperty{
							Type:                 p.Type,
							Items:                Items{},
							AdditionalProperties: AdditionalProperties{},
							Description:          p.Description,
						}
					}

					s.Definitions[objectName] = ObjectDefinition{
						Properties:  properties,
						Description: param.Schema.Description,
					}
				}
			}
		}
	}
}
