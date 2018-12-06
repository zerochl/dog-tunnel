package nat

import (
	"testing"
	"net"
	"log"
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
