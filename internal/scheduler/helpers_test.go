package scheduler

import (
	"testing"
	"time"
)

func TestConvertIntervalsIntoChunks(t *testing.T) {
	tests := []struct {
		name        string
		intervals   []TimeInterval
		expected    []TimeInterval
		expectError bool
	}{
		{
			name: "single 30-minute interval",
			intervals: []TimeInterval{
				{
					time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 30, 0, 0, time.UTC),
				},
			},
			expected: []TimeInterval{
				{
					time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 15, 0, 0, time.UTC),
				},
				{
					time.Date(2023, 1, 1, 9, 15, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 30, 0, 0, time.UTC),
				},
			},
			expectError: false,
		},
		{
			name: "single 15-minute interval",
			intervals: []TimeInterval{
				{
					time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 15, 0, 0, time.UTC),
				},
			},
			expected: []TimeInterval{
				{
					time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 15, 0, 0, time.UTC),
				},
			},
			expectError: false,
		},
		{
			name: "single 45-minute interval",
			intervals: []TimeInterval{
				{
					time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 45, 0, 0, time.UTC),
				},
			},
			expected: []TimeInterval{
				{
					time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 15, 0, 0, time.UTC),
				},
				{
					time.Date(2023, 1, 1, 9, 15, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 30, 0, 0, time.UTC),
				},
				{
					time.Date(2023, 1, 1, 9, 30, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 45, 0, 0, time.UTC),
				},
			},
			expectError: false,
		},
		{
			name: "multiple intervals",
			intervals: []TimeInterval{
				{
					time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 30, 0, 0, time.UTC),
				},
				{
					time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 14, 15, 0, 0, time.UTC),
				},
			},
			expected: []TimeInterval{
				{
					time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 15, 0, 0, time.UTC),
				},
				{
					time.Date(2023, 1, 1, 9, 15, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 30, 0, 0, time.UTC),
				},
				{
					time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 14, 15, 0, 0, time.UTC),
				},
			},
			expectError: false,
		},
		{
			name:        "empty intervals",
			intervals:   []TimeInterval{},
			expected:    []TimeInterval{},
			expectError: false,
		},
		{
			name: "invalid interval (less than 2 times)",
			intervals: []TimeInterval{
				{time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC)},
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "invalid start time (not on 15-minute boundary)",
			intervals: []TimeInterval{
				{
					time.Date(2023, 1, 1, 9, 5, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 15, 0, 0, time.UTC),
				},
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "invalid end time (not on 15-minute boundary)",
			intervals: []TimeInterval{
				{
					time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 10, 0, 0, time.UTC),
				},
			},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertIntervalsIntoChunks(tt.intervals)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d chunks, got %d", len(tt.expected), len(result))
				return
			}

			for i, chunk := range result {
				if len(chunk) != 2 {
					t.Errorf("chunk %d should have 2 times, got %d", i, len(chunk))
					continue
				}

				expectedChunk := tt.expected[i]
				if !chunk[0].Equal(expectedChunk[0]) || !chunk[1].Equal(expectedChunk[1]) {
					t.Errorf("chunk %d: expected [%v, %v], got [%v, %v]",
						i, expectedChunk[0], expectedChunk[1], chunk[0], chunk[1])
				}
			}
		})
	}
}

func TestGroupConsecutiveChunks(t *testing.T) {
	tests := []struct {
		name     string
		chunks   []TimeInterval
		expected []TimeInterval
	}{
		{
			name: "consecutive 15-minute chunks",
			chunks: []TimeInterval{
				{
					time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 15, 0, 0, time.UTC),
				},
				{
					time.Date(2023, 1, 1, 9, 15, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 30, 0, 0, time.UTC),
				},
				{
					time.Date(2023, 1, 1, 9, 30, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 45, 0, 0, time.UTC),
				},
			},
			expected: []TimeInterval{
				{
					time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 45, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "non-consecutive chunks with gap",
			chunks: []TimeInterval{
				{
					time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 15, 0, 0, time.UTC),
				},
				{
					time.Date(2023, 1, 1, 9, 15, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 30, 0, 0, time.UTC),
				},
				{
					time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 14, 15, 0, 0, time.UTC),
				},
			},
			expected: []TimeInterval{
				{
					time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 30, 0, 0, time.UTC),
				},
				{
					time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 14, 15, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "single chunk",
			chunks: []TimeInterval{
				{
					time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 15, 0, 0, time.UTC),
				},
			},
			expected: []TimeInterval{
				{
					time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 15, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "multiple separate groups",
			chunks: []TimeInterval{
				{
					time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 15, 0, 0, time.UTC),
				},
				{
					time.Date(2023, 1, 1, 9, 15, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 30, 0, 0, time.UTC),
				},
				{
					time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 14, 15, 0, 0, time.UTC),
				},
				{
					time.Date(2023, 1, 1, 14, 15, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 14, 30, 0, 0, time.UTC),
				},
				{
					time.Date(2023, 1, 1, 14, 30, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 14, 45, 0, 0, time.UTC),
				},
			},
			expected: []TimeInterval{
				{
					time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 9, 30, 0, 0, time.UTC),
				},
				{
					time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 14, 45, 0, 0, time.UTC),
				},
			},
		},
		{
			name:     "empty chunks",
			chunks:   []TimeInterval{},
			expected: []TimeInterval{},
		},
		{
			name: "overlapping chunks",
			chunks: []TimeInterval{
				{
					time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 10, 15, 0, 0, time.UTC),
				},
				{
					time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 10, 30, 0, 0, time.UTC),
				},
			},
			expected: []TimeInterval{
				{
					time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 10, 30, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "overlapping chunks in different order",
			chunks: []TimeInterval{
				{
					time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 10, 30, 0, 0, time.UTC),
				},
				{
					time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 10, 15, 0, 0, time.UTC),
				},
			},
			expected: []TimeInterval{
				{
					time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
					time.Date(2023, 1, 1, 10, 30, 0, 0, time.UTC),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := groupConsecutiveChunks(tt.chunks)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d intervals, got %d", len(tt.expected), len(result))
				return
			}

			for i, interval := range result {
				if len(interval) != 2 {
					t.Errorf("interval %d should have 2 times, got %d", i, len(interval))
					continue
				}

				expectedInterval := tt.expected[i]
				if !interval[0].Equal(expectedInterval[0]) || !interval[1].Equal(expectedInterval[1]) {
					t.Errorf("interval %d: expected [%v, %v], got [%v, %v]",
						i, expectedInterval[0], expectedInterval[1], interval[0], interval[1])
				}
			}
		})
	}
}
