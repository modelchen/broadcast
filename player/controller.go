package player

import (
	"Broadcast/utils"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"sync"
	"time"
)

type ControlState int

const (
	// CsStop 停止
	CsStop ControlState = iota
	// CsRun 运行
	CsRun
	// CsProgramPlay 节目播放
	CsProgramPlay
	// CsPause 暂停
	CsPause
	// CsTempPlay 临时播放
	CsTempPlay
	// CsInnerPlay 播放内置
	CsInnerPlay
)

// Controller 媒体播放控制器
type Controller struct {
	// enable 使能
	enable bool
	// filePath 文件存放路径
	filePath string
	// volume 音量
	volume float64
	// bill 节目单
	bill *Bill
	// playCron 定时器
	manager *ProgramManager
	// pgmPlayer 节目播放器
	pgmPlayer AudioPlayer
	// tmpPlayer 临时播放器
	tmpPlayer AudioPlayer
	// pgmCtrl 节目播放控制器
	pgmCtrl AudioController
	// tmpCtrl 临时播放控制器
	tmpCtrl AudioController
	// preTempLevel 前面正在播放的临时文件的紧急程度 1~9, 1级最高，9级最低
	preTempLevel int
	// state 状态 0：停止；1：播放；2：暂停；3：临时插入播放；4：播放内置文件
	state     ControlState
	wgPgmOver *sync.WaitGroup
	wgTmpOver *sync.WaitGroup
	//mxPgm     sync.Mutex
	inProgram bool

	stopWaitTime time.Duration
}

func BytesToBill(bsBill []byte) *Bill {
	bill := &Bill{}
	if err := json.Unmarshal(bsBill, bill); err != nil {
		utils.Logger.Errorf("转换节目单JSON字符串出错，%s", err.Error())
		return nil
	}
	return bill
}

func StrToBill(strBill string) *Bill {
	return BytesToBill([]byte(strBill))
}

// NewController 用 节目单JSON字符串 新建媒体播放控制器
//
//	strBill 节目单JSON字符串
func NewController(filePath, strBill string, enable bool, stopWaitTime time.Duration) (c *Controller) {
	return NewControllerWithBill(filePath, StrToBill(strBill), enable, stopWaitTime)
}

// NewControllerWithBill 用 节目单对象 新建媒体播放控制器
//
//	bill 节目单对象
func NewControllerWithBill(filePath string, bill *Bill, enable bool, stopWaitTime time.Duration) (c *Controller) {
	c = &Controller{
		enable:       enable,
		filePath:     filePath,
		pgmPlayer:    &BeepAudioPlayer{},
		tmpPlayer:    &BeepAudioPlayer{},
		preTempLevel: 9,
		state:        CsStop,
		manager:      NewProgramManager(),
		wgPgmOver:    nil,
		wgTmpOver:    &sync.WaitGroup{},
		inProgram:    false,
		stopWaitTime: stopWaitTime,
	}
	if c.filePath == "" {
		c.filePath = utils.GetCurrentPath() + "files/"
	}
	_ = c.SetBill(bill, false)

	return c
}

func (c *Controller) SetBill(newBill *Bill, needSave bool) error {
	return c.beforeProcess(func() (err error) {
		//c.mxPgm.Lock()
		//defer c.mxPgm.Unlock()
		if newBill == nil {
			return errors.New("节目单对象为空")
		}
		if err = newBill.Check(); err != nil {
			return err
		}
		if needSave {
			var strBill []byte
			if strBill, err = json.Marshal(newBill); err != nil {
				return err
			}
			if err = utils.WriteConf(utils.CfgBill, string(strBill)); err != nil {
				return fmt.Errorf("保存节目单失败，%s", err.Error())
			}
		}
		preState := c.state
		c.StopProgram()
		c.manager.Clear()
		c.bill = newBill

		for _, program := range c.bill.Slots {
			c.manager.AddProgram(fmt.Sprintf("0 %d %d * * *", program.StartMinute, program.StartHour), program, c.StartProgram)
			c.manager.AddFunc(fmt.Sprintf("0 %d %d * * *", program.EndMinute, program.EndHour), c.StopProgram)
		}

		if preState != CsStop {
			c.startProgramManager()
		}
		return nil
	})
}

