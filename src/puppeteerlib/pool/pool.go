package pool

import (
	"fmt"
	"io"
	"os"
	ppioutil "puppeteerlib/ioutil"
	ppstrutil "puppeteerlib/strutil"
)

const (
	STAT_ERR = iota
	STAT_READY
	STAT_RUNNING
	STAT_NOT_EXISTS
	SCREENSHOT_PREFIX = ".png"
	LOG_PREFIX        = ".log"
)

type ScreenshotInfo struct {
	PoolDir     string
	Fingerprint string
	Status      uint8
	LastUpdate  int64
}

func GetScreenshotInfo(poolDir string, url string) *ScreenshotInfo {
	if !ppstrutil.IsValidURL(url) {
		return nil
	}

	ret := new(ScreenshotInfo)
	ret.Fingerprint = ppstrutil.URL2Fingerprint(url)
	ret.PoolDir = fmt.Sprintf("%s%s%s%s%s", poolDir, string(os.PathSeparator), ret.Fingerprint[0:2], string(os.PathSeparator), ret.Fingerprint[2:4])
	setupScreenshotInfo(ret)

	return ret
}

func GetScreenshotInfoByFingerprint(poolDir string, fingerprint string) *ScreenshotInfo {
	if "" == fingerprint || "" == poolDir {
		return nil
	}

	ret := new(ScreenshotInfo)
	ret.Fingerprint = fingerprint
	ret.PoolDir = fmt.Sprintf("%s%s%s%s%s", poolDir, string(os.PathSeparator), ret.Fingerprint[0:2], string(os.PathSeparator), ret.Fingerprint[2:4])
	setupScreenshotInfo(ret)

	return ret
}

func setupScreenshotInfo(info *ScreenshotInfo) {
	filePath := GetScreenshotFilePath(info)
	logPath := GetScreenshotLogPath(info)

	fileInfo, err := os.Stat(filePath)
	if nil == err {
		info.Status = STAT_READY
		info.LastUpdate = fileInfo.ModTime().Unix()
	} else {
		_, err := os.Stat(logPath)

		if nil == err {
			info.Status = STAT_RUNNING
		} else {
			info.Status = STAT_NOT_EXISTS
		}
	}
}

func GetScreenshotFilePath(info *ScreenshotInfo) string {
	if "" == info.PoolDir {
		return ""
	}

	ret := info.PoolDir + string(os.PathSeparator) + info.Fingerprint + SCREENSHOT_PREFIX

	return ret
}

func GetScreenshotLogPath(info *ScreenshotInfo) string {
	if "" == info.PoolDir {
		return ""
	}

	ret := info.PoolDir + string(os.PathSeparator) + info.Fingerprint + LOG_PREFIX

	return ret
}

func AppendScreenshotLog(info *ScreenshotInfo, logToAppend string) bool {
	ret := false
	screenshotLogPath := GetScreenshotLogPath(info)

	if "" != screenshotLogPath {
		os.MkdirAll(info.PoolDir, ppioutil.DIR_MASK)
		_, err := os.Stat(info.PoolDir)

		if nil == err {
			fh, err := os.OpenFile(screenshotLogPath, os.O_CREATE|os.O_APPEND, ppioutil.FILE_MASK)
			if nil == err {
				ret = true
				io.WriteString(fh, logToAppend)
				fh.Close()
			}
		}
	}

	return ret
}
