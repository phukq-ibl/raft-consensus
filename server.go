package mynetwork

import (
	"errors"
	"github.com/phukq/raft-consensus/types"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"sync"
	"time"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	log.SetLevel(log.TraceLevel)
}

type Server struct {
	Id          string
	CurrentTerm uint64
	VotedFor    string
	Peers       []*Peer
	Listener    net.Listener
	ListenAddr  string

	CommitIndex      uint64
	LastAppliedIndex uint64

	chNewElection     chan int
	chCampaignTimeout chan int
	chBecomeLeader    chan int
	chVoteRequest     chan int
	chNewPeer         chan *Peer
	chHeartbeatMsg    chan types.Msg
	chRequestVote     chan types.Msg
	chVote            chan types.AppendEntries

	mutex sync.Mutex

	totalVote int8
	Type      types.ServerType
}

func NewServer(id string, listenAddr string) Server {
	srv := Server{}
	srv.Id = id
	srv.ListenAddr = listenAddr
	srv.chNewElection = make(chan int)
	srv.chCampaignTimeout = make(chan int)
	srv.chBecomeLeader = make(chan int)
	srv.chNewPeer = make(chan *Peer)
	srv.chVote = make(chan types.AppendEntries)
	srv.chVoteRequest = make(chan int)
	srv.chHeartbeatMsg = make(chan types.Msg)
	srv.Peers = make([]*Peer, 0, 10)
	srv.mutex = sync.Mutex{}
	return srv
}

func (s Server) AddNode(ip string) error {
	return nil
}

func (s Server) Start() error {
	go s.handleConnection()
	go s.run()
	return nil
}

func (s Server) run() {
	log.Info("Server run")
	timer := time.NewTimer(types.ElectionTimeout)

	for {
		select {
		case <-timer.C:
			// log.Debug("Heartbeat timeout")
			if s.Type == types.Follower {
				go s.startNewElection()
			} else if s.Type == types.Leader {
				go s.broadcastHeartBeatMsg()
				timer.Stop()
				timer.Reset(types.HeartbeatTimeout)
			}
		case <-s.chCampaignTimeout:
			s.mutex.Lock()
			s.Type = types.Follower
			s.VotedFor = ""
			s.totalVote = 0
			s.mutex.Unlock()
			timer.Stop()
			timer.Reset(types.ElectionTimeout)
		case <-s.chNewElection:
			s.mutex.Lock()
			s.Type = types.Candidate
			s.VotedFor = s.Id
			s.totalVote = 1
			s.mutex.Unlock()
			// Send request voted to other peer
			s.broadcastRequestVote()
			// Start waiting for voting
			go s.campaignWait()
			//TODO: should we stop electiontimeout?
		case p := <-s.chNewPeer:
			s.addPeer(p)
			go p.Run(s.Handle)
		case <-s.chBecomeLeader:
			s.mutex.Lock()
			s.VotedFor = ""
			s.Type = types.Leader
			s.mutex.Unlock()
			log.Info("Become leader, start send heartbeat msg")
			go s.broadcastHeartBeatMsg()
			timer.Stop()
			timer.Reset(types.HeartbeatTimeout)

		case <-s.chHeartbeatMsg:
			log.Debug("Heartbeat msg ")
			timer.Stop()
			timer.Reset(types.ElectionTimeout)

			s.mutex.Lock()
			if s.Type != types.Leader {
				s.Type = types.Follower
			}
			s.VotedFor = ""
			s.mutex.Unlock()

		}
	}
}

func (s Server) handleConnection() {
	err := s.setupListen()

	if err != nil {
		fmt.Println("Can not setup listen")
		panic(err)
	}

	// Handle new connection
	for {
		conn, err := s.Listener.Accept()

		if err != nil {
			fmt.Println("Error 2")
			// panic(err)

		} else {
			go s.setupConn(conn)
		}
	}
}

func (s *Server) setupListen() error {
	listener, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return err
	}
	s.Listener = listener
	return nil
}

func (s *Server) setupConn(conn net.Conn) {
	// do handshake
	peer := NewPeer(conn)
	err := s.doHandShake(peer)
	if err != nil {
		log.Warn("Handshake fail")
		log.Warn(err)
	}

}

func (s *Server) doHandShake(p *Peer) error {
	fmt.Println("Handshake to ", p.RemoteAddr())
	tick := time.NewTimer(types.HandshakeTimeout)

	msg := types.NewHandshakeMsg(s.Id)
	p.SendMsg(msg.ToString())

	readChan := make(chan string)

	go func() {
		log.Debug("Read data from connection")
		status, err := p.ReadMsg()
		if err != nil {
			readChan <- "error"
		} else {
			readChan <- status
		}
		log.Debug("Read data from finished")
	}()

	for {
		select {
		case <-tick.C:
			return errors.New("Timeout")
		case sMsg := <-readChan:
			log.Debug("Receive handshake msg ", msg)
			tick.Stop()
			msg, err := types.MsgFromString(sMsg)
			if err != nil || msg.Type != types.Handshake {
				return errors.New("Receive msg before handshake")
			}
			ae, err := types.AppendEntriesFromJSONString(msg.Data)

			if err != nil {
				return errors.New("Handshake msg is wrong format: " + sMsg)
			}
			if ae.LeaderId == s.Id {
				return errors.New("Peer id must be diffirent with : " + s.Id)
			}
			p.Id = ae.LeaderId
			s.chNewPeer <- p
			return nil
		}
	}
}

func (s *Server) startNewElection() {
	// Create random in range 150-300ms
	rand.Seed(time.Now().UnixNano())
	timeout := rand.Intn(150) + 150
	timer := time.NewTimer(time.Millisecond * time.Duration(timeout))
	log.Error("Start new election campaign after ", timeout, "mil")
	for {
		select {
		case <-timer.C:
			// Send Request votes to all peer
			s.chNewElection <- 1
		case <-s.chVoteRequest:
			// Cancel new election campaign if we receipt a request vote
			if !timer.Stop() {
				<-timer.C
			}
		}
	}
}

func (s *Server) broadcastHeartBeatMsg() {
	log.Debug("Broadcast heartbeat msg to ", len(s.Peers), " peer(s)")
	msg := types.NewHeartBeatMsg(s.Id)
	s.broadcast(msg.ToString())
}

func (s *Server) addPeer(p *Peer) {
	log.Debug("Add peer ", p.RemoteAddr(), p.Id)
	s.mutex.Lock()
	s.Peers = append(s.Peers, p)
	s.mutex.Unlock()
}

func (s Server) AddPeer(ip string) error {
	conn, err := net.Dial("tcp", ip)
	if err != nil {
		return err
	}

	p := NewPeer(conn)

	return s.doHandShake(p)

}

func (s *Server) Handle(p *Peer) error {
	log.Info("Start handle peer addr=", p.RemoteAddr(), ",id=", p.Id)

	for {
		msgRaw, err := p.ReadMsg()

		if err != nil {
			fmt.Println("Err", err)
			p.Disconnect()
			s.RemovePeer(p.Id)
			return err
		}

		log.Debug("Receive msg: ", msgRaw)
		// if msgRaw != "" {
		// }

		msg, err := types.MsgFromString(msgRaw)

		if err != nil {
			return errors.New("Msg format is not correct: " + msgRaw)
		}

		switch msg.Type {
		case types.RequestVote:
			s.handleRequestVoteMsg(p, msg)
		case types.VoteFor:
			s.handleVoteMsg(p, msg)
		case types.Heartbeat:
			s.chHeartbeatMsg <- msg
		}
	}
}

func (s *Server) handleRequestVoteMsg(p *Peer, msg types.Msg) {
	log.Debug("Handle request vote msg ")

	ae, _ := types.AppendEntriesFromJSONString(msg.Data)

	if s.Type != types.Follower {
		log.Debug("Server is not follower --> Doesn't vote")
		return
	}

	if s.VotedFor != "" {
		log.Debug("Server has already vote for ", s.VotedFor)
		return
	}

	s.chVoteRequest <- 1
	s.VotedFor = ae.LeaderId
	voteMsg := types.NewVoteForMsg(ae.LeaderId)
	log.Debug("Send vote for ", ae.LeaderId)
	p.SendMsg(voteMsg.ToString())
}

func (s Server) handleVoteMsg(p *Peer, msg types.Msg) {
	log.Debug("Handle vote msg ", msg.ToString())
	ae, _ := types.AppendEntriesFromJSONString(msg.Data)
	if ae.LeaderId == s.Id {
		s.chVote <- ae
	}
}

func (s Server) broadcastRequestVote() {
	log.Debug("Broadcast vote request to all peers")
	rq := types.NewRequestVoteMsg(s.Id)
	s.broadcast(rq.ToString())
}

func (s Server) broadcast(msg string) {
	for _, p := range s.Peers {
		go p.SendMsg(msg)
	}
}

func (s Server) campaignWait() {
	timer := time.NewTimer(types.CampaignTimeout)

	for {
		select {
		case <-timer.C:
			log.Debug("Campaign timeout, total vote is: ", s.totalVote)
			if s.totalVote < types.MinVote {
				log.Debug("total vote does not exceed min vote number ", types.MinVote)
				s.chCampaignTimeout <- 1
			} else {
				log.Debug("total vote exceeds min vote number")
				s.chBecomeLeader <- 1
			}
		case <-s.chVote:
			log.Debug("Receive vote msg")
			s.mutex.Lock()
			s.totalVote++
			if s.totalVote >= types.MinVote {
				if !timer.Stop() {
					// <-timer.C
				}
				s.chBecomeLeader <- 1
			}
			s.mutex.Unlock()
		}

	}
}

func (s *Server) RemovePeer(id string) {
	s.mutex.Lock()
	peerIdx := -1
	for i := 0; i < len(s.Peers); i++ {
		if s.Peers[i].Id == id {
			peerIdx = i
			break
		}
	}
	if peerIdx >= 0 {
		log.Debug("Remove peer ip=", s.Peers[peerIdx].RemoteAddr, ", id=", id)
		s.Peers[peerIdx] = s.Peers[len(s.Peers)-1]
		s.Peers = s.Peers[:len(s.Peers)-1]
	}
	s.mutex.Unlock()
}
func (s Server) GetCurrentState() types.AppendEntries {
	ae := types.AppendEntries{}
	ae.Term = s.CurrentTerm
	return ae
}
