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
