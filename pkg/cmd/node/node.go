package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/orangeseeds/holepunching/pkg/p2p"
)

var msgRecv chan bool = make(chan bool)

func main() {
	laddr := flag.String("laddr", "127.0.0.1:1111", "laddr")
	relayAddr := flag.String("relayAddr", "127.0.0.1:1112", "relay addr")

	flag.Parse()
	node := p2p.NewNode(*laddr)
	err := node.Listen()
	if err != nil {
		log.Fatal(err)
	}

	err = handshake(node, *relayAddr)
	if err != nil {
		log.Fatal(err)
	}
	for {
		var msg p2p.Message

		_, addr, err := node.ReadMsg(&msg)
		if err != nil {
			log.Println("Error reading msg", err)
			continue
		}

		log.Printf("recv %v from %v\n", msg.Type.String(), addr.String())
		switch msg.Type {
		case p2p.CONN_FOR:
			err = handleCONN_FOR(node, msg, addr)
			if err != nil {
				log.Println("Error handing conn_for: ", err)
			}
		case p2p.ACPT_FOR:
			err = handleACPT_FOR(node, msg, addr)
			if err != nil {
				log.Println("Error handing acpt_for: ", err)
			}
		case p2p.MSG:
			msgRecv <- true
		}
	}
}

func handshake(node *p2p.Node, relayAddr string) error {
	toAddr, err := net.ResolveUDPAddr("udp", relayAddr)
	if err != nil {
		return err
	}

	_, err = node.WriteTo(p2p.Message{
		Type: p2p.SYNC,
		From: node.PublicAddr,
	}, toAddr)
	if err != nil {
		return err
	}

	var msg p2p.Message
	_, _, err = node.ReadMsg(&msg)
	if err != nil {
		return err
	}
	node.PublicAddr = string(msg.Payload)
	log.Println("my addr:", node.PublicAddr)

	err = node.PeerManager.DiscoverPeers(relayAddr)
	if err != nil {
		return err
	}
	//
	// for _, val := range node.PeerManager.Peers {
	// 	fmt.Println(val.Addr)
	// }

	val := ""
	fmt.Println("Enter Value: ")
	fmt.Scanf("%s", &val)
	if val == "" {
		log.Println("Skipping")
		return nil
	}
	log.Println("Selected: ", val)
	connMsg := p2p.Message{
		Type: p2p.CONN,
		From: node.PublicAddr,
	}
	connMsg.InjectPayload(p2p.ConnPayload{
		Addr:   val,
		SentAt: time.Now().UnixNano(),
	})
	_, err = node.WriteTo(connMsg, toAddr)
	if err != nil {
		return err
	}
	return nil
}

func handleCONN_FOR(node *p2p.Node, msg p2p.Message, addr net.Addr) error {
	var connPayload p2p.ConnPayload
	err := msg.DecodeConnPayload(&connPayload)
	if err != nil {
		return err
	}
	reply := p2p.Message{
		Type: p2p.ACPT,
		From: node.PublicAddr,
	}

	roundTime := time.Now().UnixNano() - connPayload.SentAt

	reply.InjectPayload(p2p.ConnPayload{
		Addr:   msg.From,
		SentAt: time.Now().UnixNano(),
	})

	_, err = node.WriteTo(reply, addr)
	if err != nil {
		return err
	}

	log.Println("Waiting for t/2: ", roundTime/2)
	<-time.After(time.Duration(roundTime / 2))

	for {
		select {
		case <-msgRecv:
			return nil
		default:
			<-time.After(time.Duration(rand.Intn(201) * int(time.Millisecond)))
			toAddr, err := net.ResolveUDPAddr("udp", msg.From)
			if err != nil {
				return err
			}
			log.Println("Sent payload from", node.Listener.LocalAddr().String(), "to", toAddr.String())
			_, err = node.WriteTo(p2p.Message{
				Type:    p2p.MSG,
				From:    node.PublicAddr,
				Payload: []byte("Hello"),
			}, toAddr)
			if err != nil {
				log.Println(err)
				return err
			}
		}
	}
}

func handleACPT_FOR(node *p2p.Node, msg p2p.Message, addr net.Addr) error {
	var connPayload p2p.ConnPayload
	err := msg.DecodeConnPayload(&connPayload)
	if err != nil {
		return err
	}

	toAddr, err := net.ResolveUDPAddr("udp", msg.From)
	if err != nil {
		return err
	}
	log.Println("Sent payload from", node.Listener.LocalAddr().String(), "to", toAddr.String())
	_, err = node.WriteTo(p2p.Message{
		Type:    p2p.MSG,
		From:    node.PublicAddr,
		Payload: []byte("Hello"),
	}, toAddr)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
