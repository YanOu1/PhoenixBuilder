package embed

import (
	"crypto/md5"
	"fmt"
	"phoenixbuilder/fastbuilder/environment"
	"phoenixbuilder/fastbuilder/function"
	"phoenixbuilder/fastbuilder/uqHolder"
	"phoenixbuilder/minecraft"
	mc_packet "phoenixbuilder/minecraft/protocol/packet"
	"phoenixbuilder/mirror"
	"phoenixbuilder/omega/defines"
	"phoenixbuilder/omega/mainframe"
	"strings"
	"time"
)

type EmbeddedAdaptor struct {
	env              *environment.PBEnvironment
	BackendCmdFeeder chan string
	PacketFeeder     chan mc_packet.Packet
	ChunkDataFeeder  chan *mirror.ChunkData
}

func (ea *EmbeddedAdaptor) FeedBackendCommand(cmd string) {
	ea.BackendCmdFeeder <- cmd
}

func (ea *EmbeddedAdaptor) GetBackendCommandFeeder() chan string {
	return ea.BackendCmdFeeder
}

func (ea *EmbeddedAdaptor) FeedPacket(pkt mc_packet.Packet) {
	ea.PacketFeeder <- pkt
}

func (ea *EmbeddedAdaptor) GetPacketFeeder() chan mc_packet.Packet {
	return ea.PacketFeeder
}

func (rc *EmbeddedAdaptor) GetInitUQHolderCopy() *uqHolder.UQHolder {
	origHolder := rc.env.UQHolder.(*uqHolder.UQHolder)
	holderBytes := origHolder.Marshal()
	newHolder := uqHolder.NewUQHolder(origHolder.BotRuntimeID)
	err := newHolder.UnMarshal(holderBytes)
	if err != nil {
		return nil
	}
	return newHolder
}

func (rc *EmbeddedAdaptor) Write(pkt mc_packet.Packet) {
	rc.env.Connection.(*minecraft.Conn).WritePacket(pkt)
}

func (rc *EmbeddedAdaptor) FBEval(cmd string) {
	rc.env.FunctionHolder.(*function.FunctionHolder).Process(cmd)
}

func (ea *EmbeddedAdaptor) FeedChunkData(cd *mirror.ChunkData) {
	ea.ChunkDataFeeder <- cd
}

func (ea *EmbeddedAdaptor) GetChunkFeeder() chan *mirror.ChunkData {
	return ea.ChunkDataFeeder
}

func (ea *EmbeddedAdaptor) QuerySensitiveInfo(key defines.SensitiveInfoType) (result string, err error) {
	rawVal := ""
	switch key {
	case defines.SENSITIVE_INFO_SERVER_CODE_HASH:
		rawVal = ea.env.ServerCode
	case defines.SENSITIVE_INFO_USERNAME_HASH:
		_frags := strings.Split(ea.env.FBUCUsername, "|")
		if len(_frags) > 0 {
			rawVal = _frags[0]
		}
	}
	if rawVal == "" {
		return "", fmt.Errorf("no result")
	} else {
		cvt := func(in [16]byte) []byte {
			return in[:16]
		}
		hashedBytes := cvt(md5.Sum([]byte(rawVal)))
		return fmt.Sprintf("%x", hashedBytes), nil
	}
}

func EnableOmegaSystem(env *environment.PBEnvironment) *EmbeddedAdaptor {
	ea := &EmbeddedAdaptor{
		env:              env,
		BackendCmdFeeder: make(chan string, 1024),
		PacketFeeder:     make(chan mc_packet.Packet, 1024),
		ChunkDataFeeder:  make(chan *mirror.ChunkData, 1024),
	}
	fmt.Println("Starting Omega in 1 Seconds")
	time.Sleep(time.Millisecond * 10)
	omega := mainframe.NewOmega()
	omega.Bootstrap(ea)
	env.OmegaHolder = omega
	env.OmegaAdaptorHolder = ea
	env.Destructors = append(env.Destructors, func() {
		omega.Stop()
	})
	go omega.Activate()
	return ea
}
