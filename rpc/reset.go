package rpc

import "Broadcast/player"

type Reset struct{}

func (p *Reset) Run(cmd *Command, controller *player.Controller) (data string, err error) {
	return "", controller.Reset()
}
