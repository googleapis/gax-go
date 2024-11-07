// Copyright 2024, Google Inc.
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

// Package logtest is a helper for validating logging tests.
//
// To update conformance tests in this package run `go test -update_golden`
package logtest

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var updateGolden = flag.Bool("update-golden", false, "update golden files")

// DiffTest is a test helper, testing got against contents of a goldenFile.
func DiffTest(t *testing.T, tempFile, goldenFile string) {
	rawGot, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Helper()
	if *updateGolden {
		got := removeLogVariance(t, rawGot)
		if err := os.WriteFile(filepath.Join("testdata", goldenFile), got, os.ModePerm); err != nil {
			t.Fatal(err)
		}
		return
	}

	want, err := os.ReadFile(filepath.Join("testdata", goldenFile))
	if err != nil {
		t.Fatal(err)
	}
	got := removeLogVariance(t, rawGot)

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch(-want, +got): %s", diff)
	}
}

// removeLogVariance removes parts of log lines that may differ between runs
// and/or machines.
func removeLogVariance(t *testing.T, in []byte) []byte {
	if len(in) == 0 {
		return in
	}
	bs := bytes.Split(in, []byte("\n"))
	for i, b := range bs {
		if len(b) == 0 {
			continue
		}
		m := map[string]any{}
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		delete(m, "timestamp")
		if sl, ok := m["sourceLocation"].(map[string]any); ok {
			delete(sl, "file")
			// So that if test cases move around in this file they don't cause
			// failures
			delete(sl, "line")
		}
		b2, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s", b2)
		bs[i] = b2
	}
	return bytes.Join(bs, []byte("\n"))
}