func (c *Controller) stopTempPlay(reason StopReason) {
	if c.tmpCtrl == nil {
		return
	}
	c.preTempLevel = 9
	c.tmpCtrl.Stop(reason)
	c.tmpCtrl = nil
	time.Sleep(c.stopWaitTime)
}

func (c *Controller) stopPgmPlay(reason StopReason) {
	if c.pgmCtrl == nil {
		return
	}
	c.pgmCtrl.Stop(reason)
	c.pgmCtrl = nil
}

func (c *Controller) pauseTempPlay() {
	if c.tmpCtrl == nil {
		return
	}
	c.tmpCtrl.Pause()
}

func (c *Controller) pausePgmPlay() {
	if c.pgmCtrl == nil {
		return
	}
	c.pgmCtrl.Pause()
}

func (c *Controller) resumeTempPlay() {
	if c.tmpCtrl == nil {
		return
	}
	c.state = CsTempPlay
	c.tmpCtrl.Resume()
	utils.Logger.Debug("恢复临时文件播放")
}

func (c *Controller) resumePgmPlay() {
	if c.pgmCtrl == nil {
		return
	}
	c.state = CsProgramPlay
	c.pgmCtrl.Resume()
	utils.Logger.Debug("恢复节目文件播放")
}

func (c *Controller) Start() error {
	return c.beforeProcess(func() error {
		if c.state != CsStop {
			return errors.New("广播程序已经启动")
		}
		c.state = CsRun
		c.startProgramManager()
		return nil
	})
}

func (c *Controller) Stop() error {
	if c.state == CsStop {
		return errors.New("广播程序已经停止")
	}
	c.state = CsStop
	c.manager.Stop()
	c.StopProgram()
	c.stopTempPlay(ForceOver)

	c.state = CsStop
	return nil
}

func (c *Controller) startProgramManager() {
	c.manager.Start()
	// 检查当前时间是否有节目
	if currentPgm := c.bill.GetCurrentProgram(); currentPgm != nil {
		// 当前时间有节目，则开始播放节目
		go c.StartProgram(currentPgm)
	}
}

func getPlayIndex(playOrder, preIndex, totalCount int) int {
	if playOrder == 2 {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn(totalCount - 1)
		if idx == preIndex {
			if idx++; idx >= totalCount {
				idx = 0
			}
		}
		return idx
	} else {
		return preIndex + 1
	}
}

func (c *Controller) checkState() bool {
	utils.Logger.Debugf("checkState, state: %v, wgPgmOver: %v", c.state, c.wgPgmOver)
	return c.state == CsRun || c.state == CsStop || c.wgPgmOver != nil
}

func (c *Controller) StartProgram(program *Program) {
	_ = c.beforeProcess(func() error {
		if program == nil || len(program.Files) == 0 {
			return nil
		}
		c.inProgram = true
		utils.Logger.Debugf("开始[%s~%s]的节目", program.StartTime, program.EndTime)
		fileCount := len(program.Files)

		// 如果有临时播放，则等待临时播放结束
		utils.Logger.Debug("start wait tmp over")
		c.wgTmpOver.Wait()
		utils.Logger.Debug("end wait tmp over")

		var err error
		c.state = CsProgramPlay
		wg := &sync.WaitGroup{}
		for {
			playCount := 1
			idx := -1
			for {
				idx = getPlayIndex(program.PlayOrder, idx, fileCount)
				file := program.Files[idx]
				if c.checkState() {
					goto ProgramOver
				}
				if !file.Downloading {
					utils.Logger.Debug("StartProgram enter playFile")
					if c.pgmCtrl, err = playFile(c.pgmPlayer, file, c.filePath, false, c.volume, func(reason StopReason) {
						utils.Logger.Debugf("文件[%s]播放结束", file.Name)
						wg.Done()
					}); err == nil {
						utils.Logger.Debug("StartProgram playing file")
						wg.Add(1)
						if c.checkState() {
							c.stopPgmPlay(ForceOver)
						}
						wg.Wait()
					}
					utils.Logger.Debug("StartProgram end playFile")
					time.Sleep(c.stopWaitTime)
				}
				if c.checkState() {
					goto ProgramOver
				}
				playCount++
				if playCount > fileCount {
					break
				}
			}
			if program.PlayMode == Once {
				goto ProgramOver
			}
		}
	ProgramOver:
		utils.Logger.Debugf("[%s~%s]的节目结束", program.StartTime, program.EndTime)
		c.inProgram = false
		if c.wgPgmOver != nil {
			c.wgPgmOver.Done()
		} else if c.state != CsTempPlay && c.state != CsInnerPlay {
			c.state = CsRun
		}
		return nil
	})
}

func (c *Controller) StopProgram() {
	_ = c.beforeProcess(func() error {
		if !c.inProgram {
			return nil
		}
		utils.Logger.
			WithFields(log.Fields{
				"c": "ctrl",
			}).
			Debug("In Program true")
		if c.wgPgmOver != nil {
			return errors.New("已经在结束节目的过程中")
		}

		c.wgPgmOver = &sync.WaitGroup{}
		c.wgPgmOver.Add(1)

		utils.Logger.
			WithFields(log.Fields{
				"c": "ctrl",
			}).
			Debugf("PgmCtrl is nil: %v", c.pgmCtrl == nil)
		c.stopPgmPlay(TimeOver)

		utils.Logger.
			WithFields(log.Fields{
				"c": "ctrl",
			}).
			Debug("begin wait pgm over")
		c.wgPgmOver.Wait()
		utils.Logger.
			WithFields(log.Fields{
				"c": "ctrl",
			}).
			Debug("end wait pgm over")
		c.wgPgmOver = nil
		if c.state != CsTempPlay && c.state != CsInnerPlay {
			c.state = CsRun
		}
		return nil
	})
}

func (c *Controller) PlayTemp(file *MusicFile, level, timeLen int) error {
	return c.beforeProcess(func() error {
		utils.Logger.
			WithFields(log.Fields{
				"c": "ctrl",
			}).
			Debugf("播放临时文件，%s", file.Name)
		return c.tempPlayFile(file, CsTempPlay, level, timeLen)

		//if c.pgmCtrl != nil {
		//	c.pgmCtrl.Pause()
		//}
		//if c.tmpCtrl != nil {
		//	c.tmpCtrl.Stop(ForceOver)
		//}
		//c.state = CsTempPlay
		//var err error
		//c.tmpCtrl, err = playFile(c.tmpPlayer, file, c.filePath, true, c.volume, func(reason StopReason) {
		//	if reason != ForceOver {
		//		c.tmpCtrl = nil
		//		if c.pgmCtrl != nil {
		//			time.AfterFunc(time.Millisecond, func() {
		//				c.pgmCtrl.Resume()
		//				utils.Logger.Debug("恢复节目文件播放")
		//			})
		//		}
		//	}
		//})
		//
		//if err == nil && c.tmpCtrl != nil && timeLen > 0 {
		//	ctrl := c.tmpCtrl
		//	time.AfterFunc(time.Duration(timeLen)*time.Second, func() {
		//		ctrl.Stop(TimeOver)
		//	})
		//}
		//
		//return err
	})
}

func (c *Controller) PlayInner(level, index, delay int) error {
	return c.beforeProcess(func() error {
		utils.Logger.
			WithFields(log.Fields{
				"c": "ctrl",
			}).
			Debugf("播放内置[%d]号文件", index)
		fileName := fmt.Sprintf("inner_tip%d.mp3", index)
		file := &MusicFile{
			Name:      fileName,
			PlayTimes: -1,
		}

		return c.tempPlayFile(file, CsInnerPlay, level, delay)

		//if c.pgmCtrl != nil {
		//	c.pgmCtrl.Pause()
		//}
		//if c.tmpCtrl != nil {
		//	c.tmpCtrl.Stop(ForceOver)
		//}
		//c.state = CsInnerPlay
		//var err error
		//c.tmpCtrl, err = playFile(c.tmpPlayer, file, c.filePath, true, c.volume, func(reason StopReason) {
		//	utils.Logger.Debug("播放内置文件结束")
		//	if reason != ForceOver {
		//		c.tmpCtrl = nil
		//		if c.pgmCtrl != nil {
		//			time.AfterFunc(time.Millisecond, func() {
		//				if c.pgmCtrl.Paused() {
		//					c.pgmCtrl.Resume()
		//					utils.Logger.Debug("恢复节目文件播放")
		//				}
		//			})
		//		}
		//	}
		//})
		//if err == nil && c.tmpCtrl != nil && delay > 0 {
		//	ctrl := c.tmpCtrl
		//	time.AfterFunc(time.Duration(delay)*time.Second, func() {
		//		ctrl.Stop(TimeOver)
		//	})
		//}
		//
		//return err
	})
}

func (c *Controller) tempPlayFile(file *MusicFile, state ControlState, level, timeLen int) error {
	if c.preTempLevel < level {
		return errors.New("已经有更紧急的文件在播放")
	}
	c.pausePgmPlay()
	//c.pauseTempPlay()
	c.stopTempPlay(ForceOver)
	c.state = state
	var (
		err  error
		ctrl AudioController
		tmr  *time.Timer
	)
	ctrl, err = playFile(c.tmpPlayer, file, c.filePath, true, c.volume, func(reason StopReason) {
		fileType := "临时"
		if state == CsInnerPlay {
			fileType = "内置"
		}
		utils.Logger.
			WithFields(log.Fields{
				"c": "ctrl",
			}).
			Debugf("播放%s文件结束", fileType)
		c.wgTmpOver.Done()
		if reason != ForceOver {
			c.tmpCtrl = nil
			if tmr != nil {
				tmr.Stop()
				tmr = nil
			}
			if c.pgmCtrl != nil {
				var tmrPgm *time.Timer
				tmrPgm = time.AfterFunc(time.Millisecond, func() {
					c.resumePgmPlay()
					if tmrPgm != nil {
						tmrPgm.Stop()
						tmrPgm = nil
					}
				})
			}
		}
	})
	c.preTempLevel = level

	c.tmpCtrl = ctrl
	if err == nil && c.tmpCtrl != nil {
		c.wgTmpOver.Add(1)
		if timeLen > 0 {
			tmr = time.AfterFunc(time.Duration(timeLen)*time.Second, func() {
				ctrl.Stop(TimeOver)
			})
		}
	}

	return err

}

func (c *Controller) beforeProcess(afterRun func() error) error {
	if !c.enable {
		return errors.New("广播功能已经被禁用，请先启用广播功能！")
	}
	if afterRun != nil {
		return afterRun()
	}
	return nil
}

