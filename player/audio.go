package player

type StopReason int

const (
	// PlayOver 播放结束
	PlayOver StopReason = iota
	// TimeOver 定时结束
	TimeOver
	// ForceOver 强制结束
	ForceOver
)

const (
	// SeekSecs is the amount of seconds to skip forward or backward
	SeekSecs = 5
)

// AudioPlayer is an interface for playing audio tracks.
type AudioPlayer interface {
	Play(filePath string, loopCount int, volume float64, callback func(reason StopReason)) (AudioController, error)
}

// AudioController will control playing audio
type AudioController interface {
	Paused() bool
	Pause()
	PauseToggle() bool
	Resume()
	//PlayState() PlayState
	//SeekForward() error
	//SeekBackward() error
	//SpeedUp()
	//SpeedDown()
	Stop(StopReason)
	//VolumeUp()
	//VolumeDown()
	SetVolume(volume float64)
}

// PlayState represents the current state of playing audio.
type PlayState struct {
	Finished bool
	Progress float32
	Position string
	Volume   string
	Speed    string
}
