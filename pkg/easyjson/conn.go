package easyjson

import (
	"context"
	"io"
	"log"
	"net"
	"sync"
)

type EasyTcpConn struct {
	ProcessInput EasyInputHandler
	Server       *EasyTcpServer

	rawConn        net.Conn
	TConn          net.Conn
	id             string
	outputChan     chan struct{}
	outputBuffer   []byte
	inputBuffer    []byte
	outputBufferMu sync.Mutex
	readIndex      int
	writeIndex     int
	outReadIndex   int
	outWriteIndex  int
	ctx            context.Context
	cancel         context.CancelFunc
	bRun           bool
	isRunningM     sync.Mutex
	wg             *sync.WaitGroup
}

func NewEasyConn(server *EasyTcpServer, wg *sync.WaitGroup, id string, rawConn net.Conn) *EasyTcpConn {
	newConn := EasyTcpConn{
		rawConn:      rawConn,
		TConn:        rawConn,
		id:           id,
		outputChan:   make(chan struct{}, 100),
		outputBuffer: make([]byte, 4000),
		inputBuffer:  make([]byte, 4000),
		bRun:         false,
		wg:           wg,
		Server:       server,
	}

	newConn.ctx, newConn.cancel = context.WithCancel(context.Background())

	return &newConn
}

func (this *EasyTcpConn) Run(ProcessInput EasyInputHandler, ProcessOutput EasyOutputHandler) *EasyTcpConn {
	if this.rawConn == nil || this.id == "" {
		return this
	}

	this.bRun = true

	go this.handleRead(ProcessInput)
	go this.handleWrite(ProcessOutput)

	return this
}

func (this *EasyTcpConn) Close() {
	this.Server.RemoveConnection(this.id)

	this.isRunningM.Lock()
	this.bRun = false
	this.isRunningM.Unlock()

	this.rawConn.Close()
	this.cancel()
}

func (this *EasyTcpConn) handleRead(ProcessInput EasyInputHandler) {
	this.wg.Add(1)
	defer this.wg.Done()

	tcpConn, ok := this.rawConn.(*net.TCPConn)
	if !ok {
		this.Close()
		return
	}

	inputBufferTick := make([]byte, 1000)

	for {

		select {
		case _ = <-this.ctx.Done():
			log.Printf("client[%s] handleRead exit:[ok]\n", this.id)
			return
		default:

			//log.Printf("[pre] read:%d write:%d\n", this.readIndex, this.writeIndex)
			// 1. clear buffer
			if this.readIndex == this.writeIndex {
				this.readIndex = 0
				this.writeIndex = 0

				if len(this.inputBuffer) > 8000 {
					this.inputBuffer = make([]byte, 4000)
				}
			}
			//log.Printf("read:%d write:%d\n", this.readIndex, this.writeIndex)
			// 2. read
			readed, err := tcpConn.Read(inputBufferTick)
			if err != nil {
				this.isRunningM.Lock()
				if this.bRun == false {
					log.Printf("client[%s] handleRead exit:[ok]\n", this.id)
					return
				}
				this.isRunningM.Unlock()
				if err == io.EOF {
					log.Printf("client[%s] handleRead exit:[ok]\n", this.id)
				} else {
					log.Printf("client[%s] handleRead exit:[%s]\n", this.id, err.Error())
				}
				this.Close()
				return
			}

			// 3. make buffer bigger
			if this.writeIndex+readed > len(this.inputBuffer) {
				oldInputBuffer := this.inputBuffer[this.readIndex:this.writeIndex]
				this.inputBuffer = make([]byte, len(this.inputBuffer)*2)
				copy(this.inputBuffer, oldInputBuffer)
				this.writeIndex = this.writeIndex - this.readIndex
				this.readIndex = 0
			}

			copy(this.inputBuffer[this.writeIndex:], inputBufferTick[:readed])
			this.writeIndex += readed

			// 4. msg
			for this.writeIndex-this.readIndex > 0 {
				finished, bytesResult, err := ProcessInput(this, this.inputBuffer[this.readIndex:this.writeIndex])
				if err != nil {
					log.Printf("client[%s] handleRead exit:[%s]\n", this.id, err.Error())
					this.Close()
					return
				}

				if finished == 0 {
					break
				}

				this.readIndex += finished
				//log.Printf("done read:%d write:%d\n", this.readIndex, this.writeIndex)

				// # FIXME: may block wan send too slowly
				// 5. send
				this.TrySend(bytesResult)

			}
		}
	}
}

