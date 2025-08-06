// T_CONNECT and T_DISCONNECT requests are part of the Device Management
// service family and are used to establish and terminate point-to-point
// connections to KNX devices. See KNX Standard 03_08_03 Management.

package cemi

// ControlConn represents a T_CONNECT ControlData structure.
type ControlConn struct {
	ControlData
}

// TConnect creates a new T_CONNECT ControlData structure.
func TConnect() *ControlConn {
	return &ControlConn{
		ControlData: ControlData{
			Numbered:  false,
			SeqNumber: 0,
			Command:   uint8(Connect),
		},
	}
}

// NewConnReq creates a new L_Data.req message with a T_CONNECT transport control field
// using the specified source and destination addresses.
func NewConnReq(src, dst IndividualAddr) *LDataReq {
	ctrl := TConnect()

	ldata := LData{
		Control1:    Control1StdFrame | Control1NoRepeat | Control1NoSysBroadcast,
		Control2:    Control2Hops(6),
		Source:      src,
		Destination: uint16(dst),
		Data:        ctrl,
	}

	return &LDataReq{
		LData: ldata,
	}
}

// ControlDisc represents a T_DISCONNECT ControlData structure.
type ControlDisc struct {
	ControlData
}

// TDisconnect creates a new T_DISCONNECT ControlData structure.
func TDisconnect() *ControlDisc {
	return &ControlDisc{
		ControlData: ControlData{
			Numbered:  false,
			SeqNumber: 0,
			Command:   uint8(Disconnect),
		},
	}
}

// NewDiscReq creates a new L_Data.req message with a T_DISCONNECT transport control field
// using the specified source and destination addresses.
func NewDiscReq(src, dst IndividualAddr) *LDataReq {
	ctrl := TDisconnect()

	ldata := LData{
		Control1:    Control1StdFrame | Control1NoRepeat | Control1NoSysBroadcast,
		Control2:    Control2Hops(6),
		Source:      src,
		Destination: uint16(dst),
		Data:        ctrl,
	}

	return &LDataReq{
		LData: ldata,
	}
}

// ControlAck represents a T_ACK ControlData structure.
type ControlAck struct {
	ControlData
}

// TAck creates a new T_ACK ControlData structure with the given sequence number.
func TAck(seqNumber uint8) *ControlAck {
	return &ControlAck{
		ControlData: ControlData{
			Numbered:  true,
			SeqNumber: seqNumber,
			Command:   uint8(Ack),
		},
	}
}

// NewAck creates a new L_Data.req message with a T_ACK transport control field
// using the specified source and destination addresses and sequence number.
func NewAck(src, dst IndividualAddr, seq uint8) *LDataReq {
	ctrl := TAck(seq)

	ldata := LData{
		Control1:    Control1StdFrame | Control1NoSysBroadcast,
		Control2:    Control2Hops(6),
		Source:      src,
		Destination: uint16(dst),
		Data:        ctrl,
	}

	return &LDataReq{
		LData: ldata,
	}
}

// ControlNak represents a T_NAK ControlData structure.
type ControlNak struct {
	ControlData
}

// TNak creates a new T_NAK ControlData structure with the given sequence number.
func TNak(seqNumber uint8) *ControlNak {
	return &ControlNak{
		ControlData: ControlData{
			Numbered:  true,
			SeqNumber: seqNumber,
			Command:   uint8(Nak),
		},
	}
}
