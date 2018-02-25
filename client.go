package ifrit

import (
	"errors"

	"github.com/joonnna/ifrit/core"
	"github.com/joonnna/ifrit/rpc"
)

type Client struct {
	node *core.Node
}

var (
	errNoData      = errors.New("Supplied data is of length 0")
	errNoCaAddress = errors.New("Config does not contain address of CA")
)

/*
Alternative to registrating callbacks
type MsgHandler interface {
	HandleMsg([]byte) ([]byte, error)
	HandleGossip([]byte) ([]byte, error)
	ResponseHandler([]byte) []byte
}
*/

// Creates and returns a new ifrit client instance.
// Passing an empty config struct or nil will result in all defaults, see config documentation for config description.
func NewClient(conf *Config) (*Client, error) {
	nodeConf, err := parseConfig(conf)
	if err != nil {
		return nil, err
	}

	c := rpc.NewClient()

	s, err := rpc.NewServer()
	if err != nil {
		return nil, err
	}

	n, err := core.NewNode(nodeConf, c, s)
	if err != nil {
		return nil, err
	}

	client := &Client{
		node: n,
	}

	return client, nil
}

// Registers the given function as the message handler.
// Each time the ifrit client receives an application specific message(another client sent it through SendTo/SendToAll/gossipcontent), this callback will be invoked.
// The returned byte slice will be sent back as the response.
// If error is non-nil, it will be returned as the response.
func (c *Client) RegisterMsgHandler(msgHandler func([]byte) ([]byte, error)) {
	c.node.SetMsgHandler(msgHandler)
}

// Registers the given function as the message response handler.
// Will be called when a response is received due to a previous message being sent.
// Only invoked when ifrit receives a response after gossiping application data.
// SendTo/SendToAll receives their responses through the returned channel.
func (c *Client) RegisterResponseHandler(responseHandler func([]byte)) {
	c.node.SetResponseHandler(responseHandler)
}

// Shutsdown the client and all held resources.
func (c *Client) ShutDown() {
	c.node.ShutDownNode()
}

// Client starts operating.
func (c *Client) Start() {
	c.node.Start()
}

// Returns the address(ip:port, rpc endpoint) of all other ifrit clients in the network which is currently
// believed to be alive.
func (c *Client) Members() []string {
	return c.node.LiveMembers()
}

// Sends the given data to the given destination.
// The caller must ensure that the given data is not modified after calling this function.
// The returned channel will be populated with the response.
// If the destination could not be reached or timeout occurs, nil will be sent through the channel.
// The response data can be safely modified after receiving it.
func (c *Client) SendTo(dest string, data []byte) chan []byte {
	ch := make(chan []byte, 1)

	go c.node.SendMessage(dest, ch, data)

	return ch
}

// Sends the given data to all members of the network belivied to be alive.
// The returned channel functions as described in SendTo().
// The returned integer represents the amount of members the message was sent to.
func (c *Client) SendToAll(data []byte) (chan []byte, int) {
	members := c.node.LiveMembers()

	numMembers := len(members)

	//Don't want channel to be too big.
	chSize := int(float32(numMembers) * 0.10)

	if chSize <= 0 {
		chSize = 1
	}

	ch := make(chan []byte, chSize)

	go c.node.SendMessages(members, ch, data)

	return ch, numMembers
}

// Replaces the gossip set with the given data.
// This data will be exchanged with neighbors in each gossip interaction.
// Recipients will receive it through the message handler callback.
// The response generated by the message handler callback will be sent back and
// invoke the response handler callback.
func (c *Client) SetGossipContent(data []byte) error {
	if len(data) <= 0 {
		return errNoData
	}

	c.node.SetExternalGossipContent(data)

	return nil
}

// Returns ifrit's internal ID generated by the trusted CA
func (c *Client) Id() string {
	return c.node.Id()
}

// Returns the address(ip:port) of the ifrit client.
// Can be directly used as entry addresses in the config.
func (c *Client) Addr() string {
	return c.node.Addr()
}

// Returns the address(ip:port) of the ifrit http endpoint.
// Only used for debuging, populated if visualizer is enabled..
func (c *Client) HttpAddr() string {
	return c.node.HttpAddr()
}

// Returns the address(ip:port, http endpoint) of all members of the fireflies
// network believed to be alive.
// Only used for debuging, populated if visualizer is enabled.
func (c *Client) MembersHttp() []string {
	return c.node.LiveMembersHttp()
}
