package server

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"dog-tunnel/admin"
	"dog-tunnel/auth"
	"dog-tunnel/common"
)

var listenAddr = flag.String("addr", "0.0.0.0:8000", "server addr")
var listenAddrUDP = flag.String("addrudp", "0.0.0.0:8018", "udp server addr")
var bUseSSL = flag.Bool("ssl", false, "use ssl")
var bUseHttps = flag.Bool("https", false, "use https")
var certFile = flag.String("cert", "", "cert file")
var keyFile = flag.String("key", "", "key file")

var adminAddr = flag.String("admin", "", "admin addr")
var bShowVersion = flag.Bool("version", false, "show version")
var bReplaceReg = flag.Bool("replace", false, "if dup name registered, kick out the previous one, default is false")

var db_user = flag.String("dbuser", "", "db user")
var db_pass = flag.String("dbpass", "", "db password")
var db_host = flag.String("dbhost", "", "db host")

var bUseDB = false

var err error
var g_Master net.Listener

func StartServer() {
	flag.Parse()
	if *bShowVersion {
		fmt.Printf("%.2f\n", common.Version)
		return
	}
	common.Conn2ClientInfo = make(map[net.Conn]*common.ClientInfo)
	common.ServerName2Conn = make(map[string]net.Conn)
	common.Conn2Admin = make(map[net.Conn]*common.AdminInfo)
	// 监听TCP端口
	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Println("cannot listen addr:" + err.Error())
		return
	}
	// 是否使用ssl
	if *bUseSSL {
		config := &tls.Config{}
		config.Certificates = make([]tls.Certificate, 1)
		config.Certificates[0], err = tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			log.Println("load key file error", err.Error())
			return
		}
		g_Master = tls.NewListener(listener, config)
	} else {
		g_Master = listener
	}
	go func() {
		for {
			// 监听Tcp端口调用
			conn, err := g_Master.Accept()
			if err != nil {
				continue
			}
			// 收到链接请求异步执行客户端处理
			go handleClient(conn)
		}
	}()

	udpaddr, err := net.ResolveUDPAddr("udp", *listenAddrUDP)
	if err != nil {
		log.Println("Can't resolve address: ", err)
		return
	}
	// 监听UDP端口
	udpconn, err := net.ListenUDP("udp", udpaddr)
	if err != nil {
		log.Println("Error UDP listening:", err)
		return
	}

	log.Println("listenAdd: ", *listenAddrUDP)

	defer udpconn.Close()

	go udphandleClient(udpconn)
	// 如果提供DB则初始化DB连接
	if *db_host != "" {
		log.Println("init db")
		err = auth.Init(*db_user, *db_pass, *db_host)
		if err != nil {
			log.Println("mysql client fail", err.Error())
			return
		}
		defer auth.DeInit()
		bUseDB = true
	}
	log.Println("master start success")
	// 如果提供管理地址则开启管理地址初始化
	if *adminAddr != "" {
		log.Println("init admin server")
		cert, key := "", ""
		if *bUseHttps {
			cert, key = *certFile, *keyFile
		}
		err := admin.InitAdminPort(*adminAddr, cert, key)
		if err != nil {
			log.Println("admin service start fail", err.Error())
			return
		}
		log.Println("admin service start success")
	}
	// 创建信号接收channel，接收系统消息
	c := make(chan os.Signal, 1)
	// 接收结束程序信号
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Println("received signal,shutdown")
	shutdown()
}

func handleClient(conn net.Conn) {
	// 记录client info
	common.Conn2ClientInfo[conn] = &common.ClientInfo{Conn: conn, ClientMap: make(map[net.Conn]*common.Session), Id2Session: make(map[string]*common.Session), IsServer: false, Quit: make(chan bool), ResponseTime: time.Now().Unix()}
	log.Println("client linked success", conn.RemoteAddr().String())
	//TODO 监听时间，超过1800秒断开链接,此处有点莫名其妙
	common.Conn2ClientInfo[conn].Loop()
	// 执行读取操作，通过bufio的scanner进行读取,读取与写入有自定义格式
	common.Read(conn, handleResponse)
	// 如果当前读取完毕或者1800超时会断开连接
	// 断开连接之后需要关闭与清理客户端连接
	client, bHave := common.Conn2ClientInfo[conn]
	if bHave {
		// 往Quit channel写值，主动断开loop，并会断开conn
		close(client.Quit)
		if client.IsServer {
			//TODO client是服务端
			// 需要断开此服务端下面的所有客户端连接
			for conn, session := range client.ClientMap {
				conn.Close()
				//TODO 此处需要仔细看，方法内直接return，很奇怪
				common.RmId(client.ServerName, session.Id)
			}
			// 从map中删除某个item
			delete(common.ServerName2Conn, client.ServerName)
			log.Println("unregister service Name", client.ServerName)
			if bUseDB {
				// 如果使用DB，则设置用户状态为下线
				user, _ := auth.GetUser(client.UserName)
				if user != nil {
					user.OnLogout()
				}
			}
		} else {
			// 当前为用户连接
			// 获取client连接的server连接，并执行删除client连接的操作
			common.GetServerInfoByConn(conn, func(server *common.ClientInfo) {
				id := server.DelClient(conn)
				log.Println("send quit")
				common.Write(server.Conn, id, "clientquit", "")
			}, func() {})
		}
		// 从Conn2ClientInfo中删除conn
		delete(common.Conn2ClientInfo, conn)
	}
	conn.Close()
	log.Println("client disconnected", conn.RemoteAddr().String())
}

