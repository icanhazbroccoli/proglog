package discovery

import (
	"log"
	"net"

	"github.com/hashicorp/serf/serf"
)

type Config struct {
	NodeName       string
	BindAddr       *net.TCPAddr
	Tags           map[string]string
	StartJoinAddrs []string
}

type Handler interface {
	Join(name, addr string) error
	Leave(namem, addr string) error
}

type Membership struct {
	Config
	handler Handler
	serf    *serf.Serf
	events  chan serf.Event
}

func New(handler Handler, config Config) (*Membership, error) {
	c := &Membership{
		Config:  config,
		handler: handler,
	}
	if err := c.setupSerf(); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Membership) setupSerf() (err error) {
	config := serf.DefaultConfig()
	config.Init()
	config.MemberlistConfig.BindAddr = s.BindAddr.IP.String()
	config.MemberlistConfig.BindPort = s.BindAddr.Port
	s.events = make(chan serf.Event)
	config.EventCh = s.events
	config.Tags = s.Tags
	config.NodeName = s.Config.NodeName
	s.serf, err = serf.Create(config)
	if err != nil {
		return err
	}
	go s.eventHandler()
	if s.StartJoinAddrs != nil {
		_, err = s.serf.Join(s.StartJoinAddrs, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Membership) eventHandler() {
	for e := range s.events {
		switch e.EventType() {
		case serf.EventMemberJoin:
			for _, m := range e.(serf.MemberEvent).Members {
				if s.isLocal(m) {
					continue
				}
				s.handleJoin(m)
			}
		case serf.EventMemberLeave, serf.EventMemberFailed:
			for _, m := range e.(serf.MemberEvent).Members {
				if s.isLocal(m) {
					return
				}
				s.handleLeave(m)
			}
		}
	}
}

func (s *Membership) handleJoin(m serf.Member) {
	if err := s.handler.Join(
		m.Name,
		m.Tags["rpc_addr"],
	); err != nil {
		log.Printf(
			"[ERROR] proglog: failed to join: %s %s",
			m.Name,
			m.Tags["rpc_addr"],
		)
	}
}

func (s *Membership) handleLeave(m serf.Member) {
	if err := s.handler.Leave(
		m.Name,
		m.Tags["rpc_addr"],
	); err != nil {
		log.Printf(
			"[ERROR] proglog: failed to leave: %s",
			m.Name,
		)
	}
}

func (s *Membership) isLocal(member serf.Member) bool {
	return s.serf.LocalMember().Name == member.Name
}

func (s *Membership) Members() []serf.Member {
	return s.serf.Members()
}

func (s *Membership) Leave() error {
	return s.serf.Leave()
}
