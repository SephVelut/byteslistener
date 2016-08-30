package byteslistener

import (
	"io/ioutil"
	"net"
	"sync"
)

var mu = &sync.Mutex{}

type Tcp struct {
	address  string
	port     string
	conn     net.Listener
	conns    map[int]net.Conn
	quit     chan int
	handlers []func([]byte)
}

func (this Tcp) New(address string, port string) *Tcp {
	return &Tcp{address: address,
		port:     port,
		conns:    map[int]net.Conn{},
		quit:     make(chan int, 1),
		handlers: []func([]byte){}}
}

func (this *Tcp) Subscribe(callback func([]byte)) {
	this.handlers = append(this.handlers, callback)
}

func (this *Tcp) Listen() {
	if this.conn == nil {
		var err error
		this.conn, err = net.Listen("tcp", this.address+":"+this.port)
		this.handleError(err)

		go this.handleRequests()
	}
}

func (this *Tcp) Close() {
	// If more than one connection is closed, than the quit channel will need to be triggered
	// for each close or the fail-safe switch will will not work after the first pass
	this.quit <- 0
	this.conn.Close()

	mu.Lock()
	for _, conn := range this.conns {
		conn.Close()
	}
	mu.Unlock()
}

func (this *Tcp) handleRequests() {
	for {
		request, err := this.conn.Accept()
		if err != nil {
			select {
			case <-this.quit:
				return
			default:
				this.handleError(err)
			}
		}
		mu.Lock()
		var key = len(this.conns)
		this.conns[key] = request
		mu.Unlock()

		go func() {
			this.handleRequest(request)

			mu.Lock()
			request.Close()
			delete(this.conns, key)
			mu.Unlock()
		}()
	}
}

func (this *Tcp) handleRequest(request net.Conn) {
	buf, err := ioutil.ReadAll(request)
	this.handleError(err)

	for _, handler := range this.handlers {
		handler(buf)
	}
}

func (this *Tcp) handleError(err error) {
	if err != nil {
		panic(err)
	}
}
