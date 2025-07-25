package scheduler

import (
	"fmt"
	"time"
)

func convertIntervalsIntoChunks(intervals []TimeInterval) ([]TimeInterval, error) {
	var chunks []TimeInterval
	chunkDuration := 15 * time.Minute

	for _, interval := range intervals {
		if len(interval) < 2 {
			return nil, fmt.Errorf("interval must have at least 2 time values")
		}

		start := interval[0]
		end := interval[1]

		// Check that start and end times land on 00, 15, 30, or 45 minutes
		if start.Minute()%15 != 0 {
			return nil, fmt.Errorf("start time must be on 00, 15, 30, or 45 minutes, got %d", start.Minute())
		}
		if end.Minute()%15 != 0 {
			return nil, fmt.Errorf("end time must be on 00, 15, 30, or 45 minutes, got %d", end.Minute())
		}

		current := start
		for current.Before(end) {
			chunkEnd := current.Add(chunkDuration)
			chunk := TimeInterval{current, chunkEnd}
			chunks = append(chunks, chunk)
			current = chunkEnd
		}
	}

	return chunks, nil
}

func groupConsecutiveChunks(chunks []TimeInterval) []TimeInterval {
	if len(chunks) == 0 {
		return []TimeInterval{}
	}

	// Create a copy of chunks to avoid modifying the original
	chunksCopy := make([]TimeInterval, len(chunks))
	copy(chunksCopy, chunks)

	// First, sort chunks by start time
	sortChunks(chunksCopy)

	// Merge overlapping chunks first
	merged := mergeOverlappingChunks(chunksCopy)

	// Then group consecutive chunks
	return groupConsecutiveMergedChunks(merged)
}

func sortChunks(chunks []TimeInterval) {
	// Sort chunks by start time
	for i := 0; i < len(chunks)-1; i++ {
		for j := i + 1; j < len(chunks); j++ {
			if chunks[i][0].After(chunks[j][0]) {
				chunks[i], chunks[j] = chunks[j], chunks[i]
			}
		}
	}
}

func mergeOverlappingChunks(chunks []TimeInterval) []TimeInterval {
	if len(chunks) == 0 {
		return []TimeInterval{}
	}

	var merged []TimeInterval
	current := TimeInterval{chunks[0][0], chunks[0][1]}

	for i := 1; i < len(chunks); i++ {
		next := chunks[i]

		// Check if chunks overlap or are adjacent
		if !current[1].Before(next[0]) {
			// Overlapping or adjacent - merge them
			if next[1].After(current[1]) {
				current[1] = next[1]
			}
		} else {
			// No overlap - add current to result and move to next
			merged = append(merged, TimeInterval{current[0], current[1]})
			current = TimeInterval{next[0], next[1]}
		}
	}

	// Add the last merged chunk
	merged = append(merged, TimeInterval{current[0], current[1]})

	return merged
}

func groupConsecutiveMergedChunks(chunks []TimeInterval) []TimeInterval {
	if len(chunks) == 0 {
		return []TimeInterval{}
	}

	var grouped []TimeInterval

	start := chunks[0][0]
	end := chunks[0][1]

	for i := 1; i < len(chunks); i++ {
		currentStart := chunks[i][0]
		currentEnd := chunks[i][1]

		// Check if current chunk is consecutive (starts exactly when previous ended)
		if currentStart.Equal(end) {
			// Extend the current interval
			end = currentEnd
		} else {
			// Gap found, save current interval and start a new one
			grouped = append(grouped, TimeInterval{start, end})
			start = currentStart
			end = currentEnd
		}
	}

	// Add the final interval
	grouped = append(grouped, TimeInterval{start, end})

	return grouped
}
