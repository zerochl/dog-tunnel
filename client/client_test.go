package client

import (
	"testing"
	"log"
	"net"
	"dog-tunnel/common"
	"bufio"
	"dog-tunnel/nat"
)

func TestClientAsServer(t *testing.T)  {
	StartClient()
}

func TestClientAsClient(t *testing.T)  {
	StartClient()
}

func TestTest(t *testing.T)  {
	var currIdMap map[string]int
	if currIdMap == nil {
		currIdMap = make(map[string]int)
	}
	currIdMap["test"]++
	log.Println("currIdMap:", currIdMap["test1"])
}

func TestClient_MultiListen(t *testing.T) {
	g_LocalConn, _ := net.Listen("tcp", *localAddr)
	for {
		conn, err := g_LocalConn.Accept()
		if err != nil {
			continue
		}
		sessionId := common.GetId("udp")
		println("sessionId:", sessionId)
		reader := bufio.NewReader(conn)
		write := bufio.NewWriter(conn)
		arr := make([]byte, nat.SendBuffSize)
		go func() {
			for {
				size, err := reader.Read(arr)
				if err != nil {
					println("error:", err.Error())
					break
				}
				log.Println("size:", size, ";data:", string(arr[:size]))
				write.Write([]byte("测试"))
			}
		}()
	}
}