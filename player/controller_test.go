package player

import (
	"Broadcast/utils"
	"fmt"
	"github.com/spf13/viper"
	"testing"
	"time"
)

func init() {
	if err := utils.InitConf("conf_d.yaml"); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("找不到配置文件..")
		} else {
			fmt.Println("配置文件出错..")
		}
	}
	utils.InitLogger("", "", "debug")
}

func TestController_downloadFileFromUrl(t *testing.T) {

	type args struct {
		file          *MusicFile
		localFilePath string
		retryTimes    int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "测试下载文件",
			args: args{
				file: &MusicFile{
					Url: "http://192.168.0.174:8088/upload/2022-07-08/20220708094449071322601722.mp3",
				},
				localFilePath: utils.GetCurrentPath() + "files/test.mp3",
				retryTimes:    3,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := downloadFileFromUrl(tt.args.file, tt.args.localFilePath, tt.args.retryTimes); (err != nil) != tt.wantErr {
				t.Errorf("downloadFileFromUrl() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewController(t *testing.T) {
	type args struct {
		billStr string
	}
	sdt1 := time.Now().Add(20 * time.Second).Format("15:04")
	edt1 := time.Now().Add(200 * time.Second).Format("15:04")
	sdt2 := time.Now().Add(300 * time.Second).Format("15:04")
	edt2 := time.Now().Add(400 * time.Second).Format("15:04")
	tests := []struct {
		name  string
		args  args
		wantC *Controller
	}{
		{
			name: "测试新建播放器",
			args: args{
				billStr: fmt.Sprintf("{\"ver\":\"20220420\",\"name\":\"测试004\",\"id\":\"1554407887143444481\",\"slot\":["+
					"{\"sdt\":\"%s\",\"edt\":\"%s\",\"files\":["+
					"{\"fId\":\"8233563356699361280\",\"fName\":\"李健 - 贝加尔湖畔.mp3\",\"fileOrd\":1,\"url\":\"http://192.168.0.174:8088/upload/2022-07-08/20220708094449071322601722.mp3\"}"+
					"],\"playOrd\":1,\"playMode\":2},"+
					"{\"sdt\":\"%s\",\"edt\":\"%s\",\"files\":["+
					"{\"fId\":\"8233562222639251456\",\"fName\":\"陈奕迅 - 浮夸.mp3\",\"fileOrd\":1,\"url\":\"http://192.168.0.174:8088/upload/2022-07-08/20220708094019033044316381.mp3\"}"+
					"],\"playOrd\":1,\"playMode\":2}"+
					"]}", sdt1, edt1, sdt2, edt2),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewController(utils.ReadConfStr(utils.CfgFilePath), tt.args.billStr, true, 5)
			c.Start()

			time.Sleep(40 * time.Second)

			fmt.Println("播放临时文件")
			_ = c.PlayInner(2, 2, 10)

			time.Sleep(5 * time.Second)
			_ = c.PlayInner(1, 3, 10)

			time.Sleep(600 * time.Second)
		})
	}
}
