package types

import "time"
import "encoding/json"

const (
	ElectionTimeout  time.Duration = 8 * time.Second
	HandshakeTimeout time.Duration = 15 * time.Second
	CampaignTimeout  time.Duration = 5 * time.Second
	HeartbeatTimeout time.Duration = 3 * time.Second
)

const (
	MinVote int8 = 3
)

type ServerType int8

const (
	Follower ServerType = iota
	Candidate
	Leader
)

type MsgType int8

const (
	Handshake MsgType = iota
	RequestVote
	VoteFor
	Heartbeat
)

type AppendEntries struct {
	Term              uint64
	LeaderId          string
	PrevLogIndex      uint64
	PrevLogTerm       uint64
	LeaderCommitIndex uint64
}

func (ae AppendEntries) ToString() string {
	b, err := json.Marshal(ae)
	if err != nil {
		panic(err)
	}
	rs := string(b)
	return rs
}

func AppendEntriesFromJSONString(strMsg string) (AppendEntries, error) {
	ae := AppendEntries{}
	err := json.Unmarshal([]byte(strMsg), &ae)

	return ae, err
}

type Msg struct {
	Type   MsgType
	Data   string
	FromId string
}

func NewHandshakeMsg(id string) Msg {
	msg := Msg{}
	msg.Type = Handshake
	msg.FromId = id
	// ae := AppendEntries{}
	// ae.LeaderId = leaderId
	// msg.Data = ae.ToString()
	return msg
}

func NewRequestVoteMsg(leaderId string) Msg {
	msg := Msg{}
	msg.Type = RequestVote
	msg.FromId = leaderId
	ae := AppendEntries{}
	ae.LeaderId = leaderId
	msg.Data = ae.ToString()
	return msg
}
func NewVoteForMsg(leaderId string) Msg {
	msg := Msg{}
	msg.Type = VoteFor
	ae := AppendEntries{}
	ae.LeaderId = leaderId
	msg.Data = ae.ToString()
	return msg
}

func NewHeartBeatMsg(leaderId string) Msg {
	msg := Msg{}
	msg.Type = Heartbeat
	msg.FromId = leaderId
	// ae := AppendEntries{}
	// ae.LeaderId = leaderId
	// msg.Data = ae.ToString()
	return msg
}

func (msg Msg) ToString() string {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	rs := string(b)
	return rs
}

func MsgFromString(strMsg string) (Msg, error) {
	msg := Msg{}
	err := json.Unmarshal([]byte(strMsg), &msg)
	return msg, err
}
