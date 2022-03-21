package retry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestShouldGiveUp(t *testing.T) {
	r := NewRetrier(
		WithStrategy(Constant(1*time.Second)),
		WithMaxAttempts(2),
	)

	r.MarkAttempt()
	assert.False(t, r.ShouldGiveUp())

	r.MarkAttempt()
	assert.False(t, r.ShouldGiveUp())

	r.MarkAttempt()
	assert.True(t, r.ShouldGiveUp())
}

func TestShouldGiveUp_Break(t *testing.T) {
	r := NewRetrier(
		WithStrategy(Constant(1*time.Second)),
		WithMaxAttempts(500),
	)

	for i := 0; i < 100; i++ {
		r.MarkAttempt()
	}
	assert.False(t, r.ShouldGiveUp())

	r.Break()
	assert.True(t, r.ShouldGiveUp())
}

func TestShouldGiveUp_Forever(t *testing.T) {
	r := NewRetrier(
		WithStrategy(Constant(1*time.Second)),
		TryForever(),
	)

	// We can't prove that it will never give up (thanks, math!), but we probably won't try many things 10,000 times
	for i := 0; i < 10_000; i++ {
		r.MarkAttempt()
	}

	assert.False(t, r.ShouldGiveUp())
}

func TestNextInterval_ConstantStrategy(t *testing.T) {
	r := NewRetrier(
		WithStrategy(Constant(5*time.Second)),
		WithMaxAttempts(1000),
	)

	intervals := generateIntervals(r, 500)

	for _, interval := range intervals {
		assert.Equal(t, interval, 5*time.Second)
	}
}

func TestNextInterval_ConstantStrategy_WithJitter(t *testing.T) {
	expected := 5 * time.Second
	r := NewRetrier(
		WithStrategy(Constant(expected)),
		WithJitter(),
		WithMaxAttempts(1000),
	)

	intervals := generateIntervals(r, 5)

	for _, interval := range intervals {
		assert.Truef(t,
			withinJitterInterval(interval, expected),
			"actual interval %v was not within 1 second of expected interval %v", interval, expected,
		)
	}
}

func TestNextInterval_ExponentialStrategy(t *testing.T) {
	r := NewRetrier(
		WithStrategy(Exponential(2*time.Second, 0)),
		WithMaxAttempts(1000),
	)

	expectedIntervals := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
	}

	actualIntervals := generateIntervals(r, len(expectedIntervals))

	assert.Equal(t, expectedIntervals, actualIntervals)
}

func TestNextInterval_ExponentialStrategy_WithAdjustment(t *testing.T) {
	r := NewRetrier(
		WithStrategy(Exponential(2*time.Second, 3*time.Second)),
		WithMaxAttempts(1000),
	)

	expectedIntervals := []time.Duration{
		4 * time.Second,
		5 * time.Second,
		7 * time.Second,
		11 * time.Second,
		19 * time.Second,
	}

	actualIntervals := generateIntervals(r, len(expectedIntervals))

	assert.Equal(t, expectedIntervals, actualIntervals)
}

func TestNextInterval_ExponentialStrategy_WithJitter(t *testing.T) {
	r := NewRetrier(
		WithStrategy(Exponential(2*time.Second, 0)),
		WithMaxAttempts(1000),
	)

	expectedIntervals := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
	}

	actualIntervals := generateIntervals(r, len(expectedIntervals))

	for idx, actualInterval := range actualIntervals {
		assert.Truef(
			t,
			withinJitterInterval(actualInterval, expectedIntervals[idx]),
			"actual interval %v wasn't within 1s of expected interval %v", actualInterval, expectedIntervals[idx],
		)
	}
}

func generateIntervals(retrier *Retrier, howMany int) []time.Duration {
	actualIntervals := make([]time.Duration, 0, howMany)
	for i := 0; i < 5; i++ {
		actualIntervals = append(actualIntervals, retrier.NextInterval())
		retrier.MarkAttempt()
	}

	return actualIntervals
}

func withinJitterInterval(this, that time.Duration) bool {
	bigger := this
	smaller := that

	if bigger < smaller {
		bigger, smaller = smaller, bigger
	}

	diff := bigger - smaller

	return diff <= jitterInterval
}
