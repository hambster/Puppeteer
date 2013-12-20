package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	ppconf "puppeteerlib/conf"
	ppqueue "puppeteerlib/queue"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	OPT_MAX_PROC      = "max-proc"
	OPT_POOL_DIR      = "pool-dir"
	OPT_QUEUE_DIR     = "queue-dir"
	OPT_PHJS_BIN      = "phantomjs-bin"
	OPT_JS            = "js"
	MAX_PROC_DEFAULT  = "5"
	POOL_DIR_DEFAULT  = "/puppeteer/pool"
	QUEUE_DIR_DEFAULT = "/puppeteer/queue"
)

type PuppeteerConf struct {
	PoolDir      string
	QueueDir     string
	PhantomJSBin string
	JS           string
	MaxProc      uint8
}

type Scoreboard struct {
	Conf      *ppconf.PuppeteerConf
	Lock      *sync.RWMutex
	procCnt   uint8
	terminate bool
}

func (this *Scoreboard) IsTerminated() bool {
	this.Lock.RLock()
	ret := this.terminate
	this.Lock.RUnlock()

	return ret
}

func (this *Scoreboard) Terminate() {
	this.Lock.Lock()
	this.terminate = true
	this.Lock.Unlock()
}

func (this *Scoreboard) GetProcCnt() uint8 {
	ret := uint8(0)
	this.Lock.RLock()
	ret = this.procCnt
	this.Lock.RUnlock()

	return ret
}

func (this *Scoreboard) IncrProcCnt() bool {
	ret := false
	this.Lock.Lock()
	if this.Conf.MaxProc > this.procCnt {
		this.procCnt++
		ret = true
	}
	this.Lock.Unlock()

	return ret
}

func (this *Scoreboard) DecrProcCnt() {
	this.Lock.Lock()
	if 0 < this.procCnt {
		this.procCnt--
	}
	this.Lock.Unlock()
}

func NewScoreboard(conf *ppconf.PuppeteerConf) *Scoreboard {
	ret := new(Scoreboard)
	ret.Conf = conf
	ret.Lock = new(sync.RWMutex)
	ret.procCnt = 0

	return ret
}

func JobMaster(queueChannel chan string, scoreboard *Scoreboard) {
	scoreboard.IncrProcCnt()
	procCnt := scoreboard.GetProcCnt()

	scoreboard.Lock.RLock()
	queueDir := scoreboard.Conf.QueueDir
	maxProc := scoreboard.Conf.MaxProc
	scoreboard.Lock.RUnlock()

	for idx := procCnt; idx < maxProc; idx++ {
		go JobSlave(queueChannel, scoreboard)
	}

	for {
		if scoreboard.IsTerminated() {
			break
		}

		waitDir := ppqueue.GetJobWaitDir(queueDir)
		if dirHandle, err := os.Open(waitDir); nil != err {
			for {
				fileList, err := dirHandle.Readdir(1)

				if nil != err || 0 >= len(fileList) {
					break
				}

				queueFile := waitDir + string(os.PathSeparator) + fileList[0].Name()
				queueChannel <- queueFile
			}
			dirHandle.Close()
		}
		time.Sleep(time.Second)
	}

	close(queueChannel)
	scoreboard.DecrProcCnt()
}

func JobSlave(queueChannel chan string, scoreboard *Scoreboard) {
	if !scoreboard.IncrProcCnt() {
		return
	}

	scoreboard.Lock.RLock()
	queueDir := scoreboard.Conf.QueueDir
	phantomJSBin := scoreboard.Conf.PhantomJSBin
	jsPath := scoreboard.Conf.JS
	scoreboard.Lock.RUnlock()

	t := time.NewTimer(time.Second)
	for {
		if scoreboard.IsTerminated() {
			break
		}

		select {
		case queueFile, queueValid := <-queueChannel:
			if !queueValid {
				break
			}
			if sepIdx := strings.LastIndex(queueFile, string(os.PathSeparator)); -1 != sepIdx {
				runDir := ppqueue.GetJobRunDir(queueDir)
				queueFileName := string(queueFile[sepIdx+1:])
				runFile := runDir + string(os.PathSeparator) + queueFileName

				if err := os.Rename(queueFile, runFile); nil == err {
					if jobInfo := ppqueue.ReadJob(runFile); nil != jobInfo {
						if sepIdx := strings.LastIndex(jobInfo[ppqueue.TARGET_FILE], string(os.PathSeparator)); -1 != sepIdx {

						}

						cmd := exec.Command(phantomJSBin, jsPath, jobInfo[ppqueue.URL], jobInfo[ppqueue.TARGET_FILE], jobInfo[ppqueue.LOG_FILE], jobInfo[ppqueue.USER_AGENT])
						cmd.Run()
					}

					os.Remove(runFile)
				}
			}
		case <-t.C:
		}
		time.Sleep(time.Second)
	}

	scoreboard.DecrProcCnt()
}

func main() {
	cmdMap := GetCmdArg()
	puppeteerConf := GetPuppeteerConf(cmdMap)
	if !ppconf.ChkPuppeteerConf(puppeteerConf) {
		Usage()
	}

	queueChannel := make(chan string, 1)
	scoreboard := NewScoreboard(puppeteerConf)

	go JobMaster(queueChannel, scoreboard)
	time.Sleep(time.Second)

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Kill, syscall.SIGHUP, syscall.SIGTERM)

	for {
		select {
		case inSignal := <-signalChannel:
			if syscall.SIGTERM == inSignal || syscall.SIGKILL == inSignal {
				scoreboard.Terminate()
			}
			break
		default:
		}

		if scoreboard.IsTerminated() {
			if 0 == scoreboard.GetProcCnt() {
				break
			}
		}
		time.Sleep(time.Second)
	}

	os.Exit(0)
}

func Usage() {
	usageStrFmt := `
    %s [-max-proc=$numProc] [-pool-dir=$poolDir] [-queue-dir=$queueDir] [-phantomjs-bin=$phantomjsPath] [-js=$jsPath]

    -max-proc: number of maximum process to take url screenshot. default 5.
    -pool-dir: directory to store screenshot files. default "/puppeteer/pool"
    -queue-dir: directory to grab job files. default "/puppeteer/queue"
    -phantomjs-bin: binary file of phantomjs.
    -js: js script to run.

`
	usageStr := fmt.Sprintf(usageStrFmt, os.Args[0])
	fmt.Print(usageStr)
	os.Exit(1)
}

func GetPuppeteerConf(cmdMap map[string]string) *ppconf.PuppeteerConf {
	ret := new(ppconf.PuppeteerConf)
	ret.MaxProc = uint8(5)
	ret.PoolDir = POOL_DIR_DEFAULT

	if arg, ok := cmdMap[OPT_POOL_DIR]; ok {
		ret.PoolDir = arg
	}

	if arg, ok := cmdMap[OPT_QUEUE_DIR]; ok {
		ret.QueueDir = arg
	}

	if arg, ok := cmdMap[OPT_MAX_PROC]; ok {
		if numVal, err := strconv.ParseUint(arg, 10, 8); nil == err {
			ret.MaxProc = uint8(numVal)
		}
	}

	if arg, ok := cmdMap[OPT_PHJS_BIN]; ok && "" != cmdMap[OPT_PHJS_BIN] {
		ret.PhantomJSBin = arg
	}

	if arg, ok := cmdMap[OPT_JS]; ok && "" != cmdMap[OPT_JS] {
		ret.JS = arg
	}

	return ret
}

func GetCmdArg() map[string]string {
	ret := map[string]string{OPT_MAX_PROC: MAX_PROC_DEFAULT, OPT_POOL_DIR: POOL_DIR_DEFAULT, OPT_QUEUE_DIR: QUEUE_DIR_DEFAULT, OPT_PHJS_BIN: "", OPT_JS: ""}

	var equalIdx int
	for idx := int(0); idx < len(os.Args); idx++ {
		if '-' == os.Args[idx][0] {
			equalIdx = strings.Index(os.Args[idx], "=")
			if _, ok := ret[string(os.Args[idx][1:equalIdx])]; ok && -1 != equalIdx {
				ret[string(os.Args[idx][1:equalIdx])] = string(os.Args[idx][equalIdx+1:])
			}
		}
	}

	return ret
}
