package player

import (
	"Broadcast/utils"
	"fmt"
	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	//speakerInitialized bool  = false
	muPlay    sync.Mutex
	playCount int32 = 0
)

const (
	// beep quality to use for playing audio
	quality = 4
)

var (
	// maxSampleRate is used for resampling various audio formats. We also set
	// the sample rate of the speaker to this, so it essentially controls the
	// maximum quality of files played by BeepAudioPlayer.
	maxSampleRate beep.SampleRate = 44100
)

// BeepAudioPlayer is an audio player implementation that uses beep
type BeepAudioPlayer struct {
	name string
}

// BeepController manages playing audio.
//
// TODO: make this an interface. this is fine for now since we're only using
// beep our audio player.
type BeepController struct {
	name       string
	audioPanel *audioPanel
	//path       string
	stopReason StopReason
	callback   func(StopReason)
}

// audioPanel is the audio panel for the controller
type audioPanel struct {
	ctrl     *beep.Ctrl
	volume   *effects.Volume
	streamer beep.StreamSeekCloser
	//reSampler  *beep.Resampler
	//sampleRate *beep.SampleRate
}

func (a *audioPanel) free() {
	//a.sampleRate = nil
	//a.reSampler = nil
	_ = a.streamer.Close()
	a.streamer = nil
	a.volume = nil
	a.ctrl = nil

}

// newAudioPanel creates a new audio panel.
//
// count - number of times to repeat the track
func newAudioPanel(sampleRate beep.SampleRate, streamer beep.StreamSeekCloser, count int) *audioPanel {
	ctrl := &beep.Ctrl{Streamer: beep.Loop(count, streamer)}

	utils.Logger.WithFields(log.Fields{
		"src": sampleRate,
		"dst": maxSampleRate,
	}).Debug("resampling")

	reSampler := beep.Resample(quality, sampleRate, maxSampleRate, ctrl)

	volume := &effects.Volume{Streamer: reSampler, Base: 2}
	return &audioPanel{
		ctrl:     ctrl,
		volume:   volume,
		streamer: streamer,
		//reSampler:  reSampler,
		//sampleRate: &sampleRate,
	}
}

func TryLock(fName string, name string) bool {
	utils.Logger.
		WithFields(log.Fields{
			"c":    "beep",
			"name": name,
			"func": fName,
		}).Debug("before try lock")
	succ := muPlay.TryLock()
	utils.Logger.
		WithFields(log.Fields{
			"c":    "beep",
			"name": name,
			"func": fName,
			"succ": succ,
		}).Debug("after try lock")
	return succ
}

func Lock(fName string, name string) {
	utils.Logger.
		WithFields(log.Fields{
			"c":    "beep",
			"name": name,
			"func": fName,
		}).Debug("before lock")
	muPlay.Lock()
	utils.Logger.
		WithFields(log.Fields{
			"c":    "beep",
			"name": name,
			"func": fName,
		}).Debug("after lock")
}

func Unlock(fName string, name string) {
	utils.Logger.
		WithFields(log.Fields{
			"c":    "beep",
			"name": name,
			"func": fName,
		}).Debug("before unlock")
	muPlay.Unlock()
	utils.Logger.
		WithFields(log.Fields{
			"c":    "beep",
			"name": name,
			"func": fName,
		}).Debug("after unlock")
}

