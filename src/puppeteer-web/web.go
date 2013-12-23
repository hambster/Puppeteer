package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	ppconf "puppeteerlib/conf"
	ppioutil "puppeteerlib/ioutil"
	pppool "puppeteerlib/pool"
	ppqueue "puppeteerlib/queue"
	ppstrutil "puppeteerlib/strutil"
	"regexp"
	"strconv"
	"time"
)

const (
	BODY_MAX_SIZE       = 4096
	INFO_URI_PREFIX     = "/info/"
	PIC_URI_PREFIX      = "/pic/"
	HEADER_SIZE_DEFAULT = 1 << 20 //1M
	TIMEOUT_DEFAULT     = 60      //60 seconds
	ADDR_DEFAULT        = ""
	PORT_DEFAULT        = 8080
	POST_PARAM_URL      = "url"
	POST_PARAM_UAGENT   = "userAgent"
)

const (
	API_RET_ERR_IO     = -1
	API_RET_OK         = 0
	API_RET_ERR_IO_MSG = "io error"
	API_RET_OK_MSG     = ""
)

type PuppeteerWebAPIResponse struct {
	RetCode int
	RetMsg  string
	Data    interface{}
}

type PuppeteerWebAPIInfo struct {
	Key        string
	Status     uint8
	LastUpdate int64
}

type PuppeteerWebHandler struct {
	http.Handler
}

var gPuppeteerConf *ppconf.PuppeteerConf

func (this PuppeteerWebHandler) ServeHTTP(rsp http.ResponseWriter, req *http.Request) {
	if nil != req.Body {
		req.Body = http.MaxBytesReader(rsp, req.Body, BODY_MAX_SIZE)
		err := req.ParseMultipartForm(BODY_MAX_SIZE)

		if nil != err {
			rsp.WriteHeader(http.StatusRequestURITooLong)
			return
		}
	}

	pathRegexp := regexp.MustCompile("^(\\/[a-zA-Z0-9\\-\\_]+\\/)([a-f0-9]{32}\\.[\\d]+)$")
	if "GET" == req.Method {
		if matchList := pathRegexp.FindStringSubmatch(req.URL.Path); nil != matchList {
			switch matchList[1] {
			case INFO_URI_PREFIX:
				if screenshotInfo := pppool.GetScreenshotInfoByFingerprint(gPuppeteerConf.PoolDir, matchList[2]); nil != screenshotInfo {
					apiResponse := PuppeteerWebAPIResponse{
						RetCode: API_RET_OK,
						RetMsg:  "",
						Data:    PuppeteerWebAPIInfo{Key: screenshotInfo.Fingerprint, Status: screenshotInfo.Status, LastUpdate: screenshotInfo.LastUpdate}}
					jsonBytes, _ := json.Marshal(apiResponse)

					rsp.Header().Set("Content-Type", "application/json")
					io.WriteString(rsp, string(jsonBytes))
				} else {
					rsp.WriteHeader(http.StatusBadRequest)
				}
				break
			case PIC_URI_PREFIX:
				if screenshotInfo := pppool.GetScreenshotInfoByFingerprint(gPuppeteerConf.PoolDir, matchList[2]); nil != screenshotInfo {
					if pppool.STAT_READY == screenshotInfo.Status {
						filePath := pppool.GetScreenshotFilePath(screenshotInfo)
						if fh, openErr := os.OpenFile(filePath, os.O_RDONLY, ppioutil.FILE_MASK); nil == openErr {
							rsp.Header().Set("Content-Type", "image/png")
							rsp.Header().Set("Content-Disposition", "inline; filename=screenshot.png")
							io.Copy(rsp, fh)
							fh.Close()
						} else {
							rsp.WriteHeader(http.StatusNotFound)
						}
					} else {
						rsp.WriteHeader(http.StatusNotFound)

					}
				} else {
					rsp.WriteHeader(http.StatusBadRequest)
				}
				break
			default:
				rsp.WriteHeader(http.StatusNotFound)
				break
			}
		} else {
			rsp.WriteHeader(http.StatusBadRequest)
		}
	} else if "POST" == req.Method {
		targetURL := req.FormValue(POST_PARAM_URL)
		userAgent := req.FormValue(POST_PARAM_UAGENT)

		if req.URL.Path == INFO_URI_PREFIX && "" != targetURL && "" != userAgent && ppstrutil.IsValidURL(targetURL) {
			fingerprint := ppstrutil.URL2Fingerprint(targetURL)
			screenshotInfo := pppool.GetScreenshotInfoByFingerprint(gPuppeteerConf.PoolDir, fingerprint)

			apiResponse := PuppeteerWebAPIResponse{}
			if nil != screenshotInfo {
				pppool.AppendScreenshotLog(screenshotInfo, fmt.Sprintf("%d\t%s\n", time.Now().Unix(), targetURL))
				jobData := map[string]string{ppqueue.URL: targetURL,
					ppqueue.TARGET_FILE: pppool.GetScreenshotFilePath(screenshotInfo),
					ppqueue.LOG_FILE:    pppool.GetScreenshotLogPath(screenshotInfo),
					ppqueue.USER_AGENT:  userAgent}
				if ppqueue.WriteJob(gPuppeteerConf.QueueDir, jobData) {
					apiResponse.RetCode = API_RET_OK
					apiResponse.Data = PuppeteerWebAPIInfo{Key: screenshotInfo.Fingerprint, Status: pppool.STAT_RUNNING, LastUpdate: 0}
				} else {
					apiResponse.RetCode = API_RET_ERR_IO
					apiResponse.RetMsg = API_RET_ERR_IO_MSG
					apiResponse.Data = PuppeteerWebAPIInfo{Key: screenshotInfo.Fingerprint, Status: pppool.STAT_RUNNING, LastUpdate: 0}
				}
			}
			jsonBytes, _ := json.Marshal(apiResponse)

			rsp.Header().Set("Content-Type", "application/json")
			io.WriteString(rsp, string(jsonBytes))
		} else {
			rsp.WriteHeader(http.StatusBadRequest)
		}
	}
}

func main() {
	if 2 > len(os.Args) {
		Usage()
	}

	conf, port, timeout, addr := GetCmdArg()

	if nil == conf {
		Usage()
	}

	gPuppeteerConf = conf
	if logHandle, logErr := os.OpenFile(gPuppeteerConf.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, ppioutil.FILE_MASK); nil == logErr {
		log.SetOutput(logHandle)
		log.SetFlags(log.LstdFlags)
	}
	puppeteerHandler := PuppeteerWebHandler{}

	srv := &http.Server{
		Addr:           addr + ":" + strconv.Itoa(port),
		Handler:        puppeteerHandler,
		ReadTimeout:    time.Duration(timeout) * time.Second,
		WriteTimeout:   time.Duration(timeout) * time.Second,
		MaxHeaderBytes: HEADER_SIZE_DEFAULT,
	}

	srv.ListenAndServe()
}

func GetCmdArg() (*ppconf.PuppeteerConf, int, int, string) {
	if 2 > len(os.Args) {
		return nil, 0, 0, ""
	}

	conf := ppconf.LoadPuppeteerConf(os.Args[1])
	port := PORT_DEFAULT
	timeout := TIMEOUT_DEFAULT
	addr := ADDR_DEFAULT

	switch len(os.Args) {
	case 5:
		addr = os.Args[4]
		fallthrough
	case 4:
		if tmp, err := strconv.ParseInt(os.Args[3], 10, 16); nil == err && 0 < tmp {
			timeout = int(tmp)
		}
		fallthrough
	case 3:
		if tmp, err := strconv.ParseInt(os.Args[2], 10, 16); nil == err && 0 < tmp && 65535 >= tmp {
			port = int(tmp)
		}
		break
	}

	return conf, port, timeout, addr
}

func Usage() {
	usageStrFmt := `
    %s conf [port] [timeout] [addr]

    conf: puppeteer configuration. required.
    port: port to listen. optional. default 8080.
    timeout: request/response timeout. optional. default 60.
    addr: address to listen. optional. default all.
`
	usageStr := fmt.Sprintf(usageStrFmt, os.Args[0])
	fmt.Print(usageStr)
	os.Exit(1)
}
