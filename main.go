package main

import (
	"flag"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

var (
	TCP_PORT     int
	UDP_PORT     int
	RESOLVER_ADD string

	Help bool
)

func init() {
	flag.BoolVar(&Help, "help", false, "usage help.")
	flag.IntVar(&TCP_PORT, "tcp", 53, "dns-server tcp listen port.")
	flag.IntVar(&UDP_PORT, "udp", 53, "dns-server udp listen port.")
	flag.StringVar(&RESOLVER_ADD, "resolver", "192.168.0.1:53", "dns-server resolver address.")
}

func resolver(m dnsmessage.Message) *dnsmessage.Message {

	conn, err := net.Dial("udp", RESOLVER_ADD)
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	packed, err := m.Pack()
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	_, err = conn.Write(packed)
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	buf := make([]byte, 512)

	_, err = conn.Read(buf)
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	var sm dnsmessage.Message
	err = sm.Unpack(buf)
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	return &sm
}

func tcpprocess(conn net.Conn) {
	defer conn.Close()

}

func main() {

	flag.Parse()
	if Help {
		flag.Usage()
		os.Exit(1)
	}

	tcplis, err := net.ListenTCP("tcp", &net.TCPAddr{Port: 53})
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer tcplis.Close()

	udpconn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 53})
	if err != nil {
		log.Fatalln(err.Error())
	}

	defer udpconn.Close()

	var swit sync.WaitGroup

	swit.Add(2)

	go func() {
		defer swit.Done()
		for {
			conn, err := tcplis.Accept()
			if err != nil {
				log.Println(err.Error())
				time.Sleep(1 * time.Second)
				continue
			}

			go tcpprocess(conn)
		}
	}()

	go func() {
		defer swit.Done()
		for {
			buf := make([]byte, 512)
			_, addr, err := udpconn.ReadFromUDP(buf)
			if err != nil {
				log.Println(err.Error())
				continue
			}

			var m dnsmessage.Message
			err = m.Unpack(buf)
			if err != nil {
				log.Println(err.Error())
				continue
			}

			log.Printf("Message : %v\n", m)

			q := m.Questions[0]

			log.Printf("Question : %d:%s\n", q.Type, q.Name.String())

			//mr := resolver(m)
			//log.Printf("Rsponse : %v\n", mr)

			build := new(dnsmessage.Builder)

			m.Header.Response = true
			build.Start(nil, m.Header)

			err = build.StartQuestions()
			if err != nil {
				log.Println(err.Error())
				continue
			}

			err = build.Question(q)
			if err != nil {
				log.Println(err.Error())
				continue
			}

			A := dnsmessage.AResource{A: [4]byte{192, 168, 3, 1}}
			AS := dnsmessage.ResourceHeader{Name: q.Name, Class: q.Class, Type: q.Type, TTL: 3600}

			err = build.StartAnswers()
			if err != nil {
				log.Println(err.Error())
				continue
			}

			err = build.AResource(AS, A)
			if err != nil {
				log.Println(err.Error())
				continue
			}

			//packed, err := mr.Pack()
			packed, err := build.Finish()
			if err != nil {
				log.Println(err.Error())
				continue
			}

			udpconn.WriteToUDP(packed, addr)
		}
	}()

	swit.Wait()
}
