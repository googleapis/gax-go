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
	"errors"
	"io"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestProtoJSONStreamRecv(t *testing.T) {
	loc := &locationpb.Location{
		Name:        "projects/example-project/locations/us-east1",
		LocationId:  "us-east1",
		DisplayName: "New York City",
	}
	locations := []proto.Message{loc, loc, loc}

	for _, tst := range []struct {
		name    string
		want    []proto.Message
		wantErr error
		typ     protoreflect.MessageType
	}{
		{
			name:    "simple_locations",
			want:    locations,
			wantErr: io.EOF,
			typ:     loc.ProtoReflect().Type(),
		},
		{
			name:    "empty",
			wantErr: io.EOF,
		},
	} {
		s, err := prepareStream(tst.want)
		if err != nil {
			t.Errorf("%s: %v", tst.name, err)
			continue
		}
		stream := NewProtoJSONStream(s, tst.typ)
		defer stream.Close()

		got, err := stream.Recv()
		for ndx := 0; err == nil; ndx++ {
			if diff := cmp.Diff(got, tst.want[ndx], cmp.Comparer(proto.Equal)); diff != "" {
				t.Errorf("%s: got(-),want(+):\n%s", tst.name, diff)
			}
			got, err = stream.Recv()
		}
		if !errors.Is(err, tst.wantErr) {
			t.Errorf("%s: expected %s but got %v", tst.name, tst.wantErr, err)
		}
	}
}

func prepareStream(messages []proto.Message) (io.ReadCloser, error) {
	if len(messages) == 0 {
		return ioutil.NopCloser(bytes.NewReader([]byte("[]"))), nil
	}

	data := []byte("[")
	for _, m := range messages {
		d, err := protojson.Marshal(m)
		if err != nil {
			return nil, err
		}
		data = append(data, d...)
		data = append(data, ',')
	}
	// Set the trailing ',' to a closing ']'.
	data[len(data)-1] = ']'
	return ioutil.NopCloser(bytes.NewReader(data)), nil
}
