package queue

import (
	"io"
	"io/ioutil"
	"os"
	ppioutil "puppeteerlib/ioutil"
	"puppeteerlib/strutil"
	"strings"
)

const (
	URL            = "URL"
	TARGET_FILE    = "TargetFile"
	LOG_FILE       = "LogFile"
	USER_AGENT     = "UserAgent"
	JOB_PREFIX_MAX = uint16(10)
	WAIT_DIR       = "wait"
	INIT_DIR       = "init"
	RUN_DIR        = "run"
)

func GetJobInitDir(queueDir string) string {
	ret := queueDir + string(os.PathSeparator) + INIT_DIR
	return ret
}

func GetJobRunDir(queueDir string) string {
	ret := queueDir + string(os.PathSeparator) + RUN_DIR
	return ret
}

func GetJobWaitDir(queueDir string) string {
	ret := queueDir + string(os.PathSeparator) + WAIT_DIR
	return ret
}

func WriteJob(queueDir string, jobInfo map[string]string) bool {
	ret := false
	initDir := GetJobInitDir(queueDir)
	waitDir := GetJobWaitDir(queueDir)

	fileHandle, err := ioutil.TempFile(initDir, "."+strutil.GetRandomString(JOB_PREFIX_MAX))
	if nil != err {
		return false
	}

	tempPath := fileHandle.Name()
	jobPath := ""

	sepIdx := strings.LastIndex(tempPath, string(os.PathSeparator))
	sepIdx += len(string(os.PathSeparator)) + 1
	jobPath = waitDir + string(os.PathSeparator) + tempPath[sepIdx:]

	hasError := false
	for jobPropName, jobPropVal := range jobInfo {
		data := jobPropName + "=" + jobPropVal + "\n"
		dataLen := len(data)

		writeLen, writeErr := io.WriteString(fileHandle, data)
		if nil != writeErr || writeLen != dataLen {
			hasError = true
			break
		}
	}

	fileHandle.Close()
	if !hasError {
		if err := os.Rename(tempPath, jobPath); nil == err {
			ret = true
		}
	} else {
		os.Remove(tempPath)
	}

	return ret
}

func ReadJob(jobFile string) map[string]string {
	jobInfo, err := ppioutil.ParseIni(jobFile)

	if nil != err {
		return nil
	}

	return jobInfo
}
