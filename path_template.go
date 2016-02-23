// Copyright 2016, Google Inc.
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package gax

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	customVerbRegexp = regexp.MustCompile(":([^:/*}{=]+)$")
)

type matcher interface {
	match([]string) (int, error)
	String() string
}

type segment struct {
	matcher
	name string
}

type labelMatcher string

func (ls labelMatcher) match(segments []string) (int, error) {
	if len(segments) == 0 {
		return 0, fmt.Errorf("expected %s but no more segments found", ls)
	}
	if segments[0] != string(ls) {
		return 0, fmt.Errorf("expected %s but got %s", ls, segments[0])
	}
	return 1, nil
}

func (ls labelMatcher) String() string {
	return string(ls)
}

type wildcardMatcher int

func (wm wildcardMatcher) match(segments []string) (int, error) {
	if len(segments) == 0 {
		return 0, errors.New("no more segments found")
	}
	return 1, nil
}

func (wm wildcardMatcher) String() string {
	return "*"
}

type pathWildcardMatcher int

func (pwm pathWildcardMatcher) match(segments []string) (int, error) {
	length := len(segments) - int(pwm)
	if length <= 0 {
		return 0, errors.New("not sufficient segments are supplied for path wildcard")
	}
	return length, nil
}

func (pwm pathWildcardMatcher) String() string {
	return "**"
}

type ParseError struct {
	Pos     int
	Message string
}

func (pe ParseError) Error() string {
	return fmt.Sprintf("at %d, %s", pe.Pos, pe.Message)
}

func parseSegments(template string) ([]segment, error) {
	if len(template) == 0 {
		return nil, ParseError{0, "input is empty"}
	}
	var pathWildcardFound bool
	var segments []segment
	paths := strings.Split(template, "/")
	unnamedVariableCount := 0
	nameSet := map[string]struct{}{}
	charPos := 0
	var currentVarName string
	for i, path := range paths {
		// Empty path with i == 0 should be allowed for the templates starting with '/'.
		if path == "" && i != 0 {
			return nil, ParseError{charPos, "empty path component"}
		}
		var matcher matcher
		name := currentVarName
		if strings.HasPrefix(path, "{") {
			equalPos := strings.Index(path, "=")
			if equalPos > 0 {
				name = path[1:equalPos]
				path = path[equalPos+1:]
				if currentVarName != "" {
					return nil, ParseError{charPos, "recursive named bindings are not allowed"}
				}
				currentVarName = name
			} else {
				if path[len(path)-1] != '}' {
					return nil, ParseError{charPos, "'}' is expected"}
				}
				if currentVarName != "" {
					return nil, ParseError{charPos, "recursive named bindings are not allowed"}
				}
				name = path[1 : len(path)-1]
				path = "*"
			}
			if _, ok := nameSet[name]; ok {
				return nil, ParseError{charPos, fmt.Sprintf("%s appears multiple times", name)}
			}
			nameSet[name] = struct{}{}
		}
		if strings.HasPrefix(path, "}") {
			return nil, ParseError{charPos, "} is not allowed here"}
		}
		if strings.HasSuffix(path, "}") {
			path = path[:len(path)-1]
			currentVarName = ""
		}
		if path == "*" {
			if name == "" {
				name = fmt.Sprintf("$%d", unnamedVariableCount)
				unnamedVariableCount++
			}
			matcher = wildcardMatcher(0)
		} else if path == "**" {
			if pathWildcardFound {
				return nil, ParseError{charPos, "multiple ** isn't allowed"}
			}
			pathWildcardFound = true
			if name == "" {
				name = fmt.Sprintf("$%d", unnamedVariableCount)
				unnamedVariableCount++
			}
			matcher = pathWildcardMatcher(len(paths) - i - 1)
		} else {
			matcher = labelMatcher(path)
		}
		segments = append(segments, segment{matcher, name})
		charPos += len(path) + 1
	}
	return segments, nil
}

type PathTemplate struct {
	segments   []segment
	customVerb string
}

func getCustomVerb(path string) (main string, customVerb string) {
	matched := customVerbRegexp.FindStringSubmatchIndex(path)
	if len(matched) == 0 {
		return path, ""
	}
	return path[:matched[0]], path[matched[2]:]
}

func NewPathTemplate(template string) (*PathTemplate, error) {
	template, customVerb := getCustomVerb(template)
	segments, err := parseSegments(template)
	if err != nil {
		return nil, err
	}
	return &PathTemplate{segments: segments, customVerb: customVerb}, nil
}

func (pt *PathTemplate) Match(path string) (map[string]string, error) {
	path, customVerb := getCustomVerb(path)
	if pt.customVerb != customVerb {
		return nil, errors.New("custom verb doesn't match")
	}
	paths := strings.Split(path, "/")
	values := map[string]string{}
	for _, segment := range pt.segments {
		length, err := segment.match(paths)
		if err != nil {
			return nil, err
		}
		if segment.name != "" {
			value := strings.Join(paths[:length], "/")
			if oldValue, ok := values[segment.name]; ok {
				values[segment.name] = oldValue + "/" + value
			} else {
				values[segment.name] = value
			}
		}
		paths = paths[length:]
	}
	if len(paths) != 0 {
		return nil, fmt.Errorf("Trailing path %s remains after the matching", strings.Join(paths, "/"))
	}
	return values, nil
}

func (pt *PathTemplate) Instantiate(binding map[string]string) (string, error) {
	result := make([]string, 0, len(pt.segments))
	var lastVariableName string
	for _, segment := range pt.segments {
		name := segment.name
		if lastVariableName != "" && name == lastVariableName {
			continue
		}
		lastVariableName = name
		if name == "" {
			result = append(result, segment.String())
		} else if value, ok := binding[name]; ok {
			result = append(result, value)
		} else {
			return "", fmt.Errorf("%s is not found", name)
		}
	}
	built := strings.Join(result, "/")
	if pt.customVerb != "" {
		built += ":" + pt.customVerb
	}
	return built, nil
}
