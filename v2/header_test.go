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

import "testing"

func TestXGoogHeader(t *testing.T) {
	for _, tst := range []struct {
		kv   []string
		want string
	}{
		{nil, ""},
		{[]string{"abc", "def"}, "abc/def"},
		{[]string{"abc", "def", "xyz", "123", "foo", ""}, "abc/def xyz/123 foo/"},
	} {
		got := XGoogHeader(tst.kv...)
		if got != tst.want {
			t.Errorf("Header(%q) = %q, want %q", tst.kv, got, tst.want)
		}
	}
}
