// Copyright 2022, Google Inc.
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
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestProtoJSONStreamRecv(t *testing.T) {
	loc := &locationpb.Location{
		Name:        "projects/example-project/locations/us-east1",
		LocationId:  "us-east1",
		DisplayName: "New York City",
	}
	first, err := protojson.Marshal(loc)
	if err != nil {
		t.Fatal(err)
	}
	second, err := protojson.Marshal(loc)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("[" + string(first) + ",\n" + string(second) + "]")
	r := ioutil.NopCloser(bytes.NewReader(data))
	stream := NewProtoJSONStream(context.Background(), r, loc.ProtoReflect().Type())

	m, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}

	got, ok := m.(*locationpb.Location)
	if !ok {
		t.Fatalf("Expected Location, got %s", m.ProtoReflect().Type().Descriptor().Name())
	}
	if diff := cmp.Diff(got, loc, cmp.Comparer(proto.Equal)); diff != "" {
		t.Errorf("got(-),want(+):\n%s", diff)
	}

	m, err = stream.Recv()
	if err != nil {
		t.Fatal(err)
	}

	got, ok = m.(*locationpb.Location)
	if !ok {
		t.Fatalf("Expected Location, got %s", m.ProtoReflect().Type().Descriptor().Name())
	}
	if diff := cmp.Diff(got, loc, cmp.Comparer(proto.Equal)); diff != "" {
		t.Errorf("got(-),want(+):\n%s", diff)
	}

	if _, err := stream.Recv(); err != io.EOF {
		t.Errorf("expected io.EOF but got %v", err)
	}
}