func (this *EasyTcpConn) TrySend(bytesResult []byte) {
	needSendNum := len(bytesResult)
	//log.Println("try send:", needSendNum)

	this.outputBufferMu.Lock()
	defer this.outputBufferMu.Unlock()

	// 1. clear buffer
	if this.outReadIndex == this.outWriteIndex {
		this.outReadIndex = 0
		this.outWriteIndex = 0

		if len(this.outputBuffer) > 8000 {
			this.outputBuffer = make([]byte, 4000)
		}
	}

	if this.outWriteIndex+needSendNum > len(this.outputBuffer) {
		oldOutputBuffer := this.outputBuffer[this.outReadIndex:this.outWriteIndex]
		this.outputBuffer = make([]byte, len(this.outputBuffer)*2)
		copy(this.outputBuffer, oldOutputBuffer)
		this.outWriteIndex = this.outWriteIndex - this.outReadIndex
		this.outReadIndex = 0
	}

	copy(this.outputBuffer[this.outWriteIndex:], bytesResult)
	this.outWriteIndex += needSendNum

	// 2. notify send go
	this.outputChan <- struct{}{}

}

func (this *EasyTcpConn) handleWrite(ProcessOutput EasyOutputHandler) {
	this.wg.Add(1)
	defer this.wg.Done()

	tcpConn, ok := this.rawConn.(*net.TCPConn)
	if !ok {
		return
	}

	var tempSendBuffer []byte

	for {
		select {

		//blocking
		case _ = <-this.outputChan:

			// is need send?
			this.outputBufferMu.Lock()
			snapOutWriteIndex := this.outWriteIndex
			snapOutReadIndex := this.outReadIndex
			needSended := snapOutWriteIndex-snapOutReadIndex > 0
			if needSended {
				if len(tempSendBuffer) < snapOutWriteIndex-snapOutReadIndex {
					tempSendBuffer = make([]byte, snapOutWriteIndex-snapOutReadIndex)
				}
				copy(tempSendBuffer, this.outputBuffer[snapOutReadIndex:snapOutWriteIndex])
			}
			this.outputBufferMu.Unlock()

			// block send
			for needSended {
				newOutputBuffer := tempSendBuffer[:snapOutWriteIndex-snapOutReadIndex]
				err := ProcessOutput(this, newOutputBuffer)
				if err != nil {
					log.Printf("client[%s] handleWrite exit:[%s]\n", this.id, err.Error())
					this.Close()
					return
				}
				//log.Println("here", len(newOutputBuffer))
				sended, err := tcpConn.Write(newOutputBuffer)
				if err != nil {
					this.isRunningM.Lock()
					if this.bRun == false {
						this.isRunningM.Unlock()
						log.Printf("client[%s] handleWrite exit:[ok]\n", this.id)
						return
					}
					this.isRunningM.Unlock()

					log.Printf("client[%s] handleWrite exit:[%s]\n", this.id, err.Error())
					this.Close()
					return
				}

				snapOutReadIndex += sended

				if snapOutWriteIndex-snapOutReadIndex == 0 {
					break
				}
			}

			// done
			if needSended {
				this.outputBufferMu.Lock()
				this.outReadIndex = snapOutReadIndex
				this.outputBufferMu.Unlock()
			}

		case _ = <-this.ctx.Done():
			log.Printf("client[%s] handleWrite exit:[ok]\n", this.id)
			return
		}
	}

}
