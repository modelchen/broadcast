package rpc

import (
	"Broadcast/player"
	"errors"
)

type TempPlay struct{}

func (p *TempPlay) Run(cmd *Command, controller *player.Controller) (data string, err error) {

	file := &player.MusicFile{}
	var tmp interface{}
	tmp = cmd.Params["fId"]
	if tmp == nil {
		return "", errors.New("请设置文件序号，fId")
	}
	file.Id = tmp.(string)

	tmp = cmd.Params["fName"]
	if tmp == nil {
		return "", errors.New("请设置文件名，fName")
	}
	file.Name = tmp.(string)

	tmp = cmd.Params["url"]
	if tmp == nil {
		return "", errors.New("请设置文件下载地址，url")
	}
	file.Url = tmp.(string)

	tmp = cmd.Params["playTimes"]
	if tmp == nil {
		file.PlayTimes = 1
	} else {
		file.PlayTimes = int(tmp.(float64))
	}

	timeLen := cmd.Params["timeLen"]
	if timeLen == nil {
		timeLen = 0
	}
	tl, ok := timeLen.(float64)
	if !ok {
		return "", errors.New("播放时长必须是整数")
	}

	level := cmd.Params["level"]
	if level == nil {
		level = "1"
	}
	lvl, ok := level.(float64)
	if !ok {
		return "", errors.New("紧急级别必须是数字")
	}

	return "", controller.PlayTemp(file, int(lvl), int(tl))

}
