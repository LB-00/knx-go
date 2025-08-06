// Copyright 2017 Ole Krüger.
// Licensed under the MIT license which can be found in the LICENSE file.

package cemi

import (
	"io"

	"github.com/LB-00/knx-go/knx/util"
)

// TPCI is the Transport Protocol Control Information.
type TPCI uint8

// These are usable TPCI values.
const (
	Connect    TPCI = 0b00 // 0
	Disconnect TPCI = 0b01 // 1
	Ack        TPCI = 0b10 // 2
	Nak        TPCI = 0b11 // 3
)

const (
	PrefixUserMessage uint8 = 0b1011 // 11
	PrefixEscape      uint8 = 0b1111 // 15
)

// APCI is the Application-layer Protocol Control Information.
type APCI uint16

// These are usable APCI values.
const (
	// Standard APCIs
	GroupValueRead         APCI = 0b0000000000
	GroupValueResponse     APCI = 0b0001000000
	GroupValueWrite        APCI = 0b0010000000
	IndividualAddrWrite    APCI = 0b0011000000
	IndividualAddrRequest  APCI = 0b0100000000
	IndividualAddrResponse APCI = 0b0101000000
	AdcRead                APCI = 0b0110000000
	AdcResponse            APCI = 0b0111000000
	MemoryRead             APCI = 0b1000000000
	MemoryResponse         APCI = 0b1001000000
	MemoryWrite            APCI = 0b1010000000
	MaskVersionRead        APCI = 0b1100000000
	MaskVersionResponse    APCI = 0b1101000000
	Restart                APCI = 0b1110000000

	// Extended APCIs
	SystemNetworkParameterRead       APCI = 0b0111001000
	SystemNetworkParameterResponse   APCI = 0b0111001001
	SystemNetworkParameterWrite      APCI = 0b0111001010
	PropertyExtValueRead             APCI = 0b0111001100
	PropertyExtValueResponse         APCI = 0b0111001101
	PropertyExtValueWriteCon         APCI = 0b0111001110
	PropertyExtValueWriteConRes      APCI = 0b0111001111
	PropertyExtValueWriteUnCon       APCI = 0b0111010000
	PropertyExtValueInfoReport       APCI = 0b0111010001
	PropertyExtDescriptionRead       APCI = 0b0111010010
	PropertyExtDescriptionResponse   APCI = 0b0111010011
	FunctionPropertyExtCommand       APCI = 0b0111010100
	FunctionPropertyExtStateRead     APCI = 0b0111010101
	FunctionPropertyExtStateResponse APCI = 0b0111010110
	MemoryExtendedWrite              APCI = 0b0111111011
	MemoryExtendedWriteResponse      APCI = 0b0111111100
	MemoryExtendedRead               APCI = 0b0111111101
	MemoryExtendedReadResponse       APCI = 0b0111111110

	// User Message APCIs
	UserMemoryRead                APCI = 0b1011000000
	UserMemoryResponse            APCI = 0b1011000001
	UserMemoryWrite               APCI = 0b1011000010
	UserMemoryBitWrite            APCI = 0b1011000100
	UserManufacturerInfoRead      APCI = 0b1011000101
	UserManufacturerInfoResponse  APCI = 0b1011000110
	FunctionPropertyCommand       APCI = 0b1011000111
	FunctionPropertyStateRead     APCI = 0b1011001000
	FunctionPropertyStateResponse APCI = 0b1011001001

	// More Extended APCIs
	FilterTableOpen                       APCI = 0b1111000000
	FilterTableRead                       APCI = 0b1111000001
	FilterTableResponse                   APCI = 0b1111000010
	FilterTableWrite                      APCI = 0b1111000011
	RouterMemoryRead                      APCI = 0b1111001000
	RouterMemoryResponse                  APCI = 0b1111001001
	RouterMemoryWrite                     APCI = 0b1111001010
	RouterStatusRead                      APCI = 0b1111001101
	RouterStatusResponse                  APCI = 0b1111001110
	RouterStatusWrite                     APCI = 0b1111001111
	MemoryBitWrite                        APCI = 0b1111010000
	AuthorizeRequest                      APCI = 0b1111010001
	AuthorizeResponse                     APCI = 0b1111010010
	KeyWrite                              APCI = 0b1111010011
	KeyResponse                           APCI = 0b1111010100
	PropertyValueRead                     APCI = 0b1111010101
	PropertyValueResponse                 APCI = 0b1111010110
	PropertyValueWrite                    APCI = 0b1111010111
	PropertyDescriptionRead               APCI = 0b1111011000
	PropertyDescriptionResponse           APCI = 0b1111011001
	NetworkParameterRead                  APCI = 0b1111011010
	NetworkParameterResponse              APCI = 0b1111011011
	IndividualAddressSerialNumberRead     APCI = 0b1111011100
	IndividualAddressSerialNumberResponse APCI = 0b1111011101
	IndividualAddressSerialNumberWrite    APCI = 0b1111011110
	DomainAddressWrite                    APCI = 0b1111100000
	DomainAddressRead                     APCI = 0b1111100001
	DomainAddressResponse                 APCI = 0b1111100010
	DomainAddressSelectiveRead            APCI = 0b1111100011
	NetworkParameterWrite                 APCI = 0b1111100100
	LinkRead                              APCI = 0b1111100101
	LinkResponse                          APCI = 0b1111100110
	LinkWrite                             APCI = 0b1111100111
	GroupPropValueRead                    APCI = 0b1111101000
	GroupPropValueResponse                APCI = 0b1111101001
	GroupPropValueWrite                   APCI = 0b1111101010
	GroupPropValueInfoReport              APCI = 0b1111101011
	DomainAddressSerialNumberRead         APCI = 0b1111101100
	DomainAddressSerialNumberResponse     APCI = 0b1111101101
	DomainAddressSerialNumberWrite        APCI = 0b1111101110
	FileStreamInforReport                 APCI = 0b1111110000
)

