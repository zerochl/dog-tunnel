package nat

import (
	"testing"
	"net"
	"log"
	"dog-tunnel/common"
)

func TestInit(t *testing.T) {
	sock, err := net.ListenUDP("udp", &net.UDPAddr{})
	if err != nil {
		return
	}
	log.Println("local:", sock.LocalAddr().String())
	//sock.WriteToUDP([]byte("makehole"), addr)
	//addrs, err := net.InterfaceAddrs()
	//if err != nil {
	//	return
	//}
	//for _, addr := range addrs {
	//	ip, ok := addr.(*net.IPNet)
	//	if ok && ip.IP.IsGlobalUnicast() {
	//		log.Println("addrs:", addr.String())
	//	}
	//}
}

func TestStartNatServer(t *testing.T)  {
	// server
	//outIpList: 127.0.0.1;127.0.0.1
	//buster: false
	//id: 1
	//udpAddr: 127.0.0.1:8018

	// client
	//outIpList: 127.0.0.1;127.0.0.1
	//buster: true
	//id: 1
	//udpAddr: 127.0.0.1:8018


	report := func() {
		println("in report")
	}

	engin, _ := Init("127.0.0.1;127.0.0.1", false, 1, "127.0.0.1:8018")
	conn, _ := engin.GetConn(report, nil, nil)
	callback := func(conn net.Conn, sessionId, action, content string) {
		println("action:", action, ";content:", content)
	}
	common.Read(conn, callback)
}

func TestStartNatClient(t *testing.T)  {
	report := func() {
		println("in report")
	}

	engin, _ := Init("127.0.0.1;127.0.0.1", true, 1, "127.0.0.1:8018")
	conn, _ := engin.GetConn(report, nil, nil)
	callback := func(conn net.Conn, sessionId, action, content string) {
		println("action:", action, ";content:", content)
	}
	common.Read(conn, callback)
}