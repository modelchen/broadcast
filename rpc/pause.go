package rpc

import "Broadcast/player"

type Pause struct{}

func (p *Pause) Run(cmd *Command, controller *player.Controller) (data string, err error) {
	return "", controller.Pause()
}
