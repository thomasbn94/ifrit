package node

// Basic imports
import (
	"crypto/ecdsa"
	"testing"

	"github.com/joonnna/capstone/logger"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type NodeTestSuite struct {
	suite.Suite
	*Node
}

func newTestPeer(id string, numRings uint8, addr string) (*peer, *ecdsa.PrivateKey) {
	var i uint8

	peerPrivKey, err := genKeys()
	if err != nil {
		panic(err)
	}

	p := &peer{
		publicKey: &peerPrivKey.PublicKey,
		peerId:    newPeerId([]byte(id)),
		addr:      addr,
	}

	localNote := &note{
		epoch:  1,
		mask:   make([]byte, numRings),
		peerId: p.peerId,
	}

	for i = 0; i < numRings; i++ {
		localNote.mask[i] = 1
	}

	_, err = localNote.signAndMarshal(peerPrivKey)
	if err != nil {
		panic(err)
	}

	p.recentNote = localNote

	return p, peerPrivKey
}

// Make sure that VariableThatShouldStartAtFive is set to five
// before each test
func (suite *NodeTestSuite) SetupTest() {
	var numRings, i uint8

	numRings = 3

	p, priv := newTestPeer("mainPeer1234", numRings, "localhost:1000")

	n := &Node{
		viewMap:    make(map[string]*peer),
		liveMap:    make(map[string]*peer),
		timeoutMap: make(map[string]*timeout),
		ringMap:    make(map[uint8]*ring),
		numRings:   numRings,
		peer:       p,
		privKey:    priv,
		log:        logger.CreateLogger("test", "testlog"),
	}

	for i = 0; i < n.numRings; i++ {
		n.ringMap[i] = newRing(i, n.id, n.key, n.addr)
	}

	suite.Node = n
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestNodeTestSuite(t *testing.T) {
	suite.Run(t, new(NodeTestSuite))
}
