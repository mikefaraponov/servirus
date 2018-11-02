package main

import (
	"github.com/mikefaraponov/chatum"
	"io"
	"time"
)

type Client struct {
	bus Bus
	*ChatumClientDetails
	ponger chan bool
	closer chan error
	pinger *time.Ticker
}

func (c *Client) Listen() {
	for {
		msg, err := c.Recv()
		if err == io.EOF {
			c.closer <- nil
			return
		}
		if err != nil {
			c.closer <- err
			return
		}
		switch msg.GetType() {
		case chatum.EventType_DEFAULT:
			go c.BroadcastExceptSelfUUID(msg.GetMessage())
		case chatum.EventType_PONG:
			c.ponger <- true
		default:
		}
	}
}

func (c *Client) PingPong() error {
	for {
		select {
		case err := <-c.closer:
			return err
		case <-c.pinger.C:
		}

		go c.Send(NewPingMessage())

		select {
		case <-time.After(PingPongTimeout):
			return PingPongTimeoutErr
		case <-c.ponger:
		}
	}
}

func (c *Client) BroadcastExceptSelfUsername(msg string) {
	c.bus.BroadcastExceptUsername(c.newMessage(msg))
}

func (c *Client) BroadcastExceptSelfUUID(msg string) {
	c.bus.BroadcastExceptUUID(c.Id, c.newMessage(msg))
}

func (c *Client) Close() {
	close(c.ponger)
	close(c.closer)
	c.pinger.Stop()
	c.bus.Remove(c)
}

func (c *Client) newMessage(msg string) *chatum.ServerSideEvent {
	return NewMessage(c.Username, msg)
}

func NewClient(b Bus, d *ChatumClientDetails) *Client {
	return &Client{
		ChatumClientDetails: d,
		bus:                 b,
		pinger:              time.NewTicker(PingPongInterval),
		ponger:              make(chan bool, 1),
		closer:              make(chan error, 1),
	}
}
