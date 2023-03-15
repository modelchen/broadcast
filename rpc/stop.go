package rpc

import "Broadcast/player"

type Stop struct{}

func (p *Stop) Run(cmd *Command, controller *player.Controller) (data string, err error) {
	return "", controller.Stop()
}
