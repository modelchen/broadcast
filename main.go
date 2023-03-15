package main

import (
	"Broadcast/mqtt"
	"Broadcast/player"
	"Broadcast/utils"
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"time"

	//_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	//go func() {
	//	http.ListenAndServe("192.168.10.250:10000", nil)
	//}()
	execPath := utils.GetCurrentPath()
	fmt.Printf("exec path: %s\r\n", execPath)

	lockFile, err := utils.CreateLockFile(execPath + "Single.lock")
	if err != nil {
		fmt.Printf("创建单例运行锁定文件失败[%s]，说明已经有程序在运行，退出！\r\n", err.Error())
		_ = lockFile.Close()
		return
	}

	var configName string
	flag.StringVar(&configName, "conf", "config.yaml", "配置文件，默认 config.yaml")
	flag.Parse()
	fmt.Printf("config: %s\n", configName)
	if err = utils.InitConf(configName); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("找不到配置文件..")
		} else {
			fmt.Println("配置文件出错..")
		}
	}

	utils.InitLogger(execPath+"logs/", utils.ReadConfStr("log.name"), utils.ReadConfStrOrDef("log.level", "info"))

	//ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	var wgCancel sync.WaitGroup
	wgCancel.Add(1)

	signChan := make(chan os.Signal)
	signal.Notify(signChan, syscall.SIGINT, syscall.SIGTERM) //nolint:govet
	go func() {
		force := false
		for {
			osCall := <-signChan
			if force {
				os.Exit(1)
			}
			utils.Logger.Infof("收到系统调用: %v", osCall)
			wgCancel.Done()
			force = true
			utils.Logger.Infof("请再次按 ctrl+c 强制退出")
		}
	}()

	tasks := make([]func(controller *player.Controller, wg *sync.WaitGroup, wgCancel *sync.WaitGroup), 0)
	tasks = append(tasks, startMqttClient)
	tasks = append(tasks, startBroadcast)

	controller := player.NewController(
		utils.ReadConfStr(utils.CfgFilePath),
		utils.ReadConfStrOrDef(utils.CfgBill, utils.DefaultBill),
		utils.ReadConfBool(utils.CfgEnable),
		time.Duration(utils.ReadConfIntOrDef(utils.CfgStopWaitTime, 900)),
	)
	wg.Add(len(tasks))
	for _, task := range tasks {
		go task(controller, &wg, &wgCancel)
	}

	utils.Logger.Info("广播系统开始运行...")
	wg.Wait()
	_ = lockFile.Close()
	utils.Logger.Info("广播系统停止")
}

func startMqttClient(controller *player.Controller, wg *sync.WaitGroup, wgCancel *sync.WaitGroup) {
	host := utils.ReadConfStrOrDef(utils.CfgHost, utils.Host)
	userName := utils.ReadConfStrOrDef(utils.CfgUserName, utils.UserName)
	password := utils.ReadConfStrOrDef(utils.CfgPassword, utils.Password)
	clientId := utils.ReadConfStrOrDef(utils.CfgClientId, utils.ClientId)
	client := mqtt.NewBroadcastClient(controller)
	go client.StartListener(host, userName, password, clientId)

	defer wg.Done()

	wgCancel.Wait()
	client.StopListener()

	//for {
	//	select {
	//	case <-ctx.Done():
	//		client.StopListener()
	//		return
	//	}
	//}
}

func startBroadcast(controller *player.Controller, wg *sync.WaitGroup, wgCancel *sync.WaitGroup) {
	go controller.Start()

	defer wg.Done()

	wgCancel.Wait()
	_ = controller.Stop()

	//for {
	//	select {
	//	case <-ctx.Done():
	//		_ = controller.Stop()
	//		return
	//	}
	//}
}
