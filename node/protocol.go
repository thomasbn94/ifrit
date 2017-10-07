package node

import (
	"crypto/x509"
	"fmt"

	"github.com/joonnna/capstone/protobuf"
)

type correct struct {
}

type spamAccusations struct {
}

func (c correct) Rebuttal(n *Node) {
	neighbours := n.getNeighbours()

	msg := &gossip.GossipMsg{}

	noteMsg := &gossip.Note{
		Epoch: n.getEpoch(),
		Id:    n.id,
	}

	b := []byte(fmt.Sprintf("%v", noteMsg))
	signature, err := signContent(b, n.privKey)
	if err != nil {
		n.log.Err.Println(err)
		return
	}

	noteMsg.Signature = &gossip.Signature{
		R: signature.r,
		S: signature.s,
	}

	msg.Notes = append(msg.Notes, noteMsg)

	for _, addr := range neighbours {
		if addr == n.localAddr {
			continue
		}
		_, err := n.client.Gossip(addr, msg)
		if err != nil {
			n.log.Err.Println(err)
			continue
		}
	}
}

func (c correct) Gossip(n *Node) {
	msg, err := n.collectGossipContent()
	if err != nil {
		return
	}
	neighbours := n.getNeighbours()

	for _, addr := range neighbours {
		if addr == n.localAddr {
			continue
		}
		reply, err := n.client.Gossip(addr, msg)
		if err != nil {
			certs := reply.GetCertificates()
			if certs == nil {
				n.log.Err.Println(err)
				continue
			} else {
				for _, c := range certs {
					cert, err := x509.ParseCertificate(c.GetRaw())
					if err != nil {
						n.log.Err.Println(err)
						return
					}
					p, err := newPeer(nil, cert)
					if err != nil {
						n.log.Err.Println(err)
						return
					}

					n.addViewPeer(p)
					n.addLivePeer(p)

				}
			}

		}
	}
}

func (c correct) Monitor(n *Node) {
	msg := &gossip.Ping{}

	for _, ring := range n.ringMap {
		succ, err := ring.getRingSucc()
		if err != nil {
			n.log.Err.Println(err)
			continue
		}

		if succ.addr == n.localAddr {
			continue
		}

		//TODO maybe udp?
		_, err = n.client.Monitor(succ.addr, msg)
		if err != nil {
			n.log.Info.Printf("%s is dead, accusing", succ.addr)
			p := n.getViewPeer(succ.peerKey)
			if p != nil {
				peerNote := p.getNote()
				if peerNote == nil {
					return
				}

				tmp := &gossip.Accusation{
					Epoch:   (peerNote.epoch + 1),
					Accuser: n.peerId.id,
					Accused: p.peerId.id,
				}

				b := []byte(fmt.Sprintf("%v", tmp))
				signature, err := signContent(b, n.privKey)
				if err != nil {
					n.log.Err.Println(err)
					return
				}

				a := &accusation{
					peerId:    p.peerId,
					epoch:     (peerNote.epoch + 1),
					accuser:   n.peerId,
					signature: signature,
				}

				err = p.setAccusation(a)
				if err != nil {
					n.log.Err.Println(err)
					return
				}
				if !n.timerExist(p.key) {
					//TODO insert local peer in node struct?
					n.startTimer(p.key, peerNote, nil, p.addr)
				}
			}
		}
	}
}

func (sa spamAccusations) Gossip(n *Node) {
	msg, err := createFalseAccusations(n)
	if err != nil {
		n.log.Err.Println(err)
		return
	}

	allNodes := n.getView()

	for _, p := range allNodes {
		_, err := n.client.Gossip(p.addr, msg)
		if err != nil {
			n.log.Err.Println(err)
			continue
		}
	}
}

func (sa spamAccusations) Monitor(n *Node) {
	for _, ring := range n.ringMap {
		succ, err := ring.getRingSucc()
		if err != nil {
			n.log.Err.Println(err)
			continue
		}

		p := n.getViewPeer(succ.peerKey)
		if p != nil {
			peerNote := p.getNote()
			if peerNote == nil {
				return
			}

			a := &accusation{
				peerId:  p.peerId,
				epoch:   (peerNote.epoch + 1),
				accuser: n.peerId,
			}

			b := []byte(fmt.Sprintf("%v", a))
			signature, err := signContent(b, n.privKey)
			if err != nil {
				n.log.Err.Println(err)
				return
			}
			a.signature = signature

			err = p.setAccusation(a)
			if err != nil {
				n.log.Err.Println(err)
				return
			}

			if !n.timerExist(p.key) {
				n.startTimer(p.key, peerNote, nil, p.addr)
			}
		}
	}
}

func (sa spamAccusations) Rebuttal(n *Node) {
	neighbours := n.getNeighbours()

	msg := &gossip.GossipMsg{}

	noteMsg := &gossip.Note{
		Id:    n.id,
		Epoch: n.getEpoch(),
	}

	b := []byte(fmt.Sprintf("%v", noteMsg))
	signature, err := signContent(b, n.privKey)
	if err != nil {
		n.log.Err.Println(err)
		return
	}

	noteMsg.Signature = &gossip.Signature{
		R: signature.r,
		S: signature.s,
	}

	msg.Notes = append(msg.Notes, noteMsg)

	for _, addr := range neighbours {
		if addr == n.localAddr {
			continue
		}
		_, err := n.client.Gossip(addr, msg)
		if err != nil {
			n.log.Err.Println(err)
			continue
		}
	}
}

func createFalseAccusations(n *Node) (*gossip.GossipMsg, error) {
	msg := &gossip.GossipMsg{}

	view := n.getView()

	for _, p := range view {
		peerNote := p.getNote()
		if peerNote == nil {
			continue
		}

		a := &gossip.Accusation{
			Accuser: n.id,
			Epoch:   (peerNote.epoch + 1),
			Accused: peerNote.id,
		}

		b := []byte(fmt.Sprintf("%v", a))
		signature, err := signContent(b, n.privKey)
		if err != nil {
			n.log.Err.Println(err)
			return nil, err
		}

		a.Signature = &gossip.Signature{
			R: signature.r,
			S: signature.s,
		}

		msg.Accusations = append(msg.Accusations, a)
	}

	noteMsg := &gossip.Note{
		Id:    n.id,
		Epoch: n.getEpoch(),
	}

	b := []byte(fmt.Sprintf("%v", noteMsg))
	signature, err := signContent(b, n.privKey)
	if err != nil {
		return nil, err
	}

	noteMsg.Signature = &gossip.Signature{
		R: signature.r,
		S: signature.s,
	}

	msg.Notes = append(msg.Notes, noteMsg)

	return msg, nil
}