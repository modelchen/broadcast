package player

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type PlayMode int

const (
	// Once 播放一次
	Once PlayMode = iota + 1
	// Cycle 循环播放
	Cycle
)

// Program 节目信息
type Program struct {
	// StartTime 开始时间	默认了00:00
	StartTime string `json:"sdt"`
	// EndTime 结束时间	默认了01:00
	EndTime string `json:"edt"`
	// StartHour 开始时间的小时部分
	StartHour int
	// StartMinute 开始时间的分钟部分
	StartMinute int
	// StartHM 开始时间的小时分钟组合值
	StartHM int
	// EndHour 结束时间的小时部分
	EndHour int
	// EndMinute 结束时间的分钟部分
	EndMinute int
	// EndHM 结束时间的小时分钟组合值
	EndHM int
	// PlayOrder 播放顺序	1：顺序播放，2：随机播放
	PlayOrder int `json:"playOrd"`
	// PlayMode 播放模式	1：单次播放，2：循环播放
	PlayMode PlayMode `json:"playMode"`
	// Files 文件列表，最多放入10个文件，文件可放：mp3、txt
	Files []*MusicFile `json:"files"`
}

// Bill 节目单
type Bill struct {
	// Id 序号
	Id string `json:"id"`
	// AppKey 所属应用键值
	AppKey string `json:"appKey"`
	// Version 版本号
	Version string `json:"ver"`
	// Name 节目名称
	Name string `json:"name"`
	// Slots 节目列表，最多设置4个时间段
	Slots []*Program `json:"slot"`
}

func (b *Bill) Check() (err error) {
	if b.Slots != nil {
		var tmp string
		var cnt = len(b.Slots) - 1
		// 先分解出开始小时和分钟数，然后判断是否符合要求
		for i := 0; i <= cnt; i++ {
			pgm := b.Slots[i]
			// 先判断开始和结束时间格式是否正确
			tms := strings.Split(pgm.StartTime, ":")
			tmp = fmt.Sprintf("开始时间[%s]格式错误，", pgm.StartTime)
			if len(tms) != 2 {
				return errors.New(tmp + "必须以 : 分隔")
			}

			if pgm.StartHour, err = strconv.Atoi(tms[0]); err != nil {
				return errors.New(tmp + "小时部分不是整数")
			}
			if pgm.StartHour < 0 || pgm.StartHour > 23 {
				return errors.New(tmp + "小时部分必须在 0~23 之间")
			}
			if pgm.StartMinute, err = strconv.Atoi(tms[1]); err != nil {
				return errors.New(tmp + "分钟部分不是整数")
			}
			if pgm.StartMinute < 0 || pgm.StartMinute > 59 {
				return errors.New(tmp + "分钟部分必须在 0~59 之间")
			}
			pgm.StartHM = pgm.StartHour*100 + pgm.StartMinute

			tmp = fmt.Sprintf("结束时间[%s]格式错误，", pgm.EndTime)
			tms = strings.Split(pgm.EndTime, ":")
			if len(tms) != 2 {
				return errors.New(tmp + "必须以 : 分隔")
			}
			if pgm.EndHour, err = strconv.Atoi(tms[0]); err != nil {
				return errors.New(tmp + "小时部分不是整数")
			}
			if pgm.EndHour < 0 || pgm.EndHour > 23 {
				return errors.New(tmp + "小时部分必须在 0~23 之间")
			}
			if pgm.EndMinute, err = strconv.Atoi(tms[1]); err != nil {
				return errors.New(tmp + "分钟部分不是整数")
			}
			if pgm.EndMinute < 0 || pgm.EndMinute > 59 {
				return errors.New(tmp + "分钟部分必须在 0~59 之间")
			}
			pgm.EndHM = pgm.EndHour*100 + pgm.EndMinute

			// 开始和结束时间不能一样
			if pgm.StartHM == pgm.EndHM {
				return errors.New("开始和结束时间不能相同")
			}

			// 检查文件Url格式
			for j := 0; j < len(pgm.Files); j++ {
				file := pgm.Files[j]
				file.Downloading = false
				if _, err = file.GetIdFileName(); err != nil {
					return err
				}
			}
		}
		if len(b.Slots) > 1 {
			// 按开始时间排序
			b.SortByStartTime()

			if err = b.CheckHaveOverlap(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *Bill) SortByStartTime() {
	cnt := len(b.Slots)
	for i := 0; i < cnt-1; i++ {
		for j := 0; j < cnt-1-i; j++ {
			pgm1 := b.Slots[j]
			pgm2 := b.Slots[j+1]
			if pgm1.StartHM > pgm2.StartHM {
				b.Slots[j] = pgm2
				b.Slots[j+1] = pgm1
			}
		}
	}
}

func (b *Bill) CheckHaveOverlap() error {
	cnt := len(b.Slots)
	if cnt < 2 {
		return nil
	}

	cnt--
	// 检查到倒数第二个时段
	for i := 0; i < cnt; i++ {
		pgmCurrent := b.Slots[i]
		// 先判断自己是否开始和结束时间颠倒
		if pgmCurrent.StartHM >= pgmCurrent.EndHM {
			return errors.New("只有最后一组时段，开始时间可以大于结束时间")
		}
		pgmNext := b.Slots[i+1]
		if pgmCurrent.EndHM >= pgmNext.StartHM {
			return fmt.Errorf("时段[%s~%s]与[%s~%s]有重叠", pgmCurrent.StartTime, pgmCurrent.EndTime, pgmNext.StartTime, pgmNext.EndTime)
		}
	}

	// 最后一个时段单独判断
	pgmLast := b.Slots[cnt]
	if pgmLast.EndHM < pgmLast.StartHM { // 结束时间比开始时间还小，说明是跨天时段
		pgmFist := b.Slots[0]
		if pgmLast.EndHM >= pgmFist.StartHM { // 跨天时段，需要判断是否和第一时段发生重叠
			return errors.New("最后一个时段的结束时间可以比开始时间小（相当于跨天），但它不能比第一个时段的开始时间还大")
		}
	}

	return nil
}

func (b *Bill) GetCurrentProgram() *Program {
	lastIdx := len(b.Slots) - 1
	if lastIdx < 0 {
		return nil
	}
	currentTime := time.Now()
	currentHM := currentTime.Hour()*100 + currentTime.Minute()

	for i := 0; i <= lastIdx; i++ {
		pgm := b.Slots[i]
		if pgm.StartHM <= currentHM && currentHM <= pgm.EndHM {
			return pgm
		}
	}

	// 最后一个时段单独判断
	pgmLast := b.Slots[lastIdx]
	if pgmLast.EndHM < pgmLast.StartHM { // 结束时间比开始时间还小，说明是跨天时段
		if (pgmLast.StartHM <= currentHM && currentHM <= 2359) || (0 <= currentHM && currentHM <= pgmLast.EndHM) {
			return pgmLast
		}
	}

	return nil
}
