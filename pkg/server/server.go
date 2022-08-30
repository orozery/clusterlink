package server

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.ibm.com/ei-agent/pkg/client"
	"github.ibm.com/ei-agent/pkg/setupFrame"
)

var (
	maxDataBufferSize = 64 * 1024
)

type SnServer struct {
	Listener      string
	ServiceTarget string
	SnMode        bool
	SnClient      *client.SnClient
}

func (s *SnServer) SrverInit(listener, servicenode string, snmode bool, client *client.SnClient) {
	s.Listener = listener
	s.ServiceTarget = servicenode
	s.SnMode = snmode
	s.SnClient = client
}

func (s *SnServer) RunSrver() {
	fmt.Println("********** Start Server ************")
	fmt.Printf("Strart client listen: %v  send to server: %v \n", s.Listener, s.ServiceTarget)

	err := s.acceptLoop() // missing channel for signal handler
	fmt.Println("Error:", err)
}

func (s *SnServer) acceptLoop() error {
	// open listener
	acceptor, err := net.Listen("tcp", s.Listener)
	if err != nil {
		return err
	}
	// loop until signalled to stop
	for {
		c, err := acceptor.Accept()
		if err != nil {
			return err
		}
		go s.dispatch(c, s.ServiceTarget)
	}
}

func (s *SnServer) dispatch(c net.Conn, servicenode string) error {

	//choose which sevice to pass
	setupPacket := setupFrame.GetSetupPacket(c)
	if s.SnMode {
		if setupPacket.Service.Name == "Forward" {
			s.ServiceTarget = s.SnClient.Listener
			s.SnClient.Target = setupPacket.DestIp + ":" + setupPacket.DestPort
		}
	} else { //incase for regular server forward the traffic to the app destination
		s.ServiceTarget = setupPacket.DestIp + ":" + setupPacket.DestPort
	}

	nodeConn, err := net.Dial("tcp", s.ServiceTarget)

	if err != nil {
		return err
	}
	return s.ioLoop(c, nodeConn)
}

func (s *SnServer) ioLoop(cl, sn net.Conn) error {
	defer cl.Close()
	defer sn.Close()

	fmt.Println("Cient", cl.RemoteAddr().String(), "->", cl.LocalAddr().String())
	fmt.Println("Server", sn.LocalAddr().String(), "->", sn.RemoteAddr().String())
	done := &sync.WaitGroup{}
	done.Add(2)

	go s.clientToServer(done, cl, sn)
	go s.serverToClient(done, cl, sn)

	done.Wait()

	return nil
}

func (s *SnServer) clientToServer(wg *sync.WaitGroup, cl, sn net.Conn) error {

	defer wg.Done()
	bufData := make([]byte, maxDataBufferSize)
	var err error
	for {
		numBytes, err := cl.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				fmt.Printf("[clientToServer]: Read error %v\n", err)
			}

			break
		}

		_, err = sn.Write(bufData[:numBytes])
		if err != nil {
			fmt.Printf("[clientToServer]: Write error %v\n", err)
			break
		}
	}
	if err == io.EOF {
		return nil
	} else {
		return err
	}

}

func (s *SnServer) serverToClient(wg *sync.WaitGroup, cl, sn net.Conn) error {
	defer wg.Done()

	bufData := make([]byte, maxDataBufferSize)
	var err error
	for {
		numBytes, err := sn.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				fmt.Printf("[serverToClient]: Read error %v\n", err)
			}
			break
		}
		_, err = cl.Write(bufData[:numBytes])
		if err != nil {
			fmt.Printf("[serverToClient]: Write error %v\n", err)
			break
		}
	}
	return err
}

//bufSetup := make([]byte, maxSetupBufferSize)

// allocate 4B frame-buffer and 64KB payload buffer
// forever {
//    read 4B into frame-buffer
//    if frame.Type == control { // not expected yet, except for error returns from SN
// 	     read and process control frame
//    } else {
// 	 	 read(sn, payload, frame.Len) // might require multiple reads and need a timeout deadline set
//	     send(cl, payload)
//    }
// }
