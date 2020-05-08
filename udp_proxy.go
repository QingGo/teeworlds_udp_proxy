package main

import (
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var timeoutSencond = 20
var wg = &sync.WaitGroup{}

// var localSocketLock sync.Mutex

// UDPClient for udp client connection
type UDPClient struct {
	ClientAddr *net.UDPAddr
	ServerAddr *net.UDPAddr

	ProxyAddr   net.UDPAddr
	ProxySocket *net.UDPConn

	SendMsgQueue    chan []byte
	ReceiveMsgQueue chan []byte
}

func (client *UDPClient) bindRandomProxyPort(minProxyPort int, maxProxyPort int) (proxyPort int, err error) {
	proxyPort = rand.Intn(maxProxyPort-minProxyPort) + minProxyPort
	client.ProxyAddr = net.UDPAddr{
		Port: proxyPort,
		IP:   net.ParseIP("0.0.0.0"),
	}
	client.ProxySocket, err = net.ListenUDP("udp", &client.ProxyAddr)
	return proxyPort, err
}

// Init UDPClient constructor
func (client *UDPClient) Init(clientAddr *net.UDPAddr, serverAddr *net.UDPAddr) {
	client.ClientAddr = clientAddr
	client.ServerAddr = serverAddr

	minProxyPort, maxProxyPort := 22223, 22801
	// not including maxProxyPort
	for {
		proxyPort, err := client.bindRandomProxyPort(minProxyPort, maxProxyPort)
		if err == nil {
			log.Infof("bind port %d.", proxyPort)
			break
		}
	}

	client.SendMsgQueue = make(chan []byte, 100)
	client.ReceiveMsgQueue = make(chan []byte, 100)
}

// RunSendToServer goroutine to send messge to server
func (client *UDPClient) RunSendToServer(deleteClientQueue chan string) {
	defer wg.Done()
	defer client.ProxySocket.Close()
	for {
		select {
		case msg := <-client.SendMsgQueue:
			_, err := client.ProxySocket.WriteToUDP(msg, client.ServerAddr)
			if err != nil {
				log.Infof("RunSendToServer whih %s goroutine stop: %s", client.ProxyAddr.String(), err.Error())
				deleteClientQueue <- client.ClientAddr.String()
				return
			}
			log.Debugf("send message %s to server from client %s", msg, client.ClientAddr)
		case <-time.After(time.Second * time.Duration(timeoutSencond)):
			deleteClientQueue <- client.ClientAddr.String()
			log.Infof("RunSendToServer whih %s goroutine stop: not receive client data in %d s", client.ProxyAddr.String(), timeoutSencond)
			return
		}
	}

}

// RunReceiveFromServer goroutine to receive messge from server
func (client *UDPClient) RunReceiveFromServer(deleteClientQueue chan string) {
	defer wg.Done()
	buf := make([]byte, 32768)
	for {
		// raise err if RunSendToServer close socket
		n, _, err := client.ProxySocket.ReadFromUDP(buf)
		if err != nil {
			log.Infof("RunReceiveFromServer whih %s goroutine stop: %s", client.ClientAddr, err.Error())
			break
		}
		newbuf := make([]byte, n)
		copy(newbuf, buf[:n])
		client.ReceiveMsgQueue <- newbuf
		log.Debugf("receive message %s from server to client %s", newbuf, client.ClientAddr)
	}
	// pass client.ClientAddr.String() twice can cause a block when client want to reconnect after timeout, why?
	// deleteClientQueue <- client.ClientAddr.String()
}

// UDPClientManager is manager of UDPClient
type UDPClientManager struct {
	LocalPort  int
	RemoteIP   string
	RemotePort int
	ServerAddr net.UDPAddr
	LocalAddr  net.UDPAddr

	LocalSocket *net.UDPConn

	MaxClient           int
	ClientDict          map[string]*UDPClient
	ClientGoroutineDict map[string]bool
	DeleteClientQueue   chan string
	DoneQueue           map[string]chan bool
}

// Init UDPClientManager constructor
func (manager *UDPClientManager) Init(localPort int, remoteIP string, remotePort int, maxClient int) {
	manager.LocalPort = localPort
	manager.RemoteIP = remoteIP
	manager.RemotePort = remotePort
	manager.MaxClient = maxClient

	manager.ClientDict = make(map[string]*UDPClient)
	manager.ClientGoroutineDict = make(map[string]bool)
	manager.DeleteClientQueue = make(chan string, 100)
	manager.DoneQueue = make(map[string]chan bool)

	manager.ServerAddr = net.UDPAddr{
		Port: remotePort,
		IP:   net.ParseIP(remoteIP),
	}

	manager.LocalAddr = net.UDPAddr{
		Port: manager.LocalPort,
		IP:   net.ParseIP("0.0.0.0"),
	}
	var err error
	// socket should not close by set timeout here
	// should exit inactive goroutine using kinds of signal
	manager.LocalSocket, err = net.ListenUDP("udp", &manager.LocalAddr)
	if err != nil {
		log.Debugf("Fatal error: %s", err.Error())
	}
}

// IsClientExisted check if client existed
func (manager *UDPClientManager) IsClientExisted(clientAddr *net.UDPAddr) bool {
	_, isExisted := manager.ClientDict[clientAddr.String()]
	return isExisted
}

// ReceiveFromClient goroutine to receive messge from client,
// should run until this program over
func (manager *UDPClientManager) ReceiveFromClient() {
	defer manager.LocalSocket.Close()
	defer wg.Done()
	buf := make([]byte, 32768)
	for {

		n, clientAddr, err := manager.LocalSocket.ReadFromUDP(buf)
		if err != nil {
			log.Infof("ReceiveFromClient whih %s goroutine stop: %s", clientAddr, err.Error())
			break
		}
		if manager.TryAddClient(clientAddr) {
			client := manager.ClientDict[clientAddr.String()]
			// the data of dummy is innormal,
			// 消息像是先被切断又被组合到了一起，也许是因为在对面接收到消息前slice指向的array就被修改了
			// 因此先复制一份同样的slice再传入channel
			newbuf := make([]byte, n)
			copy(newbuf, buf[:n])
			client.SendMsgQueue <- newbuf
			log.Debugf("receive message %s from client %s", newbuf, clientAddr.String())
		}
	}
}

// SendToClient goroutine to send messge to specific client
func (manager *UDPClientManager) SendToClient(client *UDPClient, done chan bool) {
	defer wg.Done()
	for {
		select {
		case msg := <-client.ReceiveMsgQueue:
			manager.LocalSocket.WriteToUDP(msg, client.ClientAddr)
			log.Debugf("send message %s to client %s", msg, client.ClientAddr)
		case <-done:
			log.Infof("stop goroutine with client %s", client.ClientAddr)
			delete(manager.DoneQueue, client.ClientAddr.String())
			return
		}
	}

}

// TryAddClient try to add clent, if already existed or succeed in creating, reutrn true
func (manager *UDPClientManager) TryAddClient(clientAddr *net.UDPAddr) bool {
	if manager.IsClientExisted(clientAddr) {
		return true
	} else if len(manager.ClientDict) > manager.MaxClient {
		log.Debugf("Already have %d clients, ignore Client %s", len(manager.ClientDict), clientAddr.String())
		return false
	} else {
		client := new(UDPClient)
		client.Init(clientAddr, &manager.ServerAddr)
		manager.ClientDict[clientAddr.String()] = client
		log.Infof("Add Client %s", clientAddr.String())

		manager.DoneQueue[clientAddr.String()] = make(chan bool)
		wg.Add(1)
		go client.RunReceiveFromServer(manager.DeleteClientQueue)
		wg.Add(1)
		go manager.SendToClient(client, manager.DoneQueue[clientAddr.String()])
		wg.Add(1)
		go client.RunSendToServer(manager.DeleteClientQueue)

		return true
	}
}

// DelClientAsync delect inactive client from ClientDict when received mse from DeleteClientQueue
func (manager *UDPClientManager) DelClientAsync() {
	defer wg.Done()
	for {
		clientAddr := <-manager.DeleteClientQueue
		// it is OK even if clientAddr have been deleted
		// UDPClient object should be clean by GC cause reference counting become 0
		// socket should close by hand with defer
		log.Debugf("delete %s from manager.ClientDict", clientAddr)
		delete(manager.ClientDict, clientAddr)
		manager.DoneQueue[clientAddr] <- true
	}

}

// Run entry of UDPClientManager
func (manager *UDPClientManager) Run() {
	wg.Add(1)
	go manager.ReceiveFromClient()
	wg.Add(1)
	go manager.DelClientAsync()

	wg.Wait()

}

func main() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	// log.SetLevel(log.InfoLevel)
	log.SetReportCaller(true)

	if len(os.Args) < 4 {
		log.Infof("Usage: python3 udp_proxy.py <local port> <remote ip> <remote port>")
	}
	localPortStr, remoteIP, remotePortStr := os.Args[1], os.Args[2], os.Args[3]
	localPort, _ := strconv.Atoi(localPortStr)
	remotePort, _ := strconv.Atoi(remotePortStr)

	log.Infof("local_port: %v, remote_ip: %s, remote_port: %v", localPort, remoteIP, remotePort)

	maxClient := 12
	manager := new(UDPClientManager)
	manager.Init(localPort, remoteIP, remotePort, maxClient)
	manager.Run()
}
