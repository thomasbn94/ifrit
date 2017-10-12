package node

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/joonnna/capstone/protobuf"
	"github.com/stretchr/testify/assert"
)

func newPbNote(n *note, priv *ecdsa.PrivateKey) *gossip.Note {
	noteMsg := &gossip.Note{
		Epoch: n.epoch,
		Id:    n.id,
		Mask:  n.mask,
	}

	b := []byte(fmt.Sprintf("%v", noteMsg))
	signature, err := signContent(b, priv)
	if err != nil {
		panic(err)
	}

	n.signature = signature

	noteMsg.Signature = &gossip.Signature{
		R: signature.r,
		S: signature.s,
	}

	return noteMsg
}

func newPbAcc(a *accusation, priv *ecdsa.PrivateKey) *gossip.Accusation {
	acc := &gossip.Accusation{
		Epoch:   a.epoch,
		Accused: a.id,
		Mask:    a.mask,
		Accuser: a.accuser.id,
	}

	b := []byte(fmt.Sprintf("%v", acc))
	signature, err := signContent(b, priv)
	if err != nil {
		panic(err)
	}

	a.signature = signature

	acc.Signature = &gossip.Signature{
		R: signature.r,
		S: signature.s,
	}

	return acc
}

func (suite *NodeTestSuite) TestAddValidNote() {
	p, priv := newTestPeer("test1234", suite.numRings, "localhost:123")

	prevNote := p.getNote()

	newNote := &note{
		epoch:  prevNote.epoch + 1,
		mask:   prevNote.mask,
		peerId: p.peerId,
	}

	noteMsg := newPbNote(newNote, priv)

	suite.addViewPeer(p)

	suite.evalNote(noteMsg)

	assert.NotNil(suite.T(), p.getNote(), "Valid note not added to peer, eval failed")
	assert.NotNil(suite.T(), suite.getLivePeer(p.key), "Valid note not added to peer, eval failed")
	assert.Equal(suite.T(), p.getNote().epoch, newNote.epoch, "New note with higher epoch does not replace old one")
}

func (suite *NodeTestSuite) TestAddInvalidNote() {
	p, priv := newTestPeer("test1234", suite.numRings, "localhost:123")

	prevNote := p.getNote()

	newNote := &note{
		epoch:  prevNote.epoch - 1,
		mask:   prevNote.mask,
		peerId: p.peerId,
	}

	noteMsg := newPbNote(newNote, priv)

	suite.addViewPeer(p)

	suite.evalNote(noteMsg)

	assert.NotEqual(suite.T(), p.getNote().epoch, newNote.epoch, "Invalid note does not replace a valide one")
}

func (suite *NodeTestSuite) TestAddNoteToNonExistingPeer() {
	p, priv := newTestPeer("test1234", suite.numRings, "localhost:123")

	prevNote := p.getNote()

	newNote := &note{
		epoch:  prevNote.epoch + 1,
		mask:   prevNote.mask,
		peerId: p.peerId,
	}

	noteMsg := newPbNote(newNote, priv)

	suite.evalNote(noteMsg)

	assert.NotEqual(suite.T(), p.getNote().epoch, newNote.epoch, "Note was added to non-existing peer")
}

func (suite *NodeTestSuite) TestInvalidRebuttal() {
	p, priv := newTestPeer("test1234", suite.numRings, "localhost:123")

	prevNote := p.getNote()

	a := &accusation{
		epoch: prevNote.epoch + 2,
	}

	err := p.setAccusation(a)
	assert.Nil(suite.T(), err)

	newNote := &note{
		epoch:  a.epoch - 1,
		mask:   prevNote.mask,
		peerId: p.peerId,
	}

	noteMsg := newPbNote(newNote, priv)

	suite.addViewPeer(p)

	suite.evalNote(noteMsg)

	assert.NotEqual(suite.T(), p.getNote().epoch, newNote.epoch, "Invalid note does replaces a valid one")
}

func (suite *NodeTestSuite) TestValidRebuttal() {
	p, priv := newTestPeer("test1234", suite.numRings, "localhost:123")

	prevNote := p.getNote()

	a := &accusation{
		epoch: prevNote.epoch + 1,
	}

	err := p.setAccusation(a)
	assert.Nil(suite.T(), err)

	newNote := &note{
		epoch:  a.epoch + 1,
		mask:   prevNote.mask,
		peerId: p.peerId,
	}

	noteMsg := newPbNote(newNote, priv)

	suite.addViewPeer(p)

	suite.evalNote(noteMsg)

	assert.Equal(suite.T(), p.getNote().epoch, newNote.epoch, "Valid note does not act as rebuttal")
}

func (suite *NodeTestSuite) TestAccusedNotInView() {
	accused, _ := newTestPeer("accused1234", suite.numRings, "localhost:123")

	n := accused.getNote()

	acc := &accusation{
		epoch:   n.epoch + 1,
		mask:    n.mask,
		peerId:  n.peerId,
		accuser: suite.peerId,
	}

	accMsg := newPbAcc(acc, suite.privKey)

	suite.evalAccusation(accMsg)

	assert.Nil(suite.T(), accused.getAccusation(), "Added accusation for peer not in view")
}

func (suite *NodeTestSuite) TestAccuserNotInView() {
	accused, _ := newTestPeer("accused1234", suite.numRings, "localhost:123")

	accuser, priv := newTestPeer("accuser1234", suite.numRings, "localhost:124")

	suite.addViewPeer(accused)

	n := accused.getNote()

	acc := &accusation{
		epoch:   n.epoch + 1,
		mask:    n.mask,
		peerId:  n.peerId,
		accuser: accuser.peerId,
	}

	accMsg := newPbAcc(acc, priv)

	suite.evalAccusation(accMsg)

	assert.Nil(suite.T(), accused.getAccusation(), "Added accusation when accuser not in view")
}

func (suite *NodeTestSuite) TestValidAccStartsTimer() {
	accused, _ := newTestPeer("accused1234", suite.numRings, "localhost:123")

	accuser, priv := newTestPeer("accuser1234", suite.numRings, "localhost:124")

	suite.addViewPeer(accused)
	suite.addViewPeer(accuser)

	n := accused.getNote()

	acc := &accusation{
		epoch:   n.epoch + 1,
		mask:    n.mask,
		peerId:  accused.peerId,
		accuser: accuser.peerId,
	}

	accMsg := newPbAcc(acc, priv)

	suite.evalAccusation(accMsg)

	assert.True(suite.T(), suite.timerExist(accused.key), "Valid accusation does not start timer")
}