func playFile(player AudioPlayer, file *MusicFile, filePath string, waitDownload bool, volume float64, callback func(reason StopReason)) (AudioController, error) {
	var (
		logStr   string
		fileName string
		err      error
	)
	if fileName, err = file.GetIdFileName(); err != nil {
		logStr = fmt.Sprintf("获取Id文件名出错，%s", err.Error())
		utils.Logger.Error(logStr)
		return nil, errors.New(logStr)
	}
	fileName = filePath + fileName
	utils.Logger.
		WithFields(log.Fields{
			"c": "ctrl",
		}).
		Debugf("节目文件路径：%s", fileName)
	if _, err = os.Stat(fileName); err != nil {
		if os.IsNotExist(err) {
			if file.Url == "" {
				logStr = fmt.Sprintf("内置文件[%s]不存在，%s", file.Name, err.Error())
				utils.Logger.Error(logStr)
				return nil, errors.New(logStr)
			}
			// 文件不存，则启动下载线程
			if waitDownload {
				if err = downloadFileFromUrl(file, fileName, 3); err != nil {
					logStr = fmt.Sprintf("下载文件[%s]出错，%s", file.Url, err.Error())
					utils.Logger.Error(logStr)
					return nil, errors.New(logStr)
				}
			} else {
				go func() {
					_ = downloadFileFromUrl(file, fileName, 3)
				}()
				logStr = "文件不存在，正在后台下载"
				utils.Logger.Error(logStr)
				return nil, errors.New(logStr)
			}
		} else {
			utils.Logger.Errorf("判断文件是否存在出错，%s", err.Error())
			logStr = fmt.Sprintf("判断文件是否存在出错，%s", err.Error())
			utils.Logger.Error(logStr)
			return nil, errors.New(logStr)
		}
	}

	if file.PlayTimes == 0 {
		file.PlayTimes = 1
	}

	var ctrl AudioController
	utils.Logger.
		WithFields(log.Fields{
			"c": "ctrl",
		}).
		Debugf("准备播放[%s]文件...", file.Name)
	if ctrl, err = player.Play(fileName, file.PlayTimes, volume, callback); err != nil {
		logStr = fmt.Sprintf("播放文件[%s]出错，%s", file.Name, err.Error())
		utils.Logger.Error(logStr)
		return nil, errors.New(logStr)
	}

	return ctrl, nil
}

func (c *Controller) SetEnable(enable bool) (err error) {
	if c.enable == enable {
		return nil
	}

	c.enable = enable
	if err = utils.WriteConf(utils.CfgEnable, enable); err != nil {
		return err
	}
	if enable {
		return c.Start()
	} else {
		return c.Stop()
	}
}

func (c *Controller) Reset() (err error) {
	return c.beforeProcess(func() error {
		//if c.pgmCtrl != nil {
		//	c.pgmCtrl.Stop(ForceOver)
		//}
		c.stopPgmPlay(ForceOver)
		c.manager.Stop()
		c.manager.Clear()
		c.bill = StrToBill(utils.DefaultBill)
		var strBill []byte
		if strBill, err = json.Marshal(c.bill); err != nil {
			return fmt.Errorf("转换空节目单失败，%s", err.Error())
		}
		if err = utils.WriteConf(utils.CfgBill, string(strBill)); err != nil {
			return fmt.Errorf("保存空节目单失败，%s", err.Error())
		}

		return nil
	})
}

func (c *Controller) SetVolume(volume int) error {
	return c.beforeProcess(func() error {
		if volume < 0 {
			volume = 0
		}
		if volume > 100 {
			volume = 100
		}

		fv := float64(volume / 100.0)
		if c.pgmCtrl != nil {
			c.pgmCtrl.SetVolume(fv)
		}
		if c.tmpCtrl != nil {
			c.tmpCtrl.SetVolume(fv)
		}
		return nil
	})
}

func (c *Controller) Pause() error {
	return c.beforeProcess(func() error {
		c.state = CsPause
		//if c.pgmCtrl != nil && !c.pgmCtrl.Paused() {
		//	c.pgmCtrl.Pause()
		//}
		//if c.tmpCtrl != nil && !c.tmpCtrl.Paused() {
		//	c.tmpCtrl.Pause()
		//}
		c.pausePgmPlay()
		c.pauseTempPlay()
		return nil
	})
}

func (c *Controller) Resume() error {
	return c.beforeProcess(func() error {
		if c.state != CsPause {
			return nil
		}
		if c.tmpCtrl != nil {
			c.resumeTempPlay()
		} else if c.pgmCtrl != nil {
			c.resumePgmPlay()
		} else {
			c.state = CsRun
		}
		return nil
	})
}