// Play a track and return a controller that lets you perform changes to a running track.
func (bmp *BeepAudioPlayer) Play(fileName string, loopCount int, volume float64, callback func(reason StopReason)) (AudioController, error) {
	atomic.AddInt32(&playCount, 1)
	Lock("Play", bmp.name)
	defer Unlock("Play", bmp.name)

	c := BeepController{
		//path:     fileName,
		name:     bmp.name,
		callback: callback,
	}

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	// do not close file io, this should get freed up when we close the streamer
	//defer f.Close()

	var s beep.StreamSeekCloser
	var format beep.Format

	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".mp3":
		s, format, err = mp3.Decode(f)
		if err != nil {
			return nil, err
		}
	case ".flac":
		s, format, err = flac.Decode(f)
		if err != nil {
			return nil, err
		}
	case ".ogg":
		s, format, err = vorbis.Decode(f)
		if err != nil {
			return nil, err
		}
	case ".wav":
		s, format, err = wav.Decode(f)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("不支持的文件类型[%s]", filepath.Ext(fileName))
	}

	if atomic.LoadInt32(&playCount) <= 1 {
		utils.Logger.
			WithFields(log.Fields{
				"c":          "beep",
				"name":       bmp.name,
				"sampleRate": format.SampleRate,
				"file":       fileName,
			}).Debug("init speaker")
		_ = SpInit(maxSampleRate, format.SampleRate.N(time.Second/30))
		utils.Logger.
			WithFields(log.Fields{
				"c":    "beep",
				"name": bmp.name,
				"file": fileName,
			}).
			Debug("init speaker ok!")
	}

	c.audioPanel = newAudioPanel(format.SampleRate, s, loopCount)

	// WARNING: speaker.Play is async
	c.audioPanel.volume.Volume = volume
	c.stopReason = PlayOver
	utils.Logger.
		WithFields(log.Fields{
			"c": "beep",

			"name": bmp.name,
			"file": fileName,
		}).
		Debug("begin call play...")
	SpPlay(beep.Seq(c.audioPanel.volume, beep.Callback(func() {
		utils.Logger.
			WithFields(log.Fields{
				"c":                        "beep",
				"name":                     bmp.name,
				"file":                     fileName,
				"seq callback stop reason": c.stopReason,
			}).
			Debug("streamer callback firing, ", fileName)
		if c.stopReason == PlayOver {
			time.AfterFunc(time.Millisecond*100, func() {
				c.Stop(c.stopReason)
			})
		}
		if c.callback != nil {
			utils.Logger.
				WithFields(log.Fields{
					"c":    "beep",
					"name": bmp.name,
					"file": fileName,
				}).
				Debug("play callback")
			c.callback(c.stopReason)
			c.callback = nil
		}
	})))
	utils.Logger.
		WithFields(log.Fields{
			"c":    "beep",
			"name": bmp.name,
			"file": fileName,
		}).
		Debug("end call play...")
	// 加个延时，给播放进程一个启动时间
	time.Sleep(time.Second)

	return &c, nil
}

// PlayState returns the current state of playing audio.
//func (c *BeepController) PlayState() PlayState {
//	speaker.Lock()
//	p := c.audioPanel.streamer.Position()
//	position := c.audioPanel.sampleRate.D(p)
//	l := c.audioPanel.streamer.Len()
//	length := c.audioPanel.sampleRate.D(l)
//	percentageComplete := float32(p) / float32(l)
//	volume := c.audioPanel.volume.Volume
//	speed := c.audioPanel.reSampler.Ratio()
//	finished := c.audioPanel.finished
//	speaker.Unlock()
//
//	positionStatus := fmt.Sprintf("%v / %v", position.Round(time.Second), length.Round(time.Second))
//	volumeStatus := fmt.Sprintf("%.1f", volume)
//	speedStatus := fmt.Sprintf("%.3fx", speed)
//
//	prog := PlayState{
//		Progress: percentageComplete,
//		Position: positionStatus,
//		Volume:   volumeStatus,
//		Speed:    speedStatus,
//		Finished: finished,
//	}
//	return prog
//}

func (c *BeepController) Pause() {
	Lock("Pause", c.name)
	defer Unlock("Pause", c.name)
	if c.audioPanel == nil {
		return
	}

	c.audioPanel.ctrl.Paused = true
}

// PauseToggle pauses/unpauses audio. Returns true if currently paused, false if unpaused.
func (c *BeepController) PauseToggle() bool {
	Lock("PauseToggle", c.name)
	defer Unlock("PauseToggle", c.name)
	if c.audioPanel == nil {
		return false
	}

	c.audioPanel.ctrl.Paused = !c.audioPanel.ctrl.Paused
	return c.audioPanel.ctrl.Paused
}

// Paused returns current pause state
func (c *BeepController) Paused() bool {
	Lock("Paused", c.name)
	defer Unlock("Paused", c.name)
	if c.audioPanel == nil {
		return false
	}

	return c.audioPanel.ctrl.Paused
}

func (c *BeepController) Resume() {
	Lock("Resume", c.name)
	defer Unlock("Resume", c.name)
	if c.audioPanel == nil {
		return
	}

	c.audioPanel.ctrl.Paused = false
}

