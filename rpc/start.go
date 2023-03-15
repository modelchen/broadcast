package rpc

import "Broadcast/player"

type Start struct{}

func (p *Start) Run(cmd *Command, controller *player.Controller) (data string, err error) {
	return "", controller.Start()
}
