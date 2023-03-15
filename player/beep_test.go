package player

import (
	"Broadcast/utils"
	"fmt"
	"testing"
	"time"
)

func TestBeepAudioPlayer_Play(t *testing.T) {
	b1 := &BeepAudioPlayer{}
	b2 := &BeepAudioPlayer{}
	f1 := utils.GetCurrentPath() + "files/李健.mp3"
	c1, err1 := b1.Play(f1, 1, 1, func(reason StopReason) {
		fmt.Printf("p1 stop: %v", reason)
	})
	if err1 != nil {
		fmt.Printf("Play1 play error = %v\r\n", err1)
		return
	}
	fmt.Println("begin play p1")
	time.Sleep(10 * time.Second)
	c1.Pause()
	fmt.Println("p1 pause")

	f2 := utils.GetCurrentPath() + "files/inner_tip2.mp3"
	c2, err2 := b2.Play(f2, -1, 0.5, func(reason StopReason) {
		fmt.Printf("p2 stop: %v", reason)
	})
	if err2 != nil {
		fmt.Printf("Play1 play error = %v\r\n", err2)
		return
	}

	fmt.Println("begin play p2")
	time.Sleep(10 * time.Second)
	c2.Pause()
	fmt.Println("p2 pause")

	fmt.Println("p1 Resume")
	c1.Resume()
	time.Sleep(10 * time.Second)
	c1.Pause()
	fmt.Println("p1 pause")

	fmt.Println("p2 Resume")
	c2.Resume()
	time.Sleep(10 * time.Second)
	c2.Stop(ForceOver)

	fmt.Println("p1 Resume")
	c1.Resume()
	time.Sleep(10 * time.Second)

	c1.Stop(ForceOver)
}
