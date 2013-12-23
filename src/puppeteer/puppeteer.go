package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	ppconf "puppeteerlib/conf"
	ppioutil "puppeteerlib/ioutil"
	ppqueue "puppeteerlib/queue"
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
	ret.terminate = false

	return ret
}

func JobMaster(queueChannel chan string, scoreboard *Scoreboard) {
	scoreboard.IncrProcCnt()
	procCnt := scoreboard.GetProcCnt()

	scoreboard.Lock.RLock()
	queueDir := scoreboard.Conf.QueueDir
	maxProc := scoreboard.Conf.MaxProc
	scoreboard.Lock.RUnlock()

	log.Printf("job master starts")
	for idx := procCnt; idx < maxProc; idx++ {
		go JobSlave(queueChannel, scoreboard)
	}

	for {
		if scoreboard.IsTerminated() {
			break
		}

		waitDir := ppqueue.GetJobWaitDir(queueDir)
		if dirHandle, err := os.Open(waitDir); nil == err {
			for {
				fileList, err := dirHandle.Readdir(1)

				if nil != err || 0 >= len(fileList) {
					break
				}

				queueFile := waitDir + string(os.PathSeparator) + fileList[0].Name()
				queueChannel <- queueFile
			}
			dirHandle.Close()
		} else {
			log.Printf("open queue dir error - %s", err)
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

	log.Printf("job slave starts")
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
						if _, statErr := os.Stat(jobInfo[ppqueue.TARGET_FILE]); nil != statErr && os.IsNotExist(statErr) {
							log.Printf("process job %s for %s\n", runFile, jobInfo[ppqueue.TARGET_FILE])
							cmd := exec.Command(phantomJSBin, jsPath, jobInfo[ppqueue.URL], jobInfo[ppqueue.TARGET_FILE], jobInfo[ppqueue.LOG_FILE], jobInfo[ppqueue.USER_AGENT])
							if err := cmd.Run(); nil != err {
								log.Printf("process job err - %s", err.Error())
							}
						}
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
	if 2 > len(os.Args) {
		Usage()
	}

	puppeteerConf := ppconf.LoadPuppeteerConf(os.Args[1])
	if !ppconf.ChkPuppeteerConf(puppeteerConf) {
		Usage()
	}

	if logFH, logErr := os.OpenFile(puppeteerConf.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, ppioutil.FILE_MASK); nil == logErr {
		log.SetOutput(logFH)
		log.SetFlags(log.LstdFlags)
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
    %s [puppeteer.conf]

    puppeteer.conf: configuration of puppeteer.
`
	usageStr := fmt.Sprintf(usageStrFmt, os.Args[0])
	fmt.Print(usageStr)
	os.Exit(1)
}
