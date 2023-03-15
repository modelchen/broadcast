package rpc

import (
	"Broadcast/player"
	"encoding/json"
	"fmt"
)

type SetPlayBill struct{}

func (p *SetPlayBill) Run(cmd *Command, controller *player.Controller) (data string, err error) {
	var tmp []byte
	if tmp, err = json.Marshal(cmd.Params); err != nil {
		return "", fmt.Errorf("转换JSON参数出错，%s", err.Error())
	}

	return "", controller.SetBill(player.BytesToBill(tmp), true)
}
