package conf

import (
	"os"
	ppioutil "puppeteerlib/ioutil"
	ppqueue "puppeteerlib/queue"
	"strconv"
)

const (
	POOL_DIR      = "PoolDir"
	QUEUE_DIR     = "QueueDir"
	PHANTOMJS_BIN = "PhantomJSBin"
	JS            = "JS"
	MAX_PROC      = "MaxProc"
	LOG_FILE      = "LogFile"
)

type PuppeteerConf struct {
	PoolDir      string
	QueueDir     string
	PhantomJSBin string
	JS           string
	LogFile      string
	MaxProc      uint8
}

func LoadPuppeteerConf(confPath string) *PuppeteerConf {
	var ret *PuppeteerConf
	confInfo, err := ppioutil.ParseIni(confPath)

	if nil == err {
		poolDir, poolOk := confInfo[POOL_DIR]
		queueDir, queueOk := confInfo[QUEUE_DIR]
		phantomBin, binOk := confInfo[PHANTOMJS_BIN]
		js, jsOk := confInfo[JS]
		maxProcStr, procOk := confInfo[MAX_PROC]
		logFile, logOk := confInfo[LOG_FILE]

		if poolOk && queueOk && binOk && jsOk && procOk && logOk {
			if maxProc, err := strconv.ParseUint(maxProcStr, 10, 8); nil == err {
				ret = new(PuppeteerConf)
				ret.PoolDir = poolDir
				ret.QueueDir = queueDir
				ret.PhantomJSBin = phantomBin
				ret.JS = js
				ret.LogFile = logFile
				ret.MaxProc = uint8(maxProc)
			}
		}
	}

	return ret
}

func ChkPuppeteerConf(puppeteerConf *PuppeteerConf) bool {
	os.MkdirAll(puppeteerConf.PoolDir, ppioutil.DIR_MASK)
	os.MkdirAll(puppeteerConf.QueueDir, ppioutil.DIR_MASK)
	initDir := ppqueue.GetJobInitDir(puppeteerConf.QueueDir)
	runDir := ppqueue.GetJobRunDir(puppeteerConf.QueueDir)
	waitDir := ppqueue.GetJobWaitDir(puppeteerConf.QueueDir)
	os.MkdirAll(initDir, ppioutil.DIR_MASK)
	os.MkdirAll(runDir, ppioutil.DIR_MASK)
	os.MkdirAll(waitDir, ppioutil.DIR_MASK)

	if !ppioutil.IsDirExists(puppeteerConf.PoolDir) {
		return false
	}

	if !ppioutil.IsDirExists(puppeteerConf.QueueDir) {
		return false
	}

	_, err := os.Stat(puppeteerConf.PhantomJSBin)
	if nil != err {
		return false
	}

	return true
}
