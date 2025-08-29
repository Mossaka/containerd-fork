/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package deprecation

import (
	"strings"
	"testing"
)

func TestWarningConstants(t *testing.T) {
	testCases := []struct {
		warning  Warning
		expected string
	}{
		{CRIRegistryMirrors, Prefix + "cri-registry-mirrors"},
		{CRIRegistryAuths, Prefix + "cri-registry-auths"},
		{CRIRegistryConfigs, Prefix + "cri-registry-configs"},
		{CRICNIBinDir, Prefix + "cri-cni-bin-dir"},
		{TracingOTLPConfig, Prefix + "tracing-processor-config"},
		{TracingServiceConfig, Prefix + "tracing-service-config"},
		{NRIV010Plugin, Prefix + "nri-v010-plugin"},
	}

	for _, tc := range testCases {
		if string(tc.warning) != tc.expected {
			t.Errorf("Expected %s, got %s", tc.expected, string(tc.warning))
		}
	}
}

func TestPrefix(t *testing.T) {
	expected := "io.containerd.deprecation/"
	if Prefix != expected {
		t.Errorf("Expected prefix %s, got %s", expected, Prefix)
	}
}

func TestEnvPrefix(t *testing.T) {
	expected := "CONTAINERD_ENABLE_DEPRECATED_"
	if EnvPrefix != expected {
		t.Errorf("Expected env prefix %s, got %s", expected, EnvPrefix)
	}
}

func TestValid(t *testing.T) {
	testCases := []struct {
		warning Warning
		valid   bool
	}{
		{CRIRegistryMirrors, true},
		{CRIRegistryAuths, true},
		{CRIRegistryConfigs, true},
		{CRICNIBinDir, true},
		{TracingOTLPConfig, true},
		{TracingServiceConfig, true},
		{NRIV010Plugin, true},
		{Warning("nonexistent"), false},
		{Warning(""), false},
		{Warning("io.containerd.deprecation/fake"), false},
	}

	for _, tc := range testCases {
		result := Valid(tc.warning)
		if result != tc.valid {
			t.Errorf("Valid(%s): expected %v, got %v", tc.warning, tc.valid, result)
		}
	}
}

func TestMessage(t *testing.T) {
	testCases := []struct {
		warning Warning
		exists  bool
	}{
		{CRIRegistryMirrors, true},
		{CRIRegistryAuths, true},
		{CRIRegistryConfigs, true},
		{CRICNIBinDir, true},
		{TracingOTLPConfig, true},
		{TracingServiceConfig, true},
		{NRIV010Plugin, true},
		{Warning("nonexistent"), false},
	}

	for _, tc := range testCases {
		msg, ok := Message(tc.warning)
		if ok != tc.exists {
			t.Errorf("Message(%s) exists: expected %v, got %v", tc.warning, tc.exists, ok)
		}
		if tc.exists && msg == "" {
			t.Errorf("Message(%s) returned empty string when should exist", tc.warning)
		}
		if !tc.exists && msg != "" {
			t.Errorf("Message(%s) returned non-empty string when should not exist: %s", tc.warning, msg)
		}
	}
}

func TestMessageContent(t *testing.T) {
	// Test that messages contain expected keywords to verify they're meaningful
	testCases := []struct {
		warning  Warning
		keywords []string
	}{
		{CRIRegistryMirrors, []string{"mirrors", "deprecated", "containerd", "config_path"}},
		{CRIRegistryAuths, []string{"auths", "deprecated", "containerd", "ImagePullSecrets"}},
		{CRIRegistryConfigs, []string{"configs", "deprecated", "containerd", "config_path"}},
		{CRICNIBinDir, []string{"bin_dir", "deprecated", "containerd", "bin_dirs"}},
		{TracingOTLPConfig, []string{"otlp", "deprecated", "containerd", "OTLP", "environment"}},
		{TracingServiceConfig, []string{"tracing", "deprecated", "containerd", "OTEL", "environment"}},
		{NRIV010Plugin, []string{"NRI", "deprecated", "containerd", "v010-adapter"}},
	}

	for _, tc := range testCases {
		msg, ok := Message(tc.warning)
		if !ok {
			t.Errorf("Message(%s) should exist but doesn't", tc.warning)
			continue
		}

		for _, keyword := range tc.keywords {
			if !strings.Contains(msg, keyword) {
				t.Errorf("Message(%s) should contain keyword '%s', but message is: %s", tc.warning, keyword, msg)
			}
		}
	}
}

func TestAllDefinedWarningsHaveMessages(t *testing.T) {
	// Ensure every defined warning constant has a corresponding message
	definedWarnings := []Warning{
		CRIRegistryMirrors,
		CRIRegistryAuths,
		CRIRegistryConfigs,
		CRICNIBinDir,
		TracingOTLPConfig,
		TracingServiceConfig,
		NRIV010Plugin,
	}

	for _, warning := range definedWarnings {
		if !Valid(warning) {
			t.Errorf("Warning %s is defined but has no message", warning)
		}
		
		msg, ok := Message(warning)
		if !ok || msg == "" {
			t.Errorf("Warning %s should have a non-empty message", warning)
		}
	}
}

func TestMessagesMapIntegrity(t *testing.T) {
	// Verify the messages map has expected number of entries
	expectedCount := 7 // Update if more warnings are added
	actualCount := len(messages)
	
	if actualCount != expectedCount {
		t.Errorf("Expected %d messages, got %d", expectedCount, actualCount)
	}
	
	// Verify all messages in the map are non-empty
	for warning, msg := range messages {
		if msg == "" {
			t.Errorf("Message for warning %s is empty", warning)
		}
		if !strings.Contains(string(warning), Prefix) {
			t.Errorf("Warning %s doesn't contain expected prefix %s", warning, Prefix)
		}
	}
}

func TestWarningStringConversion(t *testing.T) {
	warning := CRIRegistryMirrors
	str := string(warning)
	expected := "io.containerd.deprecation/cri-registry-mirrors"
	
	if str != expected {
		t.Errorf("String conversion failed: expected %s, got %s", expected, str)
	}
}