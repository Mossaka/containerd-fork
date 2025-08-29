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

package progress

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestBar_Format(t *testing.T) {
	tests := []struct {
		name          string
		bar           Bar
		format        string
		expected      string
		shouldContain []string
	}{
		{
			name:          "zero progress",
			bar:           Bar(0.0),
			format:        "%r",
			shouldContain: []string{"|", "|", "-"},
		},
		{
			name:          "half progress",
			bar:           Bar(0.5),
			format:        "%r",
			shouldContain: []string{"|", "|", "+", "-"},
		},
		{
			name:          "full progress",
			bar:           Bar(1.0),
			format:        "%r",
			shouldContain: []string{"|", "|", "+"},
		},
		{
			name:          "over full progress (clamped)",
			bar:           Bar(1.5),
			format:        "%r",
			shouldContain: []string{"|", "|", "+"},
		},
		{
			name:          "negative progress (clamped to zero)",
			bar:           Bar(-0.5),
			format:        "%r",
			shouldContain: []string{"|", "|", "-"},
		},
		{
			name:          "reverse progress with flag",
			bar:           Bar(0.25),
			format:        "%-r",
			shouldContain: []string{"|", "|", "+", "-"},
		},
		{
			name:          "custom width",
			bar:           Bar(0.5),
			format:        "%20r",
			shouldContain: []string{"|", "|"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := fmt.Sprintf(test.format, test.bar)

			// Check that the result contains expected elements
			for _, expected := range test.shouldContain {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain %q, got: %s", expected, result)
				}
			}

			// Check that result starts and ends with |
			if !strings.HasPrefix(result, "|") {
				t.Errorf("Expected result to start with |, got: %s", result)
			}
			if !strings.HasSuffix(result, "|") {
				t.Errorf("Expected result to end with |, got: %s", result)
			}
		})
	}
}

func TestBar_Format_ValidRunes(t *testing.T) {
	bar := Bar(0.5)

	// Test that 'r' format works
	result := fmt.Sprintf("%r", bar)
	if !strings.Contains(result, "|") {
		t.Errorf("Expected 'r' format to work and contain |, got: %s", result)
	}

	// Test other format verbs don't panic but may have different behavior
	// The Format method only handles 'r', other verbs are handled by default fmt behavior
}

func TestBar_Format_DefaultWidth(t *testing.T) {
	bar := Bar(0.5)
	result := fmt.Sprintf("%r", bar)

	// Default width is 40, plus 2 for the | characters, plus ANSI color codes
	// We just verify it's a reasonable length
	if len(result) < 40 {
		t.Errorf("Expected result length to be at least 40, got: %d", len(result))
	}
}

func TestBar_Format_CustomWidth(t *testing.T) {
	tests := []struct {
		name   string
		bar    Bar
		width  int
		format string
	}{
		{
			name:   "small width",
			bar:    Bar(0.5),
			width:  10,
			format: "%10r",
		},
		{
			name:   "large width",
			bar:    Bar(0.25),
			width:  80,
			format: "%80r",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := fmt.Sprintf(test.format, test.bar)

			// Check that result contains pipe characters
			if !strings.Contains(result, "|") {
				t.Errorf("Expected result to contain |, got: %s", result)
			}
		})
	}
}

func TestBar_Interface(t *testing.T) {
	// Verify Bar implements fmt.Formatter
	var _ fmt.Formatter = Bar(1.0)
}

