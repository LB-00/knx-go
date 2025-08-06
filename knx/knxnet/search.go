// Licensed under the MIT license which can be found in the LICENSE file.

package knxnet

import (
	"errors"
	"fmt"
	"net"

	"github.com/LB-00/knx-go/knx/util"
)

// NewSearchReq creates a new SearchReq, addr defines where KNXnet/IP server should send the reponse to.
func NewSearchReq(addr net.Addr) (*SearchReq, error) {
	req := &SearchReq{}

	hostinfo, err := HostInfoFromAddress(addr)
	if err != nil {
		return nil, err
	}
	req.HostInfo = hostinfo

	return req, nil
}

// A SearchReq requests a discovery from all KNXnet/IP servers via multicast.
type SearchReq struct {
	HostInfo
}

// Service returns the service identifier for Search Request.
func (SearchReq) Service() ServiceID {
	return SearchReqService
}

// A SearchRes is a Search Response from a KNXnet/IP server.
type SearchRes struct {
	Control      HostInfo
	DescriptionB DescriptionBlock
}

// Service returns the service identifier for the Search Response.
func (SearchRes) Service() ServiceID {
	return SearchResService
}

// Size returns the packed size.
func (res SearchRes) Size() uint {
	return res.Control.Size() + res.DescriptionB.DeviceHardware.Size() + res.DescriptionB.SupportedServices.Size()
}

// Pack assembles the Search Response structure in the given buffer.
func (res *SearchRes) Pack(buffer []byte) {
	util.PackSome(buffer, res.Control, res.DescriptionB.DeviceHardware, res.DescriptionB.SupportedServices)
}

// Unpack parses the given service payload in order to initialize the Search Response structure.
func (res *SearchRes) Unpack(data []byte) (n uint, err error) {
	return util.UnpackSome(data, &res.Control, &res.DescriptionB.DeviceHardware, &res.DescriptionB.SupportedServices)
}

// NewSearchReqExt creates a new SearchReqExt, addr defines where KNXnet/IP server should send the response to, and params are the optional SRP blocks.
func NewSearchReqExt(addr net.Addr, params ...SRPBlock) (*SearchReqExt, error) {
	req := &SearchReqExt{}

	var hostinfo HostInfo
	if addr == nil {
		hostinfo = HostInfo{
			Protocol: UDP4,
			Address:  Address{0x00, 0x00, 0x00, 0x00},
			Port:     0,
		}
	} else {
		var err error
		hostinfo, err = HostInfoFromAddress(addr)
		if err != nil {
			return nil, err
		}
	}
	req.Control = hostinfo

	if len(params) > 0 {
		req.Parameters = make([]SRPBlock, len(params))
		copy(req.Parameters, params)
	}

	return req, nil
}

// A SearchReqExt may be used to request a discovery from all KNXnet/IP servers via multicast
// or be directed to a specific KNXnet/IP server.
type SearchReqExt struct {
	Control    HostInfo
	Parameters []SRPBlock
}

// Service returns the service identifier for Search Request Extended.
func (SearchReqExt) Service() ServiceID {
	return SearchReqExtService
}

// Size returns the packed size.
func (req SearchReqExt) Size() uint {
	size := req.Control.Size()
	for _, param := range req.Parameters {
		size += param.Size()
	}
	return size
}

// Pack assembles the Search Request Extended structure in the given buffer.
func (req *SearchReqExt) Pack(buffer []byte) {
	req.Control.Pack(buffer)
	offset := req.Control.Size()

	// Pack each SRP block.
	for _, param := range req.Parameters {
		param.Pack(buffer[offset:])
		offset += uint(param.Size())
	}
}

