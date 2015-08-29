package go_net

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"
)

type MsgPack struct {
	Cmd     int32
	Content []byte
}

const (
	PACKAGE_HEAD       = 4
	PACKAGE_CMD        = 8
	BUFFER_CONTENT_LEN = 1024
)
const (
	BUFFER_LEN = PACKAGE_CMD + BUFFER_CONTENT_LEN
)
const (
	CMD_GET_FILEINFO = 0
	CMD_GET_FILE     = 1
)

func GetGoRoutineNum() int32 {
	return int32(Min(400, int64(runtime.NumCPU()*150)))
}

func GoListen(address string) (*net.UDPConn, bool) {
	addr, err := net.ResolveUDPAddr("udp", address)

	if nil != err {
		fmt.Println("can't resolve address: ", err)
		return nil, false
	}

	udpConn, err := net.ListenUDP("udp", addr)

	if nil != err {
		fmt.Println("bind address ", address, " error:", err)
		return nil, false
	}
	return udpConn, true
}

func Byte2Int(by []byte) int32 {

	if 4 > len(by) {
		return 0
	}

	res := int32(by[3])
	res += (int32(by[2])) << 8
	res += (int32(by[1])) << 16
	res += (int32(by[0])) << 24

	return res
}

func Int2Byte(value int32) []byte {
	b_buf := bytes.NewBuffer([]byte{})
	binary.Write(b_buf, binary.BigEndian, value)
	return b_buf.Bytes()
}

func Int2ByteWithBuf(res int32, buf []byte) []byte {
	/*b_buf := bytes.NewBuffer(buf)
	binary.Write(b_buf, binary.BigEndian, value)*/

	buf[0] = byte(res >> 24)
	buf[1] = byte(res >> 16)
	buf[2] = byte(res >> 8)
	buf[3] = byte(res)

	return buf
}

func StringToMap(str string, strMap map[string]string) map[string]string {
	strArray := strings.Split(str, "&")

	for _, v := range strArray {
		strTmpArr := strings.Split(v, "=")

		if 2 > len(strTmpArr) {
			continue
		}

		strMap[strTmpArr[0]] = strTmpArr[1]
	}

	return strMap
}

func UnPackBuf(recb []byte) (*MsgPack, string) {
	if PACKAGE_CMD > len(recb) {
		return nil, "pack is too short"
	}

	rlen := Byte2Int((recb)[:PACKAGE_HEAD])

	if int32(len(recb)) != (rlen + PACKAGE_HEAD) {
		return nil, "the pack is not valid"
	}

	msgPack := &MsgPack{-1, nil}
	msgPack.Cmd = Byte2Int((recb)[PACKAGE_HEAD:PACKAGE_CMD])
	msgPack.Content = (recb)[PACKAGE_CMD:]

	return msgPack, ""
}

func PackBuf(msgPack *MsgPack) *[]byte {

	contentBuff := Int2Byte((*msgPack).Cmd)

	contentBuff = append(contentBuff, (*msgPack).Content...)

	lenBuf := Int2Byte(int32(len(contentBuff)))

	buf := append(lenBuf, contentBuff...)
	return &buf
}

func PackBufWithGivenBytes(msgPack *MsgPack, buf []byte, buf2 []byte) []byte {
	cmdBuff := buf2[:4]
	lenBuf := buf2[4:8]

	lenContent := len((*msgPack).Content)
	cmdBuff = Int2ByteWithBuf((*msgPack).Cmd, cmdBuff)
	lenBuf = Int2ByteWithBuf(int32(lenContent+len(cmdBuff)), lenBuf)

	//fmt.Println(lenBuf, int32(lenContent+len(cmdBuff)))
	//time.Sleep(10 * time.Second)

	bufHead := buf[:PACKAGE_HEAD]
	bufCmd := buf[PACKAGE_HEAD:PACKAGE_CMD]
	bufContent := buf[PACKAGE_CMD : PACKAGE_CMD+lenContent]

	copy(bufHead, lenBuf)
	copy(bufCmd, cmdBuff)
	copy(bufContent, msgPack.Content)

	return buf[:PACKAGE_CMD+lenContent]
}

func TimeOut(millSec time.Duration) time.Time {
	return time.Now().Add(time.Millisecond * millSec)
}

func Min(v1 int64, v2 int64) int64 {
	if v1 < v2 {
		return v1
	} else {
		return v2
	}
}

func Max(v1 int64, v2 int64) int64 {
	if v1 > v2 {
		return v1
	} else {
		return v2
	}
}

func Md5(v []byte) string {
	h := md5.New()
	h.Write(v) // 需要加密的字符串为 sharejs.com
	return hex.EncodeToString(h.Sum(nil))
}
