package player

import (
	"Broadcast/utils"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// MusicFile 播放的文件信息
type MusicFile struct {
	// Name 文件名称
	Name string `json:"fName"`
	// Id 文件id
	Id string `json:"fId"`
	// PlayTimes 播放次数	默认：1，只有文本、图片可编辑次数
	PlayTimes int `json:"playTimes"`
	// Url 文件地址
	Url string `json:"url"`
	// FileOrder 文件播放顺序
	FileOrder int `json:"fileOrd"`

	// Downloading 是否在下载
	Downloading bool
}

func (f *MusicFile) GetIdFileName() (string, error) {
	if f.Url == "" {
		return f.Name, nil
	}
	index := strings.LastIndex(f.Url, ".")
	if index == -1 {
		return "", errors.New("文件url缺少后缀分割符[.]")
	}

	return f.Id + f.Url[index:], nil
}

func downloadFileFromUrl(file *MusicFile, localFilePath string, retryTimes int) (err error) {
	if file.Downloading {
		return errors.New("文件正在下载")
	}
	file.Downloading = true
	defer func() { file.Downloading = false }()
	var (
		retrys  int
		buf     = make([]byte, 32*1024)
		written int64
	)

	tmpFilePath := localFilePath + ".download"
	utils.Logger.Debugf("下载文件临时路径，%s", tmpFilePath)

	retrys = 0
	for {
		var (
			tempFile *os.File
			resp     *http.Response
		)
		retrys++
		client := new(http.Client)
		client.Timeout = time.Second * 60
		tempFile, err = os.Create(tmpFilePath)
		if err != nil {
			utils.Logger.Errorf("创建临时文件出错，%s", err.Error())
			goto downloadErr
		}
		defer tempFile.Close()

		resp, err = client.Get(file.Url)
		if err != nil {
			utils.Logger.Errorf("访问[%s]出错，%s", file.Url, err.Error())
			goto downloadErr
		}
		//_, err = strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 32)
		//if err != nil {
		//	utils.Logger.Errorf("读取文件长度出错，%s",err.Error())
		//	goto downloadErr
		//}
		if resp.Body == nil {
			utils.Logger.Error("线上文件无内容")
			break
		}
		defer resp.Body.Close()

		err = nil
		for {
			nr, er := resp.Body.Read(buf)
			if nr > 0 {
				nw, ew := tempFile.Write(buf[0:nr])
				if nw > 0 {
					written += int64(nw)
				}
				if ew != nil {
					err = ew
					break
				}
				if nr != nw {
					err = io.ErrShortWrite
					break
				}
			}
			if er != nil {
				if er != io.EOF {
					err = er
				}
				break
			}
		}
		if err != nil {
			utils.Logger.Errorf("下载写入文件出错，%s", err.Error())
			goto downloadErr
		}
		tempFile.Close()
		err = os.Rename(tmpFilePath, localFilePath)
		break
	downloadErr:
		tempFile.Close()
		os.Remove(tmpFilePath)
		if retrys >= retryTimes {
			break
		}
		time.Sleep(time.Duration(5) * time.Second)
	}

	return err
}