// Unpack parses the given service payload in order to initialize the Search Request Extended structure.
func (req *SearchReqExt) Unpack(data []byte) (n uint, err error) {
	var length uint8
	if n, err = util.UnpackSome(
		data,
		&length, &req.Control,
	); err != nil {
		return
	}

	if length < 6 {
		return n, errors.New("invalid length for SearchReqExt structure")
	}

	req.Parameters = make([]SRPBlock, 0)
	for n < uint(length) {
		var param SRPBlock
		var ty ParameterType
		var paramLength uint8

		nn, err := util.UnpackSome(data[n:], &paramLength, &ty)
		if err != nil {
			return n, err
		}
		n += nn

		switch ty {
		case ParameterTypeSelectProgMode:
			param = &SelectProgMode{}
		case ParameterTypeSelectMACAddr:
			param = &SelectMACAddr{}
		case ParameterTypeSelectSrvSRP:
			param = &SelectSrvSRP{}
		case ParameterTypeRequestDIBs:
			param = &RequestDIBs{}
		default:
			fmt.Printf("Found unsupported parameter type: %d\n", ty)
			continue
		}

		nn, err = param.Unpack(data[n : n+uint(paramLength)])
		if err != nil {
			return 0, err
		}
		n += nn

		req.Parameters = append(req.Parameters, param)
	}

	return uint(length), nil
}

// SRPBlock represents a Search Request Parameter (SRP) Block used to transfer
// additional information regarding the search.
type SRPBlock interface {
	// Size returns the packed size of the SRP.
	Size() uint

	// Pack assembles the SRP in the given buffer.
	Pack(buffer []byte)

	// Unpack parses the given data in order to initialize the SRP.
	Unpack(data []byte) (n uint, err error)
}

// ParameterType represents the type of the Search Request Parameter.
type ParameterType uint8

// Currently supported Search Request Parameter Type values.
const (
	ParameterTypeInvalid        ParameterType = 0x00
	ParameterTypeSelectProgMode ParameterType = 0x01
	ParameterTypeSelectMACAddr  ParameterType = 0x02
	ParameterTypeSelectSrvSRP   ParameterType = 0x03
	ParameterTypeRequestDIBs    ParameterType = 0x04
)

// SelectProgMode represents the Select By Programming Mode SRP.
type SelectProgMode struct {
	Mandatory bool          // Indicates if the SRP is mandatory.
	Type      ParameterType // The type of the SRP, should be ParameterTypeSelectProgMode.
}

// NewSelectProgMode creates a new Select By Programming Mode SRP.
func NewSelectProgMode(mandatory bool) *SelectProgMode {
	return &SelectProgMode{
		Mandatory: mandatory,
		Type:      ParameterTypeSelectProgMode,
	}
}

// Size returns the packed size.
func (SelectProgMode) Size() uint {
	return 1
}

// Pack assembles the Select By Programming Mode SRP in the given buffer.
func (srp *SelectProgMode) Pack(buffer []byte) {
	var pld byte
	if srp.Mandatory {
		pld |= 0x80 // Set the mandatory bit.
	}
	pld |= byte(srp.Type)

	util.PackSome(buffer, pld)
}

// Unpack parses the given data in order to initialize the Select By Programming Mode SRP.
func (srp *SelectProgMode) Unpack(data []byte) (n uint, err error) {
	var length uint8
	var pld uint8
	if n, err = util.UnpackSome(
		data,
		&length, &pld,
	); err != nil {
		return
	}

	if length != uint8(srp.Size()) {
		return n, fmt.Errorf("invalid length for SelectProgMode structure: got %d, want 1", length)
	}

	srp.Mandatory = (pld & 0x80) != 0    // MSB indicates if the SRP is mandatory.
	srp.Type = ParameterType(pld & 0x7F) // Lower 7 bits indicate the type.

	return 1, nil
}

// SelectMACAddr represents the Select By MAC Address SRP.
type SelectMACAddr struct {
	Mandatory    bool
	Type         ParameterType
	HardwareAddr [6]byte
}

// NewSelectMACAddr creates a new Select By MAC Address SRP.
func NewSelectMACAddr(mandatory bool, addr [6]byte) *SelectMACAddr {
	return &SelectMACAddr{
		Mandatory:    mandatory,
		Type:         ParameterTypeSelectMACAddr,
		HardwareAddr: addr,
	}
}