// IsGroupCommand determines if the APCI indicates a group command.
func (apci APCI) IsGroupCommand() bool {
	return (apci >> 6) < 3
}

// IsStandardCommand checks if the APCI is a standard command.
func (apci APCI) IsStandardCommand() bool {
	return apci != UserMemoryRead && (apci&0x3F) == 0 && (apci>>6) < 15
}

// An AppData contains application data in a transport unit.
type AppData struct {
	Numbered  bool
	SeqNumber uint8
	Command   APCI
	Data      []byte
}

// Size retrieves the packed size.
func (app *AppData) Size() uint {
	cmdLength := uint(2)

	if !app.Command.IsStandardCommand() {
		cmdLength += 1
	}

	dataLength := uint(len(app.Data))

	if dataLength > 255 {
		dataLength = 255
	} else if dataLength < 1 {
		dataLength = 1
	}

	return cmdLength + dataLength
}

// Pack into a transport data unit including its leading length byte.
func (app *AppData) Pack(buffer []byte) {
	dataLength := len(app.Data)

	if dataLength > 255 {
		dataLength = 255
	} else if dataLength < 1 {
		dataLength = 1
	}

	if !app.Command.IsStandardCommand() {
		dataLength += 1
	}

	buffer[0] = byte(dataLength)

	if app.Numbered {
		buffer[1] |= 1<<6 | (app.SeqNumber&15)<<2
	}

	// Set the lowest two bits of buffer[1] to the highest
	// two bits of the 10 bit APCI.
	buffer[1] |= byte(app.Command>>8) & 3

	if app.Command.IsStandardCommand() {
		copy(buffer[2:], app.Data)

		// Zero out the first two bits of buffer[2] and set them
		// to the remaining two bits of the 4 bit APCI.
		buffer[2] &= 63
		buffer[2] |= byte((app.Command>>6)&3) << 6
	} else {
		// Non-standard commands use the entire first data
		// byte to encode the command.
		buffer[2] = byte(app.Command & 0xFF)

		copy(buffer[3:], app.Data)
	}
}

// A ControlData encodes control information in a transport unit.
type ControlData struct {
	Numbered  bool
	SeqNumber uint8
	Command   uint8
}

// Size retrieves the packed size.
func (ControlData) Size() uint {
	return 2
}

// Pack into a transport data unit including its leading length byte.
func (control *ControlData) Pack(buffer []byte) {
	buffer[0] = 0
	buffer[1] = 1<<7 | (control.Command & 3)

	if control.Numbered {
		buffer[1] |= 1<<6 | (control.SeqNumber&15)<<2
	}
}

// A TransportUnit is responsible to transport data.
type TransportUnit interface {
	util.Packable
}

// unpackTransportUnit parses the given data in order to extract the transport unit that it encodes.
func unpackTransportUnit(data []byte, unit *TransportUnit) (uint, error) {
	if len(data) < 2 {
		return 0, io.ErrUnexpectedEOF
	}

	// Does unit contain control information?
	if (data[1] & (1 << 7)) == 1<<7 {
		numbered := (data[1] & (1 << 6)) == 1<<6
		seqNumber := (data[1] >> 2) & 15
		command := data[1] & 3

		switch command {
		case uint8(Connect):
			*unit = TConnect()
		case uint8(Disconnect):
			*unit = TDisconnect()
		case uint8(Ack):
			*unit = TAck(seqNumber)
		case uint8(Nak):
			*unit = TNak(seqNumber)
		default:
			*unit = &ControlData{
				Numbered:  numbered,
				SeqNumber: seqNumber,
				Command:   command,
			}
		}

		return 2, nil
	}

	dataLength := int(data[0])

	if len(data) < 3 || dataLength+2 < len(data) {
		return 0, io.ErrUnexpectedEOF
	}

	app := &AppData{
		Numbered:  (data[1] & (1 << 6)) == 1<<6,
		SeqNumber: (data[1] >> 2) & 15,
	}

	p := (data[1]&3)<<2 | data[2]>>6

	if p == PrefixUserMessage || p == PrefixEscape {
		app.Command = APCI(uint16(p)<<6 | uint16(data[2]))

		app.Data = make([]byte, dataLength-1)
		copy(app.Data, data[3:])
	} else {
		app.Command = APCI(uint16(p) << 6)

		app.Data = make([]byte, dataLength)
		copy(app.Data, data[2:])
		app.Data[0] &= 63
	}

	*unit = app

	return uint(dataLength) + 2, nil
}
