package rpc

import (
	"Broadcast/player"
	"errors"
)

type Enable struct{}

func (p *Enable) Run(cmd *Command, controller *player.Controller) (data string, err error) {
	val := cmd.Params["value"]
	if val == nil {
		return "", errors.New("请设置使能参数，value")
	}
	iv, ok := val.(float64)
	if !ok {
		return "", errors.New("使能参数必须是数字")
	}

	return data, controller.SetEnable(iv >= 1)
}
