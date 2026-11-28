/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package env

import "strings"

// ExpandEnvWithDefault expands template variables with optional default for {}
func ExpandEnvWithDefault(template string, envs map[string]string, defaultValue ...string) string {
	if template == "" {
		return ""
	}

	result := template

	// Handle special case of {} - use provided default or first available file variable
	if strings.Contains(result, "{}") {
		defaultVal := ""
		if len(defaultValue) > 0 && defaultValue[0] != "" {
			defaultVal = defaultValue[0]
		}
		result = strings.ReplaceAll(result, "{}", defaultVal)
	}

	// Replace named variables
	for key, value := range envs {
		if key != "" { // Skip empty key used for {} default
			result = strings.ReplaceAll(result, "{"+key+"}", value)
		}
	}
	return result
}

// ExpandEnvSlice expands template variables in a slice of strings
func ExpandEnvSlice(templates []string, envs map[string]string) []string {
	return ExpandEnvSliceWithDefault(templates, envs)
}

// ExpandEnvSliceWithDefault expands template variables in a slice with optional default for {}
func ExpandEnvSliceWithDefault(templates []string, envs map[string]string, defaultValue ...string) []string {
	if len(templates) == 0 {
		return templates
	}

	result := make([]string, len(templates))
	for i, template := range templates {
		result[i] = ExpandEnvWithDefault(template, envs, defaultValue...)
	}
	return result
}