// Size returns the packed size.
func (SelectMACAddr) Size() uint {
	return 7
}

// Pack assembles the Select By MAC Address SRP in the given buffer.
func (srp *SelectMACAddr) Pack(buffer []byte) {
	var pld byte
	if srp.Mandatory {
		pld |= 0x80 // Set the mandatory bit.
	}
	pld |= byte(srp.Type)

	util.PackSome(buffer, pld, srp.HardwareAddr)
}

// Unpack parses the given data in order to initialize the Select By MAC Address SRP.
func (srp *SelectMACAddr) Unpack(data []byte) (n uint, err error) {
	var length uint8
	var pld uint8
	if n, err = util.UnpackSome(
		data,
		&length, &pld,
		srp.HardwareAddr[:],
	); err != nil {
		return
	}

	if length != uint8(srp.Size()) {
		return n, fmt.Errorf("invalid length for SelectMACAddr structure: got %d, want 7", length)
	}

	srp.Mandatory = (pld & 0x80) != 0    // MSB indicates if the SRP is mandatory.
	srp.Type = ParameterType(pld & 0x7F) // Lower 7 bits indicate the type.

	return 7, nil
}

// SelectSrvSRP represents the Select By Service SRP.
type SelectSrvSRP struct {
	Mandatory bool
	Type      ParameterType
	Service   ServiceFamilyType
	Version   uint8
}

// NewSelectSrvSRP creates a new Select By Service SRP.
func NewSelectSrvSRP(mandatory bool, service ServiceFamilyType, version uint8) *SelectSrvSRP {
	return &SelectSrvSRP{
		Mandatory: mandatory,
		Type:      ParameterTypeSelectSrvSRP,
		Service:   service,
		Version:   version,
	}
}

// Size returns the packed size.
func (SelectSrvSRP) Size() uint {
	return 3
}

// Pack assembles the Select By Service SRP in the given buffer.
func (srp *SelectSrvSRP) Pack(buffer []byte) {
	var pld byte
	if srp.Mandatory {
		pld |= 0x80 // Set the mandatory bit.
	}
	pld |= byte(srp.Type)

	util.PackSome(buffer, pld, srp.Service, srp.Version)
}

// Unpack parses the given data in order to initialize the Select By Service SRP.
func (srp *SelectSrvSRP) Unpack(data []byte) (n uint, err error) {
	var length uint8
	var pld uint8
	if n, err = util.UnpackSome(
		data,
		&length, &pld,
		&srp.Service, &srp.Version,
	); err != nil {
		return
	}

	if length != uint8(srp.Size()) {
		return n, fmt.Errorf("invalid length for SelectSrvSRP structure: got %d, want 3", length)
	}

	srp.Mandatory = (pld & 0x80) != 0    // MSB indicates if the SRP is mandatory.
	srp.Type = ParameterType(pld & 0x7F) // Lower 7 bits indicate the type.

	return 3, nil
}

// RequestDIBs represents the Request DIBs SRP.
type RequestDIBs struct {
	Mandatory bool
	Type      ParameterType
	DescTypes []DescriptionType
}

// NewRequestDIBs creates a new Request DIBs SRP.
func NewRequestDIBs(mandatory bool, descTypes ...DescriptionType) *RequestDIBs {
	return &RequestDIBs{
		Mandatory: mandatory,
		Type:      ParameterTypeRequestDIBs,
		DescTypes: descTypes,
	}
}

// Size returns the packed size.
func (srp RequestDIBs) Size() uint {
	lenDeskTypes := uint(len(srp.DescTypes))

	if lenDeskTypes%2 != 0 {
		// A padding Description Type (0x00) is added.
		lenDeskTypes++
	}

	return uint(2) + lenDeskTypes
}

