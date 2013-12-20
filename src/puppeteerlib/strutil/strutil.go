package strutil

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

func IsValidURL(url string) bool {
	if "" == url {
		return false
	}

	url = strings.ToLower(url)
	httpIdx := strings.Index(url, "http://")
	httpsIdx := strings.Index(url, "https://")

	if -1 == httpIdx && -1 == httpsIdx {
		return false
	}

	return true
}

func URL2Fingerprint(url string) string {
	hashHandle := md5.New()
	io.WriteString(hashHandle, url)
	md5Hash := fmt.Sprintf("%x", hashHandle.Sum(nil))
	ret := md5Hash + "." + strconv.Itoa(len(url))

	return ret
}

func GetRandomString(length uint16) string {
	charList := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
		"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
		"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}

	buffer := bytes.NewBufferString("")

	rand.Seed(time.Now().Unix())
	for idx := uint16(0); idx < length; idx++ {
		rndIdx := rand.Intn(len(charList))
		buffer.Write([]byte(charList[rndIdx]))
	}

	return buffer.String()
}
