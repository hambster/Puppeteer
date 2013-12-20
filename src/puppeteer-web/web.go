package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	ppconf "puppeteerlib/conf"
	pppool "puppeteerlib/pool"
	ppqueue "puppeteerlib/queue"
	"regexp"
	"strconv"
	"time"
)

const (
	BODY_MAX_SIZE       = 4096
	INFO_URI_PREFIX     = "/info/"
	HEADER_SIZE_DEFAULT = 1 << 20 //1M
	TIMEOUT_DEFAULT     = 60      //60 seconds
	ADDR_DEFAULT        = ""
	PORT_DEFAULT        = 8080
	POST_PARAM_URL      = "url"
	POST_PARAM_UAGENT   = "userAgent"
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

	if "GET" == req.Method {
		pathRegexp := regexp.MustCompile("^" + regexp.QuoteMeta(INFO_URI_PREFIX) + "([a-f0-9]{32}\\.[\\d]+)$")
		if matchList := pathRegexp.FindStringSubmatch(req.URL.Path); nil != matchList {
			if screenshotInfo := pppool.GetScreenshotInfoByFingerprint(gPuppeteerConf.PoolDir, matchList[1]); nil != screenshotInfo {
				apiResponse := PuppeteerWebAPIResponse{
					RetCode: 0,
					RetMsg:  "",
					Data:    PuppeteerWebAPIInfo{Key: screenshotInfo.Fingerprint, Status: screenshotInfo.Status, LastUpdate: screenshotInfo.LastUpdate}}
				jsonBytes, _ := json.Marshal(apiResponse)

				rsp.Header().Set("Content-Type", "application/json")
				io.WriteString(rsp, string(jsonBytes))
			} else {
				rsp.WriteHeader(http.StatusBadRequest)
			}
		} else {
			rsp.WriteHeader(http.StatusNotFound)
		}
	} else if "POST" == req.Method {
		targetURL := req.FormValue(POST_PARAM_URL)
		userAgent := req.FormValue(POST_PARAM_UAGENT)

		if "" != targetURL && "" != userAgent && strutil.IsValidURL(targetURL) {
			targetURL := url.QueryEscape(targetURL)
			fingerprint := strutil.URL2Fingerprint(targetURL)
			screenshotInfo := pppool.GetScreenshotByFingerprint(gPuppeteerConf.PoolDir, fingerprint)

			apiResponse := PuppeteerWebAPIResponse{}
			if nil != screenshotInfo {
				pppool.AppendScreenshotLog(screenshotInfo, "")
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
