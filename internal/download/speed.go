package download

import (
	"context"
	"io"

	"golang.org/x/time/rate"
)

// SpeedLimiter wraps an io.Reader to enforce a download rate limit
type SpeedLimiter struct {
	reader  io.Reader
	limiter *rate.Limiter
}

// NewSpeedLimiter creates a new SpeedLimiter with the given speed limit (in KBps)
func NewSpeedLimiter(reader io.Reader, speedLimitKbps int) *SpeedLimiter {
	if speedLimitKbps <= 0 {
		// No speed limit, return a normal reader
		return &SpeedLimiter{
			reader:  reader,
			limiter: nil, // No rate limiting
		}
	}

	// Convert KBps to bytes per second
	limit := rate.Limit(speedLimitKbps * 1024) // KBps to Bps
	limiter := rate.NewLimiter(limit, int(limit))

	return &SpeedLimiter{
		reader:  reader,
		limiter: limiter,
	}
}

// Read implements io.Reader with rate limiting
func (s *SpeedLimiter) Read(p []byte) (int, error) {
	n, err := s.reader.Read(p)

	// Apply rate limiting only if a limiter is set
	if err == nil && s.limiter != nil {
		// ✅ FIX: Use `context.TODO()` instead of `nil`
		s.limiter.WaitN(context.TODO(), n) // Blocks until enough tokens are available
	}

	return n, err
}