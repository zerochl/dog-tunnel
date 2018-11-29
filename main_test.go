package main

import (
	"bufio"
	"strings"
	"testing"
	"log"
	"bytes"
	"io"
	"dog-tunnel/common"
)

func TestTest(t *testing.T)  {
	//var b [8]byte
	//var bs []byte
	//bs = b[:4]
	//strByte := []byte("0001000112341234567890")

	xor := common.Xor("abcd")
	xor2 := common.Xor(xor)
	log.Println("xor:", xor)
	log.Println("xor2:", xor2)

	// 开始测试common中的Read
	//strByte := []byte{4,0,0,0,2,0,0,0,3,0,0,0,9,2,3,4,5,6,7,8}
	//
	//var l1, l2, l3 uint32
	//buf := bytes.NewReader(strByte)
	//binary.Read(buf, binary.LittleEndian, &l1)
	//binary.Read(buf, binary.LittleEndian, &l2)
	//binary.Read(buf, binary.LittleEndian, &l3)
	//log.Println("l1", l1)
	//log.Println("l2", l2)
	//log.Println("l3", l3)
	//
	//id := make([]byte, l1)
	////log.Println("test")
	//err := binary.Read(buf, binary.LittleEndian, &id)
	//if err != nil {
	//	log.Println("err:", err.Error())
	//	return
	//}
	//log.Println("id length:", len(id))
	//log.Println("id:", id)
	// 结束common中的read

	//var b [8]byte
	//var bs []byte
	//bs = b[:5]
	//buf := bytes.NewReader(strByte)
	//_, err := io.ReadFull(buf, bs)
	//if err != nil {
	//	log.Println("error:", err.Error())
	//	return
	//}
	//log.Println("b:", bs)
}

func TestScan(t *testing.T)  {
	input := "foo  bar   baz"
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		log.Println(scanner.Text())
	}
}

func TestScanCustom(t *testing.T)  {
	input := "fooabcfoofoofoo"
	scanner := bufio.NewScanner(strings.NewReader(input))
	split := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		//log.Printf("data[:3]:%s", data[:3])
		//log.Printf("%t\t%d\t%s\n", atEOF, len(data), data)
		if bytes.Equal(data[:3], []byte{'f', 'o', 'o'}) {
			return 3, []byte{'F'}, nil
		}
		if atEOF {
			return 0, nil, io.EOF
		}
		return 1, []byte(""), nil
	}
	scanner.Split(split)
	//buf := make([]byte, 12)
	//scanner.Buffer(buf, bufio.MaxScanTokenSize)
	for scanner.Scan() {
		log.Printf("scanner:%s\n", scanner.Text())
	}
}