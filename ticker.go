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
	prevFPS        []float32
	prevFPSOffset  int
}

const (
	MAX_PREV_FPS = 500
)

func NewTicker(targetFPS int64) *Ticker {
	ticker := &Ticker{
		last: time.Now(),
	}

	ticker.SetTargetFPS(targetFPS)
	ticker.prevFPS = make([]float32, MAX_PREV_FPS)

	return ticker
}

func (this *Ticker) SetTargetFPS(target int64) {
	this.targetFrametime = time.Second / time.Duration(target)
	this.targetFPS = time.NewTicker(this.targetFrametime)
}

func (this *Ticker) Tick() (deltat, framerate float64) {
	this.deltat = time.Since(this.last).Seconds()
	this.last = time.Now()

	if this.frametime >= 1 {
		this.totalFrames += int64(this.frames)
		this.totalFrametime += this.frametime

		this.frametime = 0
		this.frames = 0
	} else {
		this.frames++
		this.frametime += this.deltat
	}

	if this.frametime == 0 {
		this.avgFramerate = float64(this.totalFrames) / this.totalFrametime
		this.framerate = this.avgFramerate
	} else {
		this.framerate = float64(this.frames) / this.frametime
	}

	if this.prevFPSOffset == MAX_PREV_FPS {
		this.prevFPSOffset = 0
	}

	this.prevFPS[this.prevFPSOffset] = float32(this.framerate)
	this.prevFPSOffset++

	return this.deltat, this.framerate
}

func (this *Ticker) Reset() {
	this.last = time.Now()
}

func (this *Ticker) Wait() {
	<-this.targetFPS.C
}
