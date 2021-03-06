package server

import "goraklib/protocol"

type ReceiveWindow struct {
	session *Session

	lowestIndex uint32
	highestIndex uint32
	datagrams map[uint32]*protocol.Datagram

	canFlush bool
}

func NewReceiveWindow(session *Session) *ReceiveWindow {
	return &ReceiveWindow{session, 0, 0, make(map[uint32]*protocol.Datagram), false}
}

func (window *ReceiveWindow) SetHighestIndex(index uint32) {
	window.highestIndex = index
}

func (window *ReceiveWindow) SetLowestIndex(index uint32) {
	window.lowestIndex = index
}

func (window *ReceiveWindow) SubmitDatagram(datagram *protocol.Datagram) {
	window.datagrams[datagram.SequenceNumber] = datagram
	if datagram.SequenceNumber > window.highestIndex {
		window.highestIndex = datagram.SequenceNumber
	}

	for i := window.lowestIndex + 1; i <= window.highestIndex; i++ {
		if _, ok := window.datagrams[i]; !ok {
			break
		}
		if i == window.highestIndex {
			window.canFlush = true
		}
	}
}

func (window *ReceiveWindow) Tick() {
	if window.canFlush {
		window.Release()
		window.datagrams = map[uint32]*protocol.Datagram{}
		return
	}

	var nak = protocol.NewNAK()
	for i := window.lowestIndex + 1; i < window.highestIndex; i++ {
		if _, ok := window.datagrams[i]; ok {
			continue
		}
		nak.Packets = append(nak.Packets, i)
	}
	window.session.SendUnconnectedPacket(nak)
}

func (window *ReceiveWindow) Release() {
	for i := window.lowestIndex; i <= window.highestIndex; i++ {
		window.session.manager.HandleDatagram(window.datagrams[i], window.session)
	}

	for i := window.lowestIndex + 1; i <= window.highestIndex; i++ {
		window.lowestIndex++
	}
}