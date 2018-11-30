// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gax

import "bytes"

// XGoogHeader is for use by the Google Cloud Libraries only.
//
// XGoogHeader formats key-value pairs.
// The resulting string is suitable for x-goog-api-client header.
func XGoogHeader(keyval ...string) string {
	if len(keyval) == 0 {
		return ""
	}
	if len(keyval)%2 != 0 {
		panic("gax.Header: odd argument count")
	}
	var buf bytes.Buffer
	for i := 0; i < len(keyval); i += 2 {
		buf.WriteByte(' ')
		buf.WriteString(keyval[i])
		buf.WriteByte('/')
		buf.WriteString(keyval[i+1])
	}
	return buf.String()[1:]
}