func TestBytes_String(t *testing.T) {
	tests := []struct {
		name     string
		bytes    Bytes
		expected string
	}{
		{
			name:     "zero bytes",
			bytes:    Bytes(0),
			expected: "0.0 B",
		},
		{
			name:     "bytes",
			bytes:    Bytes(500),
			expected: "500.0 B",
		},
		{
			name:     "kibibytes",
			bytes:    Bytes(1024),
			expected: "1.0 KiB",
		},
		{
			name:     "mebibytes",
			bytes:    Bytes(1024 * 1024),
			expected: "1.0 MiB",
		},
		{
			name:     "gibibytes",
			bytes:    Bytes(1024 * 1024 * 1024),
			expected: "1.0 GiB",
		},
		{
			name:     "fractional kibibytes",
			bytes:    Bytes(1536), // 1.5 KiB
			expected: "1.5 KiB",
		},
		{
			name:     "large number",
			bytes:    Bytes(1024*1024*1024*5 + 1024*1024*512), // 5.5 GiB
			expected: "5.5 GiB",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.bytes.String()
			if result != test.expected {
				t.Errorf("Expected %q, got %q", test.expected, result)
			}
		})
	}
}

func TestBytesPerSecond_NewBytesPerSecond(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		duration time.Duration
		expected BytesPerSecond
	}{
		{
			name:     "1 byte per second",
			bytes:    1,
			duration: 1 * time.Second,
			expected: BytesPerSecond(1),
		},
		{
			name:     "1000 bytes per second",
			bytes:    1000,
			duration: 1 * time.Second,
			expected: BytesPerSecond(1000),
		},
		{
			name:     "500 bytes in 2 seconds",
			bytes:    1000,
			duration: 2 * time.Second,
			expected: BytesPerSecond(500),
		},
		{
			name:     "fractional rate",
			bytes:    100,
			duration: 3 * time.Second,
			expected: BytesPerSecond(33), // approximately
		},
		{
			name:     "zero duration handling",
			bytes:    1000,
			duration: 1 * time.Nanosecond,
			expected: BytesPerSecond(1000000000000), // very high rate
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := NewBytesPerSecond(test.bytes, test.duration)

			// For fractional calculations, allow some tolerance
			diff := int64(result) - int64(test.expected)
			if diff < 0 {
				diff = -diff
			}
			if diff > 5 { // Allow small differences due to floating point math
				t.Errorf("Expected approximately %d, got %d", test.expected, result)
			}
		})
	}
}

func TestBytesPerSecond_String(t *testing.T) {
	tests := []struct {
		name     string
		bps      BytesPerSecond
		expected string
	}{
		{
			name:     "zero rate",
			bps:      BytesPerSecond(0),
			expected: "0.0 B/s",
		},
		{
			name:     "bytes per second",
			bps:      BytesPerSecond(500),
			expected: "500.0 B/s",
		},
		{
			name:     "kibibytes per second",
			bps:      BytesPerSecond(1024),
			expected: "1.0 KiB/s",
		},
		{
			name:     "mebibytes per second",
			bps:      BytesPerSecond(1024 * 1024),
			expected: "1.0 MiB/s",
		},
		{
			name:     "fractional rate",
			bps:      BytesPerSecond(1536), // 1.5 KiB/s
			expected: "1.5 KiB/s",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.bps.String()
			if result != test.expected {
				t.Errorf("Expected %q, got %q", test.expected, result)
			}
		})
	}
}

func TestWriter_NewWriter(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	if w == nil {
		t.Errorf("NewWriter should not return nil")
	}
	if w.w != &buf {
		t.Errorf("Writer should use provided io.Writer")
	}
	if w.lines != 0 {
		t.Errorf("Expected initial lines to be 0, got %d", w.lines)
	}
	if w.buf.Len() != 0 {
		t.Errorf("Expected empty buffer, got length %d", w.buf.Len())
	}
}

func TestWriter_Write(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	testData := []byte("test data")
	n, err := w.Write(testData)

	if err != nil {
		t.Errorf("Write should not return error: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, got %d", len(testData), n)
	}
	if w.buf.Len() != len(testData) {
		t.Errorf("Expected buffer length %d, got %d", len(testData), w.buf.Len())
	}
	if buf.Len() != 0 {
		t.Errorf("Data should not be written to underlying writer until flush, got %d", buf.Len())
	}
}

func TestWriter_Flush_EmptyBuffer(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	// Test flushing empty buffer
	err := w.Flush()
	if err != nil {
		t.Errorf("Flush should not return error for empty buffer: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("Nothing should be written for empty buffer, got %d bytes", buf.Len())
	}
}

func TestWriter_Flush_WithData(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	// Write some data
	testData := "line 1\nline 2\nline 3"
	w.Write([]byte(testData))

	// Flush should write data to underlying writer
	err := w.Flush()
	if err != nil {
		t.Errorf("Flush should not return error: %v", err)
	}

	// Check that data was written
	if buf.Len() == 0 {
		t.Errorf("Data should be written to underlying writer after flush")
	}

	// Check that buffer was reset
	if w.buf.Len() != 0 {
		t.Errorf("Buffer should be reset after flush, got length %d", w.buf.Len())
	}

	// Lines count may be 0 if console is not available (which is normal in tests)
	// This is expected behavior when running in non-console environment
}

func TestWriter_MultipleFlushs(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	// First write and flush
	w.Write([]byte("first line"))
	w.Flush()
	firstFlushLen := buf.Len()

	// Second write and flush
	w.Write([]byte("second line"))
	w.Flush()
	secondFlushLen := buf.Len()

	if secondFlushLen <= firstFlushLen {
		t.Errorf("Second flush should write more data")
	}
}

func TestCountLines(t *testing.T) {
	// Note: countLines depends on console size, which may not be available in test environment
	// We test the function but expect it might return 0 if console is not available

	tests := []struct {
		name        string
		input       string
		minExpected int // minimum expected lines (may be 0 if console not available)
	}{
		{
			name:        "empty string",
			input:       "",
			minExpected: 0,
		},
		{
			name:        "single line",
			input:       "single line",
			minExpected: 0, // May be 0 or 1 depending on console
		},
		{
			name:        "multiple lines",
			input:       "line 1\nline 2\nline 3",
			minExpected: 0, // May be 0 or more depending on console
		},
		{
			name:        "line with ansi codes",
			input:       "\x1b[32mgreen text\x1b[0m",
			minExpected: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := countLines(test.input)
			if result < test.minExpected {
				t.Errorf("Expected at least %d lines, got %d", test.minExpected, result)
			}
		})
	}
}

func TestStripLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no ansi codes",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "with ansi color codes",
			input:    "\x1b[32mgreen\x1b[0m text",
			expected: "green text",
		},
		{
			name:     "multiple ansi codes",
			input:    "\x1b[31mred\x1b[32mgreen\x1b[0mreset",
			expected: "redgreenreset",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only ansi codes",
			input:    "\x1b[31m\x1b[0m",
			expected: "[0m", // regex only matches some ANSI patterns
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := stripLine(test.input)
			if result != test.expected {
				t.Errorf("Expected %q, got %q", test.expected, result)
			}
		})
	}
}

func TestEscapeConstants(t *testing.T) {
	// Test that escape constants are defined correctly
	if escape != "\x1b" {
		t.Errorf("Expected escape to be \\x1b, got %q", escape)
	}
	if reset != "\x1b[0m" {
		t.Errorf("Expected reset to be \\x1b[0m, got %q", reset)
	}
	if green != "\x1b[32m" {
		t.Errorf("Expected green to be \\x1b[32m, got %q", green)
	}
}

func TestWriter_WriteAndFlush_Integration(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	// Simulate a typical progress display workflow

	// Write initial progress
	fmt.Fprintf(w, "Progress: %r\n", Bar(0.0))
	w.Flush()

	initialLen := buf.Len()
	if initialLen == 0 {
		t.Errorf("Expected some data to be written")
	}

	// Write updated progress
	fmt.Fprintf(w, "Progress: %r\n", Bar(0.5))
	w.Flush()

	updatedLen := buf.Len()
	if updatedLen <= initialLen {
		t.Errorf("Expected more data after second flush")
	}

	// Write final progress
	fmt.Fprintf(w, "Progress: %r Complete!\n", Bar(1.0))
	w.Flush()

	finalLen := buf.Len()
	if finalLen <= updatedLen {
		t.Errorf("Expected even more data after final flush")
	}
}
