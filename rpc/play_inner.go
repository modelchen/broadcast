package rpc

import (
	"Broadcast/player"
	"errors"
)

type PlayInner struct{}

func (p *PlayInner) Run(cmd *Command, controller *player.Controller) (data string, err error) {
	index := cmd.Params["index"]
	if index == nil {
		return "", errors.New("请设置内置音量序号，index")
	}
	idx, ok := index.(float64)
	if !ok {
		return "", errors.New("音量参数必须是数字")
	}

	delay := cmd.Params["delay"]
	if delay == nil {
		return "", errors.New("请设置播放延时，delay")
	}
	dly, ok := delay.(float64)
	if !ok {
		return "", errors.New("播放延时必须是数字")
	}

	level := cmd.Params["level"]
	if level == nil {
		level = "1"
	}
	lvl, ok := level.(float64)
	if !ok {
		return "", errors.New("紧急级别必须是数字")
	}

	return "", controller.PlayInner(int(lvl), int(idx), int(dly))
}
