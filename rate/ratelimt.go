package rate

import (
	"sync"
	"time"
)

// Window represents a fixed-window
type Window interface {
	// Start returns the start boundary
	Start() time.Time

	// Count returns the accumulated count
	Count() int64

	// AddCount increments the accumulated count by n
	AddCount(n int64)

	// Reset sets the state of the window with the given settings
	Reset(s time.Time, c int64)
}

type NewWindow func() Window

type LocalWindow struct {
	start int64
	count int64
}

func NewLocalWindow() *LocalWindow {
	return &LocalWindow{}
}

func (w *LocalWindow) Start() time.Time {
	return time.Unix(0, w.start)
}

func (w *LocalWindow) Count() int64 {
	return w.count
}

func (w *LocalWindow) AddCount(n int64) {
	w.count += n
}

func (w *LocalWindow) Reset(s time.Time, c int64) {
	w.start = s.UnixNano()
	w.count = c
}

type RateLimiter struct {
	size  time.Duration
	limit int64

	mu sync.Mutex

	curr Window
	prev Window
}

func NewRateLimiter(size time.Duration, limit int64, newWindow NewWindow) *RateLimiter {
	currWin := newWindow()

	// The previous window is static (i.e. no add changes will happen within it),
	// so we always create it as an instance of LocalWindow
	prevWin := NewLocalWindow()

	return &RateLimiter{
		size:  size,
		limit: limit,
		curr:  currWin,
		prev:  prevWin,
	}

}

// Size returns the time duration of one window size
func (rl *RateLimiter) Size() time.Duration {
	return rl.size
}

// Limit returns the maximum events permitted to happen during one window size
func (rl *RateLimiter) Limit() int64 {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.limit
}

func (rl *RateLimiter) SetLimit(limit int64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.limit = limit
}

// shorthand for GrantN(time.Now(), 1)

func (rl *RateLimiter) Grant() bool {
	return rl.GrantN(time.Now(), 1)
}

// reports whether n events may happen at time now

func (rl *RateLimiter) GrantN(now time.Time, n int64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.advance(now)

	elapsed := now.Sub(rl.curr.Start())
	weight := float64(rl.size-elapsed) / float64(rl.size)
	count := int64(weight*float64(rl.prev.Count())) + rl.curr.Count()

	if count+n > rl.limit {
		return false
	}

	rl.curr.AddCount(n)
	return true
}

// advance updates the current/previous windows resulting from the passage of time
func (rl *RateLimiter) advance(now time.Time) {
	// Calculate the start boundary of the expected current-window.
	newCurrStart := now.Truncate(rl.size)

	diffSize := newCurrStart.Sub(rl.curr.Start()) / rl.size
	if diffSize >= 1 {
		// The current-window is at least one-window-size behind the expected one.
		newPrevCount := int64(0)
		if diffSize == 1 {
			// The new previous-window will overlap with the old current-window,
			// so it inherits the count.
			newPrevCount = rl.curr.Count()
		}

		rl.prev.Reset(newCurrStart.Add(-rl.size), newPrevCount)
		// The new current-window always has zero count.
		rl.curr.Reset(newCurrStart, 0)
	}
}