func udphandleClient(conn *net.UDPConn) {

	for {

		data := make([]byte, 1024)
		//TODO 此处读取data并未使用
		_, remoteAddr, err := conn.ReadFromUDP(data)
		if err != nil {
			log.Println("failed to read UDP msg because of ", err.Error())
			break
		}
		//TODO 将地址原封不动返回？
		conn.WriteToUDP([]byte(remoteAddr.String()), remoteAddr)
	}
}
// 接收客户端发送过来的数据
func handleResponse(conn net.Conn, id string, action string, content string) {
	log.Println("got", id, action, content)
	// 更新保存的客户端信息接收response的时间
	common.GetClientInfoByConn(conn, func(client *common.ClientInfo) {
		client.ResponseTime = time.Now().Unix()
	}, func() {
	})
	// action分为如下几个
	// 1:init
	// 2:tunnel_error
	// 3:makeholefail
	// 4:makeholeok
	// 5:report_addrlist
	// 6:success_bust_a
	// 7:tunnel_close
	// 8:tunnel_open
	// 9:tunnel_msg_c
	// 10:tunnel_msg_s
	// 11:tunnel_close_s
	switch action {
	case "init":
		clientInfoStr := content
		var clientInfo common.ClientSetting
		err := json.Unmarshal([]byte(clientInfoStr), &clientInfo)
		if err != nil {
			log.Println("error decode clientinfo, kick out", conn.RemoteAddr().String())
			common.Write(conn, "0", "showandquit", "server decode clientInfo error")
			return
		}
		if common.Version != clientInfo.Version {
			s_version := fmt.Sprintf("%.2f", common.Version)
			c_version := fmt.Sprintf("%.2f", clientInfo.Version)
			log.Println("version not eq", conn.RemoteAddr().String(), s_version, c_version)
			common.Write(conn, "0", "showandquit", "client version:"+c_version+" not eq with server:"+s_version)
			return
		}
		ServerName := clientInfo.Name
		if clientInfo.ClientType == "reg" {
			var user *auth.User
			if bUseDB {
				if clientInfo.AccessKey == "" {
					user, _ = auth.GetUser("test")
				} else {
					user, _ = auth.GetUserByKey(clientInfo.AccessKey)
				}
			} else {
				user = &auth.User{UserType: auth.UserType_Admin}
			}
			//fmt.Printf("%+v\n", user)
			if user == nil {
				common.Write(conn, "0", "showandquit", "invalid user accessKey:"+clientInfo.AccessKey+"!!!")
				return
			}
			if !user.CheckOnlineServiceNum() {
				common.Write(conn, "0", "showandquit", "online service num cannot overstep "+strconv.Itoa(user.MaxOnlineServerNum))
				return
			}
			if !user.CheckIpLimit(conn.RemoteAddr().(*net.TCPAddr).IP.String()) {
				common.Write(conn, "0", "showandquit", "ip limit service num cannot overstep "+strconv.Itoa(user.MaxSameIPServers))
				return
			}
			f := func() {
				common.ServerName2Conn[ServerName] = conn
				common.GetClientInfoByConn(conn, func(info *common.ClientInfo) {
					info.ServerName = ServerName
					info.IsServer = true
					info.Id2MakeSession = make(map[string]*common.UDPMakeSession)
					info.UserName = user.UserName
					info.ClientKey = clientInfo.ClientKey
				}, func() {})
				log.Println("client reg service success", conn.RemoteAddr().String(), user.UserName, ServerName)
				common.Write(conn, "0", "show", "register service ok, user:"+user.UserName)
			}
			common.GetClientInfoByName(ServerName, func(server *common.ClientInfo) {
				if *bReplaceReg {
					_conn := server.Conn
					close(server.Quit)
					for conn, session := range server.ClientMap {
						conn.Close()
						common.RmId(server.ServerName, session.Id)
					}
					delete(common.ServerName2Conn, server.ServerName)
					log.Println("force unregister service Name", server.ServerName)
					if bUseDB {
						user, _ := auth.GetUser(server.UserName)
						if user != nil {
							user.OnLogout()
						}
					}
					delete(common.Conn2ClientInfo, _conn)
					common.Write(_conn, "0", "showandquit", "some one kick you out")
					_conn.Close()
					f()
				} else {
					common.Write(conn, "0", "showandretry", "already have the ServerName!")
				}
			}, f)
		} else if clientInfo.ClientType == "link" {
			if clientInfo.Mode < 0 || clientInfo.Mode > 2 {
				clientInfo.Mode = 0
			}
			ServerName := clientInfo.Name
			bAuth := true
			common.GetClientInfoByName(ServerName, func(info *common.ClientInfo) {
				var user *auth.User
				if bUseDB {
					user, _ = auth.GetUser(info.UserName)
				} else {
					user = &auth.User{UserType: auth.UserType_Admin}
				}
				//fmt.Printf("%+v\n", user)
				if user == nil {
					common.Write(conn, "0", "showandquit", "invalid user:"+info.UserName+"!!!")
					bAuth = false
					return
				}
				if info.ClientKey != clientInfo.ClientKey {
					common.Write(conn, "0", "showandquit", "clientkey invalid!!!")
					bAuth = false
					return
				}
				if !user.CheckSessionNum(len(info.ClientMap)) {
					common.Write(conn, "0", "showandquit", "session numcannot overstep "+strconv.Itoa(len(info.ClientMap)))
					bAuth = false
					return
				}
				if !user.CheckPipeNum(clientInfo.PipeNum) {
					common.Write(conn, "0", "showandquit", "pipenum cannot overstep "+strconv.Itoa(user.MaxPipeNum))
					bAuth = false
					return
				}
			}, func() {
				common.Write(conn, "0", "showandquit", "serverName invalid!!!")
				bAuth = false
			})
			if !bAuth {
				return
			}
			common.GetClientInfoByConn(conn, func(client *common.ClientInfo) {
				client.ServerName = ServerName
			}, func() {
			})
			common.GetClientInfoByName(ServerName, func(server *common.ClientInfo) {
				log.Println("client link service success", conn.RemoteAddr().String(), ServerName)
				server.AddClient(conn, clientInfo)
			}, func() {
				common.Write(conn, "0", "showandquit", "donnt have this service name")
			})
		}
	case "tunnel_error":
		common.GetServerInfoByConn(conn, func(server *common.ClientInfo) {
			log.Println("<<=====tunnel_error", server.ServerName, conn.RemoteAddr().String())
			session, bHave := server.Id2Session[id]
			if bHave {
				session.Status = "fail"
				common.Write(session.ClientA, "0", "showandquit", content)
				server.DelClient(session.ClientA)
			}
		}, func() {
		})
	case "makeholefail":
		common.GetServerInfoByConn(conn, func(server *common.ClientInfo) {
			udpsession, bHave := server.Id2MakeSession[id]
			if bHave {
				log.Println("<<=====make hole fail", conn.RemoteAddr().String(), udpsession.ServerName, udpsession.SessionId, id)
				sessionId := udpsession.SessionId
				session, _bHave := server.Id2Session[sessionId]
				if _bHave {
					session.Status = "fail"
					session.MakeHoleResponseN++
					session.MakeHoleHasFail = true
					if session.MakeHoleResponseN == session.Setting.PipeNum {
						if session.Method == "udp" {
							session.RestartSession(server.ServerName)
						} else if session.Method == "restart" {
							if session.Setting.Mode == 0 {
								tmp := session.ClientA
								session.ClientA = session.ClientB
								session.ClientB = tmp
								session.StartCSMode()
							} else {
								server.DelClient(session.ClientB)
							}
						} else {
							server.DelClient(session.ClientA)
						}
					}
				}
				udpsession.Remove(false)
			}
		}, func() {
		})
	case "makeholeok":
		common.GetServerInfoByConn(conn, func(server *common.ClientInfo) {
			if content == "csmode" {
				session, _bHave := server.Id2Session[id]
				if _bHave {
					log.Println("<<=====make hole ok", conn.RemoteAddr().String(), server.ServerName, session.Id)
					session.Status = "ok"
					session.MakeHoleResponseN++
				}
			}
			udpsession, bHave := server.Id2MakeSession[id]
			if bHave {
				log.Println("<<=====make hole ok", conn.RemoteAddr().String(), udpsession.ServerName, udpsession.SessionId, id)
				sessionId := udpsession.SessionId
				session, _bHave := server.Id2Session[sessionId]
				if _bHave {
					session.MakeHoleResponseN++
					if session.MakeHoleResponseN == session.Setting.PipeNum {
						if !session.MakeHoleHasFail {
							session.Status = "ok"
						}
					}
				}
				udpsession.Remove(false)
			}
		}, func() {
		})
	case "report_addrlist":
		common.GetServerInfoByConn(conn, func(server *common.ClientInfo) {
			udpsession, bHave := server.Id2MakeSession[id]
			//log.Println("test", udpsession, id, server.ServerName)
			if bHave {
				log.Println("<<===report addr list ok", conn.RemoteAddr().String(), udpsession.ServerName, udpsession.Id)
				udpsession.BeginMakeHole(1, content)
			}
		}, func() {
		})
	case "success_bust_a":
		common.GetServerInfoByConn(conn, func(server *common.ClientInfo) {
			udpsession, bHave := server.Id2MakeSession[id]
			if bHave {
				log.Println("<<=====success_bust_a", conn.RemoteAddr().String(), udpsession.ServerName, udpsession.SessionId, id)
				udpsession.BeginMakeHole(2, content)
			}
		}, func() {
		})
	// for c/s mode
	case "tunnel_close":
		common.GetServerInfoByConn(conn, func(server *common.ClientInfo) {
			session := server.GetSession(conn)
			if session != nil {
				common.Write(session.ClientB, session.Id+"-"+id, "csmode_s_tunnel_close", content)
			} else {
				println("no session")
			}
		}, func() {
		})
	case "tunnel_open":
		common.GetServerInfoByConn(conn, func(server *common.ClientInfo) {
			session := server.GetSession(conn)
			if session != nil {
				common.Write(session.ClientB, session.Id+"-"+id, "csmode_s_tunnel_open", content)
			} else {
				println("no session")
			}
		}, func() {
		})
	case "tunnel_msg_c":
		common.GetServerInfoByConn(conn, func(server *common.ClientInfo) {
			var user *auth.User
			if bUseDB {
				user, _ = auth.GetUser(server.UserName)
			} else {
				user = &auth.User{UserType: auth.UserType_Admin}
			}
			if user == nil {
				common.Write(conn, "0", "showandquit", "cannot get userinfo of this service "+server.UserName)
				return
			}
			if !user.UpdateCSMode(len(content)) {
				common.Write(conn, "0", "showandquit", "reach today's csmode data limit")
				return
			}
			session := server.GetSession(conn)
			if session != nil {
				common.Write(session.ClientB, session.Id+"-"+id, "csmode_msg_c", content)
			} else {
				println("no session")
			}
		}, func() {
		})
	case "tunnel_msg_s":
		common.GetServerInfoByConn(conn, func(server *common.ClientInfo) {
			var user *auth.User
			if bUseDB {
				user, _ = auth.GetUser(server.UserName)
			} else {
				user = &auth.User{UserType: auth.UserType_Admin}
			}
			if user == nil {
				common.Write(conn, "0", "showandquit", "cannot get userinfo of this service"+server.UserName)
				return
			}
			if !user.UpdateCSMode(len(content)) {
				common.Write(conn, "0", "show", "reach today's csmode data limit")
				return
			}
			arr := strings.Split(id, "-")
			clientId := arr[0]
			session, bHave := server.Id2Session[clientId]
			if bHave {
				common.Write(session.ClientA, id, "csmode_msg_s", content)
			} else {
				println("no session")
			}
		}, func() {
		})
	case "tunnel_close_s":
		common.GetServerInfoByConn(conn, func(server *common.ClientInfo) {
			arr := strings.Split(id, "-")
			clientId := arr[0]
			session, bHave := server.Id2Session[clientId]
			if bHave {
				common.Write(session.ClientA, id, "csmode_c_tunnel_close", content)
			} else {
				println("no session")
			}
		}, func() {
		})
	}
}

func shutdown() {
	for conn, client := range common.Conn2ClientInfo {
		if !client.IsServer {
			log.Println("shutdown client", client.ServerName)
			common.Write(conn, "0", "showandquit", "server shutdown")
		} else {
			log.Println("unregister service Name", client.ServerName)
			if bUseDB {
				user, _ := auth.GetUser(client.UserName)
				if user != nil {
					user.OnLogout()
				}
			}
			//donnot showandquit,because client_server need to reconnect
		}
	}
}
