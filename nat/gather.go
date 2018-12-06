package nat

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

var lanNets = []*net.IPNet{
	{net.IPv4(10, 0, 0, 0), net.CIDRMask(8, 32)},
	{net.IPv4(172, 16, 0, 0), net.CIDRMask(12, 32)},
	{net.IPv4(192, 168, 0, 0), net.CIDRMask(16, 32)},
	{net.ParseIP("fc00"), net.CIDRMask(7, 128)},
}

type candidate struct {
	Addr *net.UDPAddr
}

func (c candidate) String() string {
	return fmt.Sprintf("%v", c.Addr)
}

func (c candidate) Equal(c2 candidate) bool {
	return c.Addr.IP.Equal(c2.Addr.IP) && c.Addr.Port == c2.Addr.Port
}

func pruneDups(cs []candidate) []candidate {
	ret := make([]candidate, 0, len(cs))
	for _, c := range cs {
		unique := true
		for _, c2 := range ret {
			if c.Equal(c2) {
				unique = false
				break
			}
		}
		if unique {
			ret = append(ret, c)
		}
	}
	return ret
}
// 收集候选人
// sock:本地监听的UDP
// outIpList:提供的本地IP地址
// udpAddr:服务端的UDP服务地址
func GatherCandidates(sock *net.UDPConn, outIpList string, udpAddr string) ([]candidate, error) {
	laddr := sock.LocalAddr().(*net.UDPAddr)
	ret := []candidate{}
	switch {
	case laddr.IP.IsLoopback():
		return nil, errors.New("Connecting over loopback not supported")
	case laddr.IP.IsUnspecified():// 未指定，未使用
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			ip, ok := addr.(*net.IPNet)
			if ok && ip.IP.IsGlobalUnicast() {
				ret = append(ret, candidate{&net.UDPAddr{IP: ip.IP, Port: laddr.Port}})
			}
		}
	default:
		ret = append(ret, candidate{laddr})
	}

	addip := func(ipStr string, port int) {
		ip := net.ParseIP(ipStr)
		if port == 0 {
			port = laddr.Port
		}
		bHave := false
		for _, info := range ret {
			if info.Addr.IP.Equal(ip) && info.Addr.Port == port {
				bHave = true
				break
			}
		}
		if !bHave {
			ret = append(ret, candidate{&net.UDPAddr{IP: ip, Port: port}})
		}
	}

	if udpAddr != "" {
		addr, err := net.ResolveUDPAddr("udp", udpAddr)
		if err != nil {
			fmt.Println("Can't resolve udp address: ", err)
			return nil, err
		}
		p2pAddr := ""

		for i := 0; i < 5; i++ {
			println("write: ", "makehole", ";", sock.LocalAddr().String())
			// 发送数据到服务端UDP
			sock.WriteToUDP([]byte("makehole"), addr)
			buf := make([]byte, 100)
			sock.SetReadDeadline(time.Now().Add(time.Duration(1) * time.Second))
			n, udpAddr, err := sock.ReadFromUDP(buf)
			if err != nil {
				println("Can't ReadFromUDP: ", err, addr.String())
				continue
			} else {
				// 获取p2p地址，其实是自己的出口地址，被服务端返回过来而已，此处这么做的道理是自己如果是内网并不知道自己外网的地址
				p2pAddr = string(buf[0:n])
				println("read: ", p2pAddr, ";udpAddr:", udpAddr.String())
				break
			}
		}

		addLen := len(p2pAddr)
		if addLen > 0 {
			tmparr := strings.Split(p2pAddr, ":")

			var strip string
			var strport string
			strip, strport = tmparr[0], tmparr[1]
			ip := net.ParseIP(strip)
			port, _ := strconv.Atoi(strport)
			ret = append(ret, candidate{&net.UDPAddr{IP: ip, Port: port}})
		}
	}
	arr := strings.Split(outIpList, ";")

	for _, ip := range arr {
		addip(ip, 0)
	}

	/*	for _, info := range ret {
			log.Println("init ip:", info.Addr.String())
	}*/
	return ret, nil
}
