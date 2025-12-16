// Copyright 2025, Google Inc.
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
	"os"
	"sync"
	"testing"
)

func TestIsFeatureEnabled(t *testing.T) {
	tests := []struct {
		name          string
		envVar        string
		envValue      string
		expected      bool
		expectedCache bool
	}{
		{
			name:          "EnabledFeature",
			envVar:        "GOOGLE_SDK_GO_EXPERIMENTAL_TRACING",
			envValue:      "true",
			expected:      true,
			expectedCache: true,
		},
		{
			name:          "DisabledFeature",
			envVar:        "GOOGLE_SDK_GO_EXPERIMENTAL_ANOTHER",
			envValue:      "false",
			expected:      false,
			expectedCache: false,
		},
		{
			name:          "MissingFeature",
			envVar:        "GOOGLE_SDK_GO_EXPERIMENTAL_MISSING",
			envValue:      "",
			expected:      false,
			expectedCache: false,
		},
		{
			name:          "CaseInsensitiveTrue",
			envVar:        "GOOGLE_SDK_GO_EXPERIMENTAL_MIXED_CASE",
			envValue:      "True",
			expected:      true,
			expectedCache: true,
		},
		{
			name:          "CaseInsensitiveTrue",
			envVar:        "GOOGLE_SDK_GO_EXPERIMENTAL_UPPER_CASE",
			envValue:      "TRUE",
			expected:      true,
			expectedCache: true,
		},
		{
			name:          "OtherValue",
			envVar:        "GOOGLE_SDK_GO_EXPERIMENTAL_INVALID",
			envValue:      "1",
			expected:      false,
			expectedCache: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the global state for each test to ensure isolation
			featureEnabledStore = nil
			featureEnabledOnce = sync.Once{}

			if tt.envValue != "" {
				os.Setenv(tt.envVar, tt.envValue)
				defer os.Unsetenv(tt.envVar)
			}

			if got := IsFeatureEnabled(tt.envVar[len("GOOGLE_SDK_GO_EXPERIMENTAL_"):]); got != tt.expected {
				t.Errorf("IsFeatureEnabled() = %v, want %v", got, tt.expected)
			}

			// Verify caching behavior after the first call
			if tt.expectedCache && featureEnabledStore[tt.envVar[len("GOOGLE_SDK_GO_EXPERIMENTAL_"):]] != true {
				t.Errorf("Feature %s not correctly cached as true", tt.envVar)
			} else if !tt.expectedCache && featureEnabledStore[tt.envVar[len("GOOGLE_SDK_GO_EXPERIMENTAL_"):]] == true {
				t.Errorf("Feature %s incorrectly cached as true", tt.envVar)
			}
		})
	}

	// Test that subsequent calls to IsFeatureEnabled do not re-read environment variables
	t.Run("CachingPreventsReread", func(t *testing.T) {
		// Clear previous state
		featureEnabledStore = nil
		featureEnabledOnce = sync.Once{}

		// Set an environment variable for the first call
		os.Setenv("GOOGLE_SDK_GO_EXPERIMENTAL_CACHED_FEATURE", "true")
		defer os.Unsetenv("GOOGLE_SDK_GO_EXPERIMENTAL_CACHED_FEATURE")

		// First call, should read from env and cache
		if !IsFeatureEnabled("CACHED_FEATURE") {
			t.Fatalf("Expected CACHED_FEATURE to be enabled on first call")
		}

		// Unset the environment variable after the first call
		os.Unsetenv("GOOGLE_SDK_GO_EXPERIMENTAL_CACHED_FEATURE")

		// Second call, should use cached value and still be true
		if !IsFeatureEnabled("CACHED_FEATURE") {
			t.Errorf("Expected CACHED_FEATURE to remain enabled due to caching")
		}
		// Check a new feature that was never set, should be false
		if IsFeatureEnabled("NEW_FEATURE_AFTER_CACHE") {
			t.Errorf("Expected NEW_FEATURE_AFTER_CACHE to be false as it was set after init")
		}
	})

	// Test with multiple environment variables set
	t.Run("MultipleEnvVars", func(t *testing.T) {
		// Clear previous state
		featureEnabledStore = nil
		featureEnabledOnce = sync.Once{}

		os.Setenv("GOOGLE_SDK_GO_EXPERIMENTAL_FEATURE1", "true")
		os.Setenv("GOOGLE_SDK_GO_EXPERIMENTAL_FEATURE2", "false")
		os.Setenv("GOOGLE_SDK_GO_EXPERIMENTAL_FEATURE3", "true")
		defer os.Unsetenv("GOOGLE_SDK_GO_EXPERIMENTAL_FEATURE1")
		defer os.Unsetenv("GOOGLE_SDK_GO_EXPERIMENTAL_FEATURE2")
		defer os.Unsetenv("GOOGLE_SDK_GO_EXPERIMENTAL_FEATURE3")

		if !IsFeatureEnabled("FEATURE1") {
			t.Errorf("Expected FEATURE1 to be enabled")
		}
		if IsFeatureEnabled("FEATURE2") {
			t.Errorf("Expected FEATURE2 to be disabled")
		}
		if !IsFeatureEnabled("FEATURE3") {
			t.Errorf("Expected FEATURE3 to be enabled")
		}
		if IsFeatureEnabled("NONEXISTENT_FEATURE") {
			t.Errorf("Expected NONEXISTENT_FEATURE to be disabled")
		}
	})
}
