package rpc

import (
	"Broadcast/player"
	"Broadcast/utils"
	"encoding/json"
	"errors"
)

type Process struct {
	command   *Command
	processor Processor
}

// NewProcess 根据传入的消息创建处理RPC对象
//
//	 process 处理对象
//		err 错误对象
func NewProcess(msg string) (process *Process, err error) {
	cmd := &Command{}
	err = json.Unmarshal([]byte(msg), cmd)
	if err != nil {
		return nil, err
	}

	utils.Logger.Debugf("rpc method: %s", cmd.Method)

	var processor Processor

	switch cmd.Method {
	case CmdSetPlayBill:
		processor = &SetPlayBill{}
	case CmdSetVolume:
		processor = &SetVolume{}
	case CmdTempPlay:
		processor = &TempPlay{}
	case CmdPlayInner:
		processor = &PlayInner{}
	case CmdStart:
		processor = &Start{}
	case CmdPause:
		processor = &Pause{}
	case CmdResume:
		processor = &Resume{}
	case CmdStop:
		processor = &Stop{}
	case CmdEnable:
		processor = &Enable{}
	case CmdReset:
		processor = &Reset{}
	default:
		return nil, errors.New("不支持的命令")
	}

	return &Process{
		command:   cmd,
		processor: processor,
	}, nil
}

// Command RPC命令
type Command struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

// Processor 命令处理接口
type Processor interface {
	Run(*Command, *player.Controller) (string, error)
}

func (p *Process) Run(controller *player.Controller) (string, error) {
	return p.processor.Run(p.command, controller)
}