// Pack assembles the Request DIBs SRP in the given buffer.
func (srp *RequestDIBs) Pack(buffer []byte) {
	var pld byte
	if srp.Mandatory {
		pld |= 0x80 // Set the mandatory bit.
	}
	pld |= byte(srp.Type)

	descTypes := make([]byte, len(srp.DescTypes))
	for i := range srp.DescTypes {
		descTypes[i] = byte(srp.DescTypes[i])
	}

	// Ensure there is an even number of Description Types.
	if len(descTypes)%2 != 0 {
		descTypes = append(descTypes, 0x00)
	}

	length := uint8(2 + len(descTypes))

	util.PackSome(buffer, length, pld, descTypes)
}

// Unpack parses the given data in order to initialize the Request DIBs SRP.
func (srp *RequestDIBs) Unpack(data []byte) (n uint, err error) {
	var length uint8
	var pld uint8
	if n, err = util.UnpackSome(
		data,
		&length, &pld,
	); err != nil {
		return
	}

	srp.Mandatory = (pld & 0x80) != 0    // MSB indicates if the SRP is mandatory.
	srp.Type = ParameterType(pld & 0x7F) // Lower 7 bits indicate the type.

	var descTypes []DescriptionType
	for _, b := range data[n:length] {
		descType := DescriptionType(b)
		descTypes = append(descTypes, descType)
	}
	srp.DescTypes = descTypes

	if length != uint8(srp.Size()) {
		return n, errors.New("invalid length for RequestDIBs SRP structure")
	}

	return uint(length), nil
}

// A SearchResExt is a Search Response Extended from a KNXnet/IP server.
type SearchResExt struct {
	Control HostInfo
	DIBs    []DIB
}

// Service returns the service identifier for the Search Response Extended.
func (SearchResExt) Service() ServiceID {
	return SearchResExtService
}

// Size returns the packed size.
func (res SearchResExt) Size() uint {
	size := res.Control.Size()
	for _, dib := range res.DIBs {
		size += dib.Size()
	}
	return size
}

// Pack assembles the Search Response Extended structure in the given buffer.
func (res *SearchResExt) Pack(buffer []byte) {
	offset := res.Control.Size()
	res.Control.Pack(buffer[:offset])

	// Pack each DIB.
	for _, dib := range res.DIBs {
		dib.Pack(buffer[offset:])
		offset += dib.Size()
	}
}

// Unpack parses the given service payload in order to initialize the Search Response Extended structure.
func (res *SearchResExt) Unpack(data []byte) (n uint, err error) {

	if n, err = util.UnpackSome(
		data,
		&res.Control,
	); err != nil {
		return
	}

	var length uint8
	var ty DescriptionType

	// Unpack each DIB.
	for n < uint(len(data)) {
		_, err := util.UnpackSome(data[n:], &length, (*uint8)(&ty))
		if err != nil {
			return n, err
		}

		var dib DIB

		switch ty {
		case DescriptionTypeDeviceInfo:
			dib = &DeviceInformationBlock{}

		case DescriptionTypeSupportedServiceFamilies:
			dib = &SupportedServicesDIB{}

		case DescriptionTypeIPConfig:
			dib = &IPConfigDIB{}

		case DescriptionTypeIPCurrentConfig:
			dib = &IPCurrentConfigDIB{}

		case DescriptionTypeKNXAddresses:
			dib = &KNXAddrsDIB{}

		case DescriptionTypeSecuredServiceFamilies:
			dib = &SecuredServicesDIB{}

		case DescriptionTypeTunnellingInfo:
			dib = &TunnellingInfoDIB{}

		case DescriptionTypeExtendedDeviceInfo:
			dib = &ExtendedDeviceInfoDIB{}

		case DescriptionTypeManufacturerData:
			dib = &ManufacturerDataDIB{}

		default:
			fmt.Printf("Found unsupported DIB with code: 0x%02x", uint8(ty))
			continue
		}

		_, err = dib.Unpack(data[n : n+uint(length)])
		if err != nil {
			return 0, err
		}
		n += uint(length)
		res.DIBs = append(res.DIBs, dib)
	}

	return
}
