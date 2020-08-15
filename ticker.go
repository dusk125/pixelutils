package pixelutils

import "time"

type Ticker struct {
	frames          int
	frametime       float64
	last            time.Time
	deltat          float64
	framerate       float64
	targetFPS       *time.Ticker
	targetFrametime time.Duration

	totalFrames    int64
	totalFrametime float64
	avgFramerate   float64
	storePrev      bool
	prevFPS        []float32
	prevFPSOffset  int
}

const (
	MAX_PREV_FPS = 500
)

// NewTicker returns a new ticker with the given target fps and 500 stored previous framerates.
func NewTicker(targetFPS int64) *Ticker {
	return NewTickerV(targetFPS, MAX_PREV_FPS)
}

// NewTickerV returns a new ticker with the given target fps and a custom amount of stored previous framerates.
// 	If maxPrevFPS is <= 0, don't store any previous framerates.
func NewTickerV(targetFPS, maxPrevFPS int64) *Ticker {
	ticker := &Ticker{
		last: time.Now(),
	}

	ticker.SetTargetFPS(targetFPS)
	ticker.storePrev = maxPrevFPS != 0
	if ticker.storePrev {
		ticker.prevFPS = make([]float32, maxPrevFPS)
	}

	return ticker
}

// SetTargetFPS sets the target ticker rate; will be less than or equal to target.
func (ticker *Ticker) SetTargetFPS(target int64) {
	ticker.targetFrametime = time.Second / time.Duration(target)
	if ticker.targetFPS != nil {
		ticker.targetFPS.Stop()
	}
	ticker.targetFPS = time.NewTicker(ticker.targetFrametime)
}

// Tick ticks the tickers and calculates the framerate and framedelta
func (ticker *Ticker) Tick() (deltat, framerate float64) {
	ticker.deltat = time.Since(ticker.last).Seconds()
	ticker.last = time.Now()

	if ticker.frametime >= 1 {
		ticker.totalFrames += int64(ticker.frames)
		ticker.totalFrametime += ticker.frametime

		ticker.frametime = 0
		ticker.frames = 0
	} else {
		ticker.frames++
		ticker.frametime += ticker.deltat
	}

	if ticker.frametime == 0 {
		ticker.avgFramerate = float64(ticker.totalFrames) / ticker.totalFrametime
		ticker.framerate = ticker.avgFramerate
	} else {
		ticker.framerate = float64(ticker.frames) / ticker.frametime
	}

	if ticker.storePrev {
		if ticker.prevFPSOffset == MAX_PREV_FPS {
			ticker.prevFPSOffset = 0
		}

		ticker.prevFPS[ticker.prevFPSOffset] = float32(ticker.framerate)
		ticker.prevFPSOffset++
	}

	return ticker.deltat, ticker.framerate
}

// Reset the ticker
func (ticker *Ticker) Reset() {
	ticker.last = time.Now()
}

// Wait for the timer to complete its timeout
func (ticker *Ticker) Wait() {
	<-ticker.targetFPS.C
}

// Return the last framerate
func (ticker Ticker) Framerate() float64 {
	return ticker.framerate
}

// Return the last Deltat
func (ticker Ticker) Deltat() float64 {
	return ticker.deltat
}

// Return the average framerate
func (ticker Ticker) AvgFramerate() float64 {
	return ticker.avgFramerate
}

// Return the list of previous framerates
func (ticker Ticker) PrevFramerates() []float32 {
	return ticker.prevFPS
}

// Return the current target frametime
func (ticker Ticker) TargetFrametime() time.Duration {
	return ticker.targetFrametime
}
