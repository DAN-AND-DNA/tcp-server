package easyjson

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
)

type EasyInputHandler func(conn *EasyTcpConn, InputBuffer []byte) (Readed int, bytesResult []byte, err error)
type EasyOutputHandler func(conn *EasyTcpConn, OutputBuffer []byte) (err error)

type EasyConf struct {
	HandleInput  EasyInputHandler
	HandleOutput EasyOutputHandler
	Address      string
}

type EasyTcpServer struct {
	Conf EasyConf

	conns       map[string]*EasyTcpConn
	connsM      sync.RWMutex
	baseId      uint32
	listener    net.Listener
	newConnChan chan net.Conn
	bRun        bool
	isRunningM  sync.Mutex
	ctx         context.Context
	cancel      context.CancelFunc
	connWg      sync.WaitGroup
	wg          sync.WaitGroup
}

func NewEasyTcpServer() *EasyTcpServer {
	easyTcpServer := EasyTcpServer{
		conns:       make(map[string]*EasyTcpConn),
		baseId:      1000,
		bRun:        false,
		newConnChan: make(chan net.Conn),
		wg:          sync.WaitGroup{},
		connWg:      sync.WaitGroup{},
	}

	easyTcpServer.ctx, easyTcpServer.cancel = context.WithCancel(context.Background())

	return &easyTcpServer
}

func (this *EasyTcpServer) Run(conf EasyConf) error {
	if this.Conf.Address == "" {
		this.Conf.Address = "0.0.0.0:3000"
	}

	this.Conf = conf

	var err error
	this.listener, err = net.Listen("tcp", this.Conf.Address)
	if err != nil {
		return err
	}

	this.bRun = true
	log.Println("server Run:[ok]")
	log.Printf("start listening at: %s\n", this.Conf.Address)

	this.AcceptNewConn()
	this.ProcessNewConn()
	return nil
}

func (this *EasyTcpServer) Stop() {
	this.isRunningM.Lock()
	this.bRun = false
	this.isRunningM.Unlock()

	this.cancel()

	// make all no block
	this.listener.Close()
	this.wg.Wait()

	log.Println("server is waitting for conn close...")
	for id, conn := range this.conns {
		delete(this.conns, id)
		conn.Close()
	}
	this.connWg.Wait()
	log.Println("server Stop:[ok]")
}

func (this *EasyTcpServer) AcceptNewConn() {
	go func() {
		this.wg.Add(1)
		defer this.wg.Done()

		log.Println("server AcceptNewConn:[ok]")
		for {
			select {
			case <-this.ctx.Done():
				log.Println("server AcceptNewConn exit:[ok]")
				return

			default:
				// blocking
				rawConn, err := this.listener.Accept()
				if err != nil {
					this.isRunningM.Lock()
					if !this.bRun {
						this.isRunningM.Unlock()
						log.Println("server AcceptNewConn exit:[ok]")
						return
					}
					this.isRunningM.Unlock()
					log.Printf("server AcceptNewConn got error:[%s]\n", err.Error())
					continue
				}

				// blocking: one conn
				this.newConnChan <- rawConn
			}
		}
	}()
}

func (this *EasyTcpServer) ProcessNewConn() {
	go func() {
		this.wg.Add(1)
		defer this.wg.Done()

		log.Println("server ProcessNewConn:[ok]")
		for {
			select {
			case rawConn := <-this.newConnChan:
				this.baseId++
				newConnId := fmt.Sprintf("easyid#%d", this.baseId)
				log.Printf("server ProcessNewConn new conn:[%s]\n", newConnId)
				newConn := NewEasyConn(this, &this.connWg, newConnId, rawConn).Run(this.Conf.HandleInput, this.Conf.HandleOutput)
				this.conns[newConnId] = newConn

			case <-this.ctx.Done():
				log.Println("server ProcessNewConn exit:[ok]")
				return
			}
		}
	}()
}

func (this *EasyTcpServer) RemoveConnection(id string) {
	this.connsM.Lock()
	defer this.connsM.Unlock()

	delete(this.conns, id)
	log.Printf("client[%s] is closed\n", id)
}

func (this *EasyTcpServer) GetConnection(id string) (*EasyTcpConn, bool) {
	this.connsM.RLock()
	defer this.connsM.RUnlock()

	conn, ok := this.conns[id]
	if ok {
		return conn, true
	}

	return nil, false
}
