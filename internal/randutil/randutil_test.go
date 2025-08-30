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

package randutil

import (
	"math"
	"testing"
)

func TestInt63n(t *testing.T) {
	tests := []struct {
		name string
		n    int64
	}{
		{"small_positive", 10},
		{"medium_positive", 1000},
		{"large_positive", 1000000},
		{"max_int32", int64(math.MaxInt32)},
		{"close_to_max_int64", math.MaxInt64 - 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Int63n(tt.n)
			if result < 0 || result >= tt.n {
				t.Errorf("Int63n(%d) = %d, want 0 <= result < %d", tt.n, result, tt.n)
			}
		})
	}

	// Test multiple calls return different values (statistically likely)
	t.Run("randomness", func(t *testing.T) {
		const samples = 100
		results := make(map[int64]bool)
		duplicates := 0

		for i := 0; i < samples; i++ {
			result := Int63n(10000)
			if results[result] {
				duplicates++
			}
			results[result] = true
		}

		// With 100 samples in a range of 10,000, we expect very few duplicates
		if duplicates > samples/10 {
			t.Errorf("Too many duplicates: %d/%d, randomness may be compromised", duplicates, samples)
		}
	})
}

func TestInt63nPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Int63n(0) should panic, but it didn't")
		}
	}()
	Int63n(0)
}

func TestInt63nNegative(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Int63n(-1) should panic, but it didn't")
		}
	}()
	Int63n(-1)
}

func TestInt63(t *testing.T) {
	// Test multiple calls
	for i := 0; i < 10; i++ {
		result := Int63()
		if result < 0 || result >= math.MaxInt64 {
			t.Errorf("Int63() = %d, want 0 <= result < %d", result, math.MaxInt64)
		}
	}

	// Test randomness - multiple calls should produce different values
	t.Run("randomness", func(t *testing.T) {
		const samples = 50
		results := make(map[int64]bool)
		duplicates := 0

		for i := 0; i < samples; i++ {
			result := Int63()
			if results[result] {
				duplicates++
			}
			results[result] = true
		}

		// With crypto/rand, duplicates in int64 space should be extremely rare
		if duplicates > 0 {
			t.Errorf("Unexpected duplicates: %d/%d in int64 space", duplicates, samples)
		}
	})
}

func TestIntn(t *testing.T) {
	tests := []struct {
		name string
		n    int
	}{
		{"small_positive", 10},
		{"medium_positive", 1000},
		{"large_positive", 1000000},
		{"max_int32", math.MaxInt32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Intn(tt.n)
			if result < 0 || result >= tt.n {
				t.Errorf("Intn(%d) = %d, want 0 <= result < %d", tt.n, result, tt.n)
			}
		})
	}

	// Test randomness
	t.Run("randomness", func(t *testing.T) {
		const samples = 100
		results := make(map[int]bool)
		duplicates := 0

		for i := 0; i < samples; i++ {
			result := Intn(1000)
			if results[result] {
				duplicates++
			}
			results[result] = true
		}

		// With 100 samples in a range of 1,000, we expect few duplicates
		if duplicates > samples/5 {
			t.Errorf("Too many duplicates: %d/%d, randomness may be compromised", duplicates, samples)
		}
	})
}

func TestIntnPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Intn(0) should panic, but it didn't")
		}
	}()
	Intn(0)
}

func TestIntnNegative(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Intn(-1) should panic, but it didn't")
		}
	}()
	Intn(-1)
}

func TestInt(t *testing.T) {
	// Test multiple calls
	for i := 0; i < 10; i++ {
		result := Int()
		if result < 0 {
			t.Errorf("Int() = %d, want result >= 0", result)
		}
	}

	// Test randomness - multiple calls should produce different values
	t.Run("randomness", func(t *testing.T) {
		const samples = 50
		results := make(map[int]bool)
		duplicates := 0

		for i := 0; i < samples; i++ {
			result := Int()
			if results[result] {
				duplicates++
			}
			results[result] = true
		}

		// With crypto/rand in int space, duplicates should be very rare
		if duplicates > 0 {
			t.Errorf("Unexpected duplicates: %d/%d in int space", duplicates, samples)
		}
	})
}

func TestAllFunctionsConsistency(t *testing.T) {
	// Test that functions are consistent with each other
	n := int64(1000)

	for i := 0; i < 10; i++ {
		int63nResult := Int63n(n)
		intnResult := int64(Intn(int(n)))

		// Both should be in the same range
		if int63nResult < 0 || int63nResult >= n {
			t.Errorf("Int63n consistency check failed: %d not in [0, %d)", int63nResult, n)
		}
		if intnResult < 0 || intnResult >= n {
			t.Errorf("Intn consistency check failed: %d not in [0, %d)", intnResult, n)
		}
	}
}

func TestEdgeCases(t *testing.T) {
	// Test with n = 1 (should always return 0)
	t.Run("n_equals_1", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			if result := Int63n(1); result != 0 {
				t.Errorf("Int63n(1) = %d, want 0", result)
			}
			if result := Intn(1); result != 0 {
				t.Errorf("Intn(1) = %d, want 0", result)
			}
		}
	})

	// Test with n = 2 (should return 0 or 1)
	t.Run("n_equals_2", func(t *testing.T) {
		results := make(map[int64]bool)
		for i := 0; i < 50; i++ {
			result := Int63n(2)
			if result != 0 && result != 1 {
				t.Errorf("Int63n(2) = %d, want 0 or 1", result)
			}
			results[result] = true
		}
		// Should have seen both 0 and 1 with high probability
		if !results[0] || !results[1] {
			t.Errorf("Int63n(2) should produce both 0 and 1, got results: %v", results)
		}
	})
}

func BenchmarkInt63n(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Int63n(1000000)
	}
}

func BenchmarkInt63(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Int63()
	}
}

func BenchmarkIntn(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Intn(1000000)
	}
}

func BenchmarkInt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Int()
	}
}
