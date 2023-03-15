package player

import (
	"Broadcast/utils"
	"github.com/faiface/beep"
	"github.com/hajimehoshi/oto"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sync"
)

var (
	mu      sync.Mutex
	mixer   beep.Mixer
	samples [][2]float64
	buf     []byte
	context *oto.Context
	player  *oto.Player
	wgStop  *sync.WaitGroup
	inCycle bool
)

// Init initializes audio playback through speaker. Must be called before using this package.
//
// The bufferSize argument specifies the number of samples of the speaker's buffer. Bigger
// bufferSize means lower CPU usage and more reliable playback. Lower bufferSize means better
// responsiveness and less delay.
func SpInit(sampleRate beep.SampleRate, bufferSize int) error {
	if inCycle {
		utils.Logger.
			WithFields(log.Fields{
				"c": "speaker",
			}).
			Debug("enter SpClose")
		SpClose()
		utils.Logger.
			WithFields(log.Fields{
				"c": "speaker",
			}).
			Debug("exit SpClose")
	}
	mu.Lock()
	defer mu.Unlock()

	mixer = beep.Mixer{}

	numBytes := bufferSize * 4
	samples = make([][2]float64, bufferSize)
	buf = make([]byte, numBytes)

	var err error
	context, err = oto.NewContext(int(sampleRate), 2, 2, numBytes)
	if err != nil {
		return errors.Wrap(err, "failed to initialize speaker")
	}
	player = context.NewPlayer()

	go func() {
		inCycle = true
		defer func() {
			inCycle = false
		}()
		for {
			if wgStop != nil {
				utils.Logger.
					WithFields(log.Fields{
						"c": "speaker",
					}).
					Debug("exit for cycle")
				wgStop.Done()
				return
			} else {
				update()
			}
		}
	}()

	return nil
}

// Close closes the playback and the driver. In most cases, there is certainly no need to call Close
// even when the program doesn't play anymore, because in properly set systems, the default mixer
// handles multiple concurrent processes. It's only when the default device is not a virtual but hardware
// device, that you'll probably want to manually manage the device from your application.
func SpClose() {
	//mu.Lock()
	//defer mu.Unlock()

	utils.Logger.
		WithFields(log.Fields{
			"c": "speaker",
		}).
		Debug("enter speaker close")
	if player != nil {
		utils.Logger.
			WithFields(log.Fields{
				"c": "speaker",
			}).
			Debug("player not nil")
		if inCycle && wgStop == nil {
			utils.Logger.
				WithFields(log.Fields{
					"c": "speaker",
				}).
				Debug("wgStop is nil")
			wgStop = &sync.WaitGroup{}
			wgStop.Add(1)
			wgStop.Wait()
			wgStop = nil
			utils.Logger.
				WithFields(log.Fields{
					"c": "speaker",
				}).
				Debug("close done")
		}
		player.Close()
		context.Close()
		player = nil
	}
	utils.Logger.
		WithFields(log.Fields{
			"c": "speaker",
		}).
		Debug("exit speaker close")
}

// Lock locks the speaker. While locked, speaker won't pull new data from the playing Stramers. Lock
// if you want to modify any currently playing Streamers to avoid race conditions.
//
// Always lock speaker for as little time as possible, to avoid playback glitches.
func SpLock() {
	mu.Lock()
}

// Unlock unlocks the speaker. Call after modifying any currently playing Streamer.
func SpUnlock() {
	mu.Unlock()
}

// Play starts playing all provided Streamers through the speaker.
func SpPlay(s ...beep.Streamer) {
	mu.Lock()
	mixer.Add(s...)
	mu.Unlock()
}

// Clear removes all currently playing Streamers from the speaker.
func SpClear() {
	mu.Lock()
	mixer.Clear()
	mu.Unlock()
}

// update pulls new data from the playing Streamers and sends it to the speaker. Blocks until the
// data is sent and started playing.
func update() {
	mu.Lock()
	//defer mu.Unlock()
	mixer.Stream(samples)
	mu.Unlock()

	for i := range samples {
		for c := range samples[i] {
			val := samples[i][c]
			if val < -1 {
				val = -1
			}
			if val > +1 {
				val = +1
			}
			valInt16 := int16(val * (1<<15 - 1))
			low := byte(valInt16)
			high := byte(valInt16 >> 8)
			buf[i*4+c*2+0] = low
			buf[i*4+c*2+1] = high
		}
	}

	player.Write(buf)
}
