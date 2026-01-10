package telegram

import (
	"testing"
	"time"
)

func TestOffsetClock(t *testing.T) {
	// Create a clock with a known negative offset (e.g., -1 hour)
	// If local time is 10:00, offset clock should return 09:00
	offset := -1 * time.Hour
	c := &OffsetClock{offset: offset}

	now := time.Now()
	clockNow := c.Now()

	// Allow for small execution time difference
	diff := clockNow.Sub(now.Add(offset))
	if diff < -time.Second || diff > time.Second {
		t.Errorf("Clock time incorrect. Expected ~%v, got %v (diff: %v)", now.Add(offset), clockNow, diff)
	}
}

func TestSyncTime(t *testing.T) {
	// This test requires internet connection
	offset := syncTime()
	t.Logf("Calculated offset: %v", offset)

	// We can't strictly assert the value since we don't know the real time vs system time here,
	// but it shouldn't panic and should return 'some' duration (or 0 if failed, which logs error).
}
