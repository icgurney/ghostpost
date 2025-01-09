package smtp

import (
	"ghostpost/internal/storage"
	"net"

	"github.com/pires/go-proxyproto"
)

type Server struct {
	port          string
	storage       storage.Storage
	acceptDomains []string
}

func NewServer(port string, storage storage.Storage, acceptDomains []string) *Server {
	return &Server{
		port:          port,
		storage:       storage,
		acceptDomains: acceptDomains,
	}
}

func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.port)
	if err != nil {
		return err
	}
	defer listener.Close()

	proxyListener := &proxyproto.Listener{Listener: listener}
	defer proxyListener.Close()

	for {
		conn, err := proxyListener.Accept()
		if err != nil {
			return err
		}
		go NewHandler(conn, s.storage, s.acceptDomains).Handle()
	}
}
