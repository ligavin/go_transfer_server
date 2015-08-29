// go_server project server.go
package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"go_net"
	"net"
	"os"
	"strconv"
	"time"
	//	"time"
	"runtime"
	//	"server"
)

const (
	RETRY_TIMES = 20
)

type FileBuck struct {
	Start   int64
	End     int64
	No      int16
	BufFile []byte
}

var address = flag.String("address", "", "usage:ip:port")

var g_filesize int64 = 0
var g_fileBuck []FileBuck
var g_buckSize int64 = 0
var g_fileName string
var g_fileMd5 string

func main() {
	StartTime := time.Now()
	flag.Parse()

	if *address == "" {
		flag.Usage()
		return
	}
	runtime.GOMAXPROCS(runtime.NumCPU())

	addr, err := net.ResolveUDPAddr("udp", *address)
	if err != nil {
		fmt.Println("net.ResolveUDPAddr fail.", err)
		return
	}

	udpConn, err := net.DialUDP("udp", nil, addr)
	defer udpConn.Close()

	if err != nil {
		return
	}

	reqMsgPack := &go_net.MsgPack{go_net.CMD_GET_FILEINFO, nil}

	reserverBuf := make([]byte, go_net.BUFFER_LEN)

	recMsgPack, errStr := SendAndRec(udpConn, reqMsgPack, 5000, reserverBuf)

	if "" != errStr {
		fmt.Println("init: send and rec error,", errStr)
		return
	}
	udpConn.Close()

	//buf := make([]byte, go_net.BUFFER_LEN)
	respMap := map[string]string{}
	respMap = go_net.StringToMap(string(recMsgPack.Content), respMap)
	fmt.Println("resp:", string(recMsgPack.Content))

	var fileSize uint64
	fileSize, err = strconv.ParseUint(respMap["filesize"], 10, 0)
	//g_filesize, err = strconv.Atoi(respMap["filesize"])

	g_filesize = int64(fileSize)

	if nil != err {
		fmt.Println("get file size err,err", err)
		return
	}

	g_fileName = respMap["filename"]
	g_fileMd5 = respMap["md5"]

	fmt.Println("filename:", g_fileName,
		",filesize:", g_filesize,
		"md5", g_fileMd5)

	fmt.Println("the file size is:", g_filesize, ",", float64(g_filesize)/1024/1024, "M")

	fmt.Println("num of goroutine:", go_net.GetGoRoutineNum())

	DownFile(address)
	EndTime := time.Now()
	fmt.Println(EndTime.Sub(StartTime))
}

func SendAndRec(conn *net.UDPConn, sendMsgPack *go_net.MsgPack, d time.Duration,
	reserverBuf []byte) (*go_net.MsgPack, string) {
	buf := go_net.PackBuf(sendMsgPack)

	conn.SetWriteDeadline(go_net.TimeOut(d))
	wLen, err := conn.Write(*buf)

	if wLen != int(len(*buf)) || nil != err {
		errStr := "send faile,sendlen:" + strconv.Itoa(wLen) + ",err:" + err.Error()
		return nil, errStr
	}

	recBuf := reserverBuf[0:]
	conn.SetReadDeadline(go_net.TimeOut(d))
	rlen, _, err := conn.ReadFromUDP(recBuf)
	if err != nil {
		return nil, "recv data error," + err.Error()
	}

	recMsgPack, errStr := go_net.UnPackBuf(recBuf[:rlen])

	if "" != errStr {
		return nil, "unpack recvbuf fail," + errStr
	}
	return recMsgPack, ""
}

func DownFile(address *string) {
	g_buckSize = int64(go_net.GetGoRoutineNum())
	g_buckFileSize := g_filesize / g_buckSize

	for i := int64(0); i < g_buckSize; i++ {
		fileBuck := FileBuck{0, 0, 0, nil}

		fileBuck.Start = i*g_buckFileSize + 1
		fileBuck.End = fileBuck.Start + g_buckFileSize - 1
		fileBuck.No = int16(i)

		if i == g_buckSize-1 {
			fileBuck.End = g_filesize
		}

		g_fileBuck = append(g_fileBuck, fileBuck)
	}

	cFinish := make(chan int16)

	for i := int64(0); i < g_buckSize; i++ {
		go GetFile(cFinish, address, &g_fileBuck[i])
	}

	for i := int64(0); i < g_buckSize; i++ {
		no := <-cFinish
		fmt.Println(no, " buck finished")
	}

	h := md5.New()
	for i := int64(0); i < g_buckSize; i++ {
		h.Write(g_fileBuck[i].BufFile)
	}

	fmt.Println("rec file md5:", hex.EncodeToString(h.Sum(nil)))
	fmt.Println(" start write file")

	g_fileName = "_" + g_fileName
	file, err := os.OpenFile(g_fileName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)

	if err != nil {
		fmt.Println("open file error,", err)
		return
	}

	StartTime := time.Now()

	for i := int64(0); i < g_buckSize; i++ {
		wLen, err := file.Write(g_fileBuck[i].BufFile)

		if nil != err || wLen != len(g_fileBuck[i].BufFile) {
			wLen1, err := file.Write(g_fileBuck[i].BufFile[wLen:])
			if nil != err || wLen != len(g_fileBuck[i].BufFile) {
				fmt.Println("write file error:", err, ",wlen:", wLen+wLen1)
				break
			}
		}
	}

	fmt.Println("write file time", time.Now().Sub(StartTime))
	fmt.Println("the file size is:", g_filesize, ",", float64(g_filesize)/1024/1024, "M")

	fmt.Println("all finished")
}

func GetFile(cFinish chan int16, address *string, fileBuck *FileBuck) {
	addr, err := net.ResolveUDPAddr("udp", *address)
	if err != nil {
		fmt.Println("net.ResolveUDPAddr fail.", err)
		return
	}

	udpConn, err := net.DialUDP("udp", nil, addr)
	defer udpConn.Close()

	if err != nil {
		return
	}

	start := fileBuck.Start
	end := fileBuck.End

	file_len := end - start + 1

	if file_len <= 0 {
		fmt.Println("file len not valid,len:", file_len)
		return
	}

	fileBuck.BufFile = make([]byte, file_len)

	count := 0

	reserverBuf := make([]byte, go_net.BUFFER_LEN)

	for {
		if start > fileBuck.End {
			break
		}

		end := go_net.Min(start+go_net.BUFFER_CONTENT_LEN-1, fileBuck.End)
		reqMsgPack := &go_net.MsgPack{go_net.CMD_GET_FILE, nil}
		contentStr := "start=" + strconv.FormatInt(start, 10) + "&end=" +
			strconv.FormatInt(end, 10)

		//fmt.Println(contentStr)

		(*reqMsgPack).Content = []byte(contentStr)

		i := 0

		var recMsgPack *go_net.MsgPack
		var errStr string

		for ; i < RETRY_TIMES; i++ {
			recMsgPack, errStr = SendAndRec(udpConn, reqMsgPack, 2000, reserverBuf[0:])

			rlen := 0
			if recMsgPack != nil && recMsgPack.Content != nil {
				rlen = len(recMsgPack.Content)
			}
			if rlen == 0 || "" != errStr {
				fmt.Println("send and rec error,", errStr, "recv content len:", rlen)
			} else {
				break
			}
		}

		if i == RETRY_TIMES {
			fmt.Println("get file failed")
		}

		//是否会拷贝多次内存
		//fileBuck.BufFile = append(fileBuck.BufFile, recMsgPack.Content...)

		copy(fileBuck.BufFile[start-(fileBuck.Start):], recMsgPack.Content)
		//time.Sleep(5. * time.Second)
		if len(recMsgPack.Content) == 0 {
			break
		}

		start += int64(len(recMsgPack.Content))

		rlen := start - fileBuck.Start

		count++
		if count%(1024*5) == 0 {
			fmt.Println("No", fileBuck.No, " recv bufffer size:", rlen,
				",", float64(rlen)/1024/1024, " M",
				",count", count)
		}

	}

	fmt.Println("total rec len", len(fileBuck.BufFile), ",no", fileBuck.No,
		",start:", fileBuck.Start, ",end:", fileBuck.End)

	cFinish <- fileBuck.No //over
}
