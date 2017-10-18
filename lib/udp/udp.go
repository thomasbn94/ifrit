package udp

import (
	"fmt"
	"net"
	"time"

	"github.com/joonnna/firechain/lib/netutils"
	"github.com/joonnna/firechain/logger"
)

type Server struct {
	log  *logger.Log
	conn *net.UDPConn
	addr string
}

func NewServer(log *logger.Log) (*Server, error) {
	port := netutils.GetOpenPort()
	hostName := netutils.GetLocalIP()

	addr := fmt.Sprintf("%s:%d", hostName, port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	return &Server{log: log, conn: conn, addr: addr}, nil
}

func (s Server) Send(addr string, data []byte) ([]byte, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		s.log.Err.Println(err)
		return nil, err
	}

	c, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		s.log.Err.Println(err)
		return nil, err
	}
	c.SetDeadline(time.Now().Add(time.Second * 3))
	defer c.Close()

	_, err = c.Write(data)
	if err != nil {
		s.log.Err.Println(err)
		return nil, err
	}

	bytes := make([]byte, 256)

	n, err := c.Read(bytes)
	if err != nil {
		s.log.Err.Println(err)
		return nil, err
	}

	return bytes[:n], nil
}

func (s *Server) Serve(signMsg func([]byte) ([]byte, error), exitChan chan bool) error {
	for {
		select {
		case <-exitChan:
			return nil
		default:
			bytes := make([]byte, 256)

			n, addr, err := s.conn.ReadFrom(bytes)
			if err != nil {
				s.log.Err.Println(err)
				continue
			}

			resp, err := signMsg(bytes[:n])
			if err != nil {
				s.log.Err.Println(err)
				continue

			}

			_, err = s.conn.WriteTo(resp, addr)
			if err != nil {
				s.log.Err.Println(err)
				continue
			}
		}
	}

	return nil
}

func (s Server) Addr() string {
	return s.addr
}

func (s *Server) Shutdown() {
	s.conn.Close()
}
