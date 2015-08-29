package server

import (
	//	"runtime"
	//	"bytes"
	"fmt"
	"go_net"
	"net"
	"os"
	//	"runtime"
	//	"strings"
	//	"time"
	//	"runtime"
	//	"runtime/debug"
	"strconv"
)

var g_fileInfo os.FileInfo
var g_file *os.File

var g_fileBuffer []byte
var g_fileMd5 string

type ChanBuffer struct {
	realBuf    []byte
	buf        []byte
	conn       *net.UDPConn
	remoteAddr *net.UDPAddr
}

func Run(filename *string, conn *net.UDPConn) {
	var err error
	g_fileInfo, err = os.Stat(*filename)

	if nil != err {
		fmt.Println("get file info error,", filename, ",", err)
		return
	}

	g_file, err = os.OpenFile(*filename, os.O_RDONLY, 0400)

	if nil != err {
		fmt.Println("open file error,", err)
		return
	}

	g_fileBuffer = make([]byte, g_fileInfo.Size())
	rlen, err := g_file.ReadAt(g_fileBuffer, 0)

	if int64(rlen) != g_fileInfo.Size() || err != nil {
		fmt.Println("read file err:", err, ",rlen:", rlen)
		return
	}

	fmt.Println("file size:", g_fileInfo.Size(), " byte", ",", float64(g_fileInfo.Size())/1024/1024, "M")
	fmt.Println("start cal file md5")
	g_fileMd5 = go_net.Md5(g_fileBuffer)
	fmt.Println("calc finished.file md5:", g_fileMd5)

	fmt.Println("server ready, server is running")

	reqChan := make(chan *ChanBuffer)
	bufChan := make(chan *ChanBuffer, go_net.GetGoRoutineNum())
	buf := make([]byte, go_net.BUFFER_LEN)

	for i := int32(0); i < go_net.GetGoRoutineNum(); i++ {
		chanBuf := &ChanBuffer{}
		//底层buf
		chanBuf.realBuf = make([]byte, go_net.BUFFER_LEN)
		bufChan <- chanBuf
		go Response(reqChan, bufChan)
	}

	for {

		readLen, remoteAddr, err := conn.ReadFromUDP(buf)
		if nil != err {
			fmt.Println("read error:", err)
			continue
		}

		realBuf := buf[0:readLen]
		//fmt.Println("\nget a new request:", remoteAddr.String(), ",readlen:", readLen)

		if readLen > go_net.PACKAGE_CMD {
			//fmt.Println("recv content:", string(realBuf[go_net.PACKAGE_CMD:]))
		} else {
			//fmt.Println("recv content null")
		}

		chanBuf := <-bufChan
		chanBuf.buf = chanBuf.realBuf[0:readLen]
		chanBuf.conn = conn
		chanBuf.remoteAddr = remoteAddr

		cpLen := copy(chanBuf.buf, realBuf)

		if cpLen != readLen {
			fmt.Println("cp error,cpLen:", cpLen, ",realLen:", readLen)
			continue
		}
		/*for i := 0; i < 1000000; i++ {
			reqChan <- chanBuf
		}*/
		reqChan <- chanBuf
	}
}

func Response(reqChan chan *ChanBuffer, bufChan chan *ChanBuffer) {

	var msgPack *go_net.MsgPack
	var errStr string
	buf := make([]byte, go_net.BUFFER_LEN)
	fileInfoStr := "filesize=" + strconv.FormatInt(g_fileInfo.Size(), 10) +
		"&filename=" + g_fileInfo.Name() + "&md5=" + g_fileMd5

	infoByte := []byte(fileInfoStr)
	respMsgPack := &go_net.MsgPack{-1, nil}
	tmpBuf := make([]byte, go_net.BUFFER_LEN)
	strMap := map[string]string{}

	for {
		chanBuf := <-reqChan
		conn := chanBuf.conn
		remoteAddr := chanBuf.remoteAddr

		if msgPack, errStr = go_net.UnPackBuf(chanBuf.buf); "" != errStr {
			fmt.Println("unpack failed")
			return
		}

		switch (*msgPack).Cmd {
		case go_net.CMD_GET_FILEINFO:

			(*respMsgPack).Content = infoByte
			fmt.Println("cmd:get file size")

		case go_net.CMD_GET_FILE:
			//respMsgPack.Content = buf[:1] /*
			ContentStr := string((*msgPack).Content)
			contentMap := go_net.StringToMap(ContentStr, strMap)

			//fmt.Println("cmd:get file")
			//fmt.Println("start:", contentMap["start"], ",end:", contentMap["end"], ",contentstr", ContentStr)
			start, err1 := strconv.ParseInt((contentMap["start"]), 10, 0)
			end, err2 := strconv.ParseInt((contentMap["end"]), 10, 0)

			if err1 != nil || err2 != nil || start < 0 || end < 0 || start > end {
				fmt.Println("parse error,recv data:", ContentStr)
				return
			}

			rbuf, rlen, errStr := GetFileBuffer(g_file, start, end)

			if "" != errStr {
				fmt.Println("get file error:", errStr)
			}
			//fmt.Println(rlen, len(rbuf))
			respMsgPack.Content = rbuf[:rlen]
			//fmt.Println("req len:", end-start+1, "ret len:", rlen)
		}
		sendBuf := go_net.PackBufWithGivenBytes(respMsgPack, buf, tmpBuf)

		//fmt.Println("resp:", go_net.Byte2Int(sendBuf[0:4]), go_net.Byte2Int(sendBuf[4:8]),
		//	len(sendBuf))

		conn.SetWriteDeadline(go_net.TimeOut(500))
		conn.WriteToUDP(sendBuf, remoteAddr)

		bufChan <- chanBuf
	}

}

func GetFileBuffer(file *os.File, start int64, end int64) ([]byte, int64, string) {
	buf_len := go_net.Min(end-start+1, go_net.BUFFER_CONTENT_LEN)

	realEnd := start - 1 + buf_len
	realEnd = go_net.Min(realEnd, g_fileInfo.Size())

	readByte := g_fileBuffer[start-1 : realEnd]

	rlen := realEnd - (start - 1)
	return readByte, rlen, ""
}
