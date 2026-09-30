// Copyright 2026 Google LLC
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

package upload

// startState is the default initial state, corresponding to an uploader being configured but no
// setup RPCs for an upload being sent.
type startState struct{}

// transmitState corresponds to an upload that has been configured and started, and able to send data.
type transmitState struct{}

// terminalState is the final terminal state.  It doesn't implement
// any dispatched methods, as the upload has concluded.
type terminalState struct{}
