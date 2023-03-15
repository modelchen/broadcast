package rpc

import (
	"Broadcast/player"
	"errors"
)

type SetVolume struct{}

func (p *SetVolume) Run(cmd *Command, controller *player.Controller) (data string, err error) {
	volume := cmd.Params["value"]
	if volume == nil {
		return "", errors.New("请设置音量参数，value")
	}
	iv, ok := volume.(float64)
	if !ok {
		return "", errors.New("音量参数必须是数字")
	}

	return "", controller.SetVolume(int(iv))
}
