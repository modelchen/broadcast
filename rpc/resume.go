package rpc

import "Broadcast/player"

type Resume struct{}

func (p *Resume) Run(cmd *Command, controller *player.Controller) (data string, err error) {
	return "", controller.Resume()
}