// SetVolume the playing track
func (c *BeepController) SetVolume(volume float64) {
	Lock("SetVolume", c.name)
	defer Unlock("SetVolume", c.name)

	if c.audioPanel == nil {
		return
	}

	c.audioPanel.volume.Volume = volume
}

// VolumeUp the playing track
//func (c *BeepController) VolumeUp() {
//	speaker.Lock()
//	defer speaker.Unlock()
//
//	c.audioPanel.volume.Volume += 0.1
//}

// VolumeDown the playing track
//func (c *BeepController) VolumeDown() {
//	speaker.Lock()
//	defer speaker.Unlock()
//
//	c.audioPanel.volume.Volume -= 0.1
//}

// SpeedUp increases speed
//func (c *BeepController) SpeedUp() {
//	speaker.Lock()
//	defer speaker.Unlock()
//
//	c.audioPanel.reSampler.SetRatio(c.audioPanel.reSampler.Ratio() * 16 / 15)
//}

// SpeedDown slows down speed
//func (c *BeepController) SpeedDown() {
//	speaker.Lock()
//	defer speaker.Unlock()
//
//	c.audioPanel.reSampler.SetRatio(c.audioPanel.reSampler.Ratio() * 15 / 16)
//}

// SeekForward moves progress forward
//func (c *BeepController) SeekForward() error {
//	speaker.Lock()
//	defer speaker.Unlock()
//
//	newPos := c.audioPanel.streamer.Position()
//	newPos += c.audioPanel.sampleRate.N(time.Second * SeekSecs)
//	if newPos < 0 {
//		newPos = 0
//	}
//	if newPos >= c.audioPanel.streamer.Len() {
//		newPos = c.audioPanel.streamer.Len() - SeekSecs
//	}
//	if err := c.audioPanel.streamer.Seek(newPos); err != nil {
//		return fmt.Errorf("could not seek to new position [%d]: %s", newPos, err)
//	}
//	return nil
//}

// SeekBackward moves progress backward
//func (c *BeepController) SeekBackward() error {
//	speaker.Lock()
//	defer speaker.Unlock()
//
//	newPos := c.audioPanel.streamer.Position()
//	newPos -= c.audioPanel.sampleRate.N(time.Second * SeekSecs)
//	if newPos < 0 {
//		newPos = 0
//	}
//	if newPos >= c.audioPanel.streamer.Len() {
//		newPos = c.audioPanel.streamer.Len() - 1
//	}
//	if err := c.audioPanel.streamer.Seek(newPos); err != nil {
//		return fmt.Errorf("could not seek to new position [%d]: %s", newPos, err)
//	}
//	return nil
//}

// Stop must be thread safe
func (c *BeepController) Stop(reason StopReason) {
	if TryLock("Stop", c.name) {
		defer Unlock("Stop", c.name)
	} else {
		utils.Logger.
			WithFields(log.Fields{
				"c":      "beep",
				"name":   c.name,
				"reason": reason,
			}).Debug("try lock fail, exit stop ")
		return
	}

	if c.audioPanel == nil {
		return
	}
	// free up streamer
	// NOTE: this will cause the stremer to finish, and the seq callback will
	// fire
	c.stopReason = reason
	if atomic.LoadInt32(&playCount) <= 1 {
		utils.Logger.
			WithFields(log.Fields{
				"c":      "beep",
				"name":   c.name,
				"reason": reason,
			}).Debug("enter close speaker")
		atomic.StoreInt32(&playCount, 0)
		SpClose()
		utils.Logger.
			WithFields(log.Fields{
				"c":      "beep",
				"name":   c.name,
				"reason": reason,
			}).Debug("exit close speaker")
	} else {
		atomic.AddInt32(&playCount, -1)
	}

	if c.audioPanel.streamer != nil {
		utils.Logger.
			WithFields(log.Fields{
				"c":      "beep",
				"name":   c.name,
				"reason": reason,
			}).Debug("closing audioPanel streamer")
		c.audioPanel.free()
		c.audioPanel = nil
	}

	utils.Logger.
		WithFields(log.Fields{
			"c":      "beep",
			"name":   c.name,
			"reason": reason,
		}).Debugf("stop reason: %v", reason)
	if reason != PlayOver && c.callback != nil {
		utils.Logger.
			WithFields(log.Fields{
				"c":      "beep",
				"name":   c.name,
				"reason": reason,
			}).Debug("stop callback")
		c.callback(reason)
		c.callback = nil
	}
}
