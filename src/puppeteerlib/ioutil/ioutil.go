package ioutil

import (
	"bufio"
	"bytes"
	"os"
	"strings"
)

const (
	LINE_MAX  = 8192
	DIR_MASK  = 0710
	FILE_MASK = 0644
)

func IsDirExists(dirPath string) bool {
	ret := false

	if info, err := os.Stat(dirPath); nil == err && info.IsDir() {
		ret = true
	}

	return ret
}

func ParseIni(filePath string) (map[string]string, error) {
	var ret map[string]string
	var retErr error
	buffer := bytes.NewBufferString("")
	fileInfo, retErr := os.Stat(filePath)

	if nil != retErr || fileInfo.IsDir() {
		return nil, nil
	}

	fileHandle, retErr := os.Open(filePath)

	if nil != retErr {
		return nil, nil
	}
	defer fileHandle.Close()

	reader := bufio.NewReaderSize(fileHandle, LINE_MAX)
	ret = make(map[string]string)
	for {
		line, prefixed, retErr := reader.ReadLine()

		if nil != retErr {
			break
		}

		if prefixed {
			buffer.Write(line)
			continue
		}

		var tmp string
		if 0 < buffer.Len() {
			buffer.Write(line)
			tmp = buffer.String()
			buffer.Reset()
		} else {
			tmp = string(line)
		}
		var tmpLen int
		tmpLen = len(tmp)

		if 0 == tmpLen || '#' == tmp[0] {
			continue
		}

		if equalIdx := strings.Index(tmp, "="); -1 < equalIdx {
			key := string(tmp[0:equalIdx])
			val := string("")

			if equalIdx < (tmpLen - 1) {
				bgn := equalIdx + 1
				end := tmpLen - 1
				if '"' != tmp[bgn] || '"' != tmp[end] {
					equalIdx++
					val = string(tmp[bgn:])
				} else {
					bgn++
					val = string(tmp[bgn:end])
				}
			}

			ret[key] = val
		} else {
			buffer.Reset()
		}
	}

	return ret, retErr
}
