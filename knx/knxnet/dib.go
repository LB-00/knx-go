// Licensed under the MIT license which can be found in the LICENSE file.

package knxnet

import (
	"errors"
	"fmt"
	"net"

	"github.com/LB-00/knx-go/knx/cemi"
	"github.com/LB-00/knx-go/knx/util"
)

const (
	friendlyNameMaxLen = 30
)

// DescriptionType describes the type of a DeviceInformationBlock.
type DescriptionType uint8

const (
	// DescriptionTypeDeviceInfo describes Device information e.g. KNX medium.
	DescriptionTypeDeviceInfo DescriptionType = 0x01

	// DescriptionTypeSupportedServiceFamilies describes Service families supported by the device.
	DescriptionTypeSupportedServiceFamilies DescriptionType = 0x02

	// DescriptionTypeIPConfig describes IP configuration.
	DescriptionTypeIPConfig DescriptionType = 0x03

	// DescriptionTypeIPCurrentConfig describes current IP configuration.
	DescriptionTypeIPCurrentConfig DescriptionType = 0x04

	// DescriptionTypeKNXAddresses describes KNX addresses.
	DescriptionTypeKNXAddresses DescriptionType = 0x05

	// DescriptionTypeSecuredServiceFamilies describes Service families that use KNX Secure.
	DescriptionTypeSecuredServiceFamilies DescriptionType = 0x06

	// DescriptionTypeTunnellingInfo describes Tunnelling information.
	DescriptionTypeTunnellingInfo DescriptionType = 0x07

	// DescriptionTypeExtendedDeviceInfo describes extended device information.
	DescriptionTypeExtendedDeviceInfo DescriptionType = 0x08

	// DescriptionTypeManufacturerData describes a DIB structure for further data defined by device manufacturer.
	DescriptionTypeManufacturerData DescriptionType = 0xfe
)

// KNXMedium describes the KNX medium type.
type KNXMedium uint8

const (
	// KNXMediumTP1 is the TP1 medium
	KNXMediumTP1 KNXMedium = 0x02
	// KNXMediumPL110 is the PL110 medium
	KNXMediumPL110 KNXMedium = 0x04
	// KNXMediumRF is the RF medium
	KNXMediumRF KNXMedium = 0x10
	// KNXMediumIP is the IP medium
	KNXMediumIP KNXMedium = 0x20
)

// ProjectInstallationIdentifier describes a KNX project installation identifier.
type ProjectInstallationIdentifier uint16

// DeviceStatus describes the device status.
type DeviceStatus uint8

// DeviceSerialNumber desribes the serial number of a device.
type DeviceSerialNumber [6]byte

// DeviceInformationBlock contains information about a device.
type DeviceInformationBlock struct {
	Type                    DescriptionType
	Medium                  KNXMedium
	Status                  DeviceStatus
	Source                  cemi.IndividualAddr
	ProjectIdentifier       ProjectInstallationIdentifier
	SerialNumber            DeviceSerialNumber
	RoutingMulticastAddress Address
	HardwareAddr            net.HardwareAddr
	FriendlyName            string
}

// Size returns the packed size.
func (DeviceInformationBlock) Size() uint {
	return 54
}

// Pack assembles the device information structure in the given buffer.
func (dib *DeviceInformationBlock) Pack(buffer []byte) {
	buf := make([]byte, friendlyNameMaxLen)
	util.PackString(buf, friendlyNameMaxLen, dib.FriendlyName)

	util.PackSome(
		buffer,
		uint8(dib.Size()), uint8(dib.Type),
		uint8(dib.Medium), uint8(dib.Status),
		uint16(dib.Source),
		uint16(dib.ProjectIdentifier),
		dib.SerialNumber[:],
		dib.RoutingMulticastAddress[:],
		[]byte(dib.HardwareAddr),
		buf,
	)
}

// Unpack parses the given data in order to initialize the structure.
func (dib *DeviceInformationBlock) Unpack(data []byte) (n uint, err error) {
	var length uint8

	dib.HardwareAddr = make([]byte, 6)
	if n, err = util.UnpackSome(
		data,
		&length, (*uint8)(&dib.Type),
		(*uint8)(&dib.Medium), (*uint8)(&dib.Status),
		(*uint16)(&dib.Source),
		(*uint16)(&dib.ProjectIdentifier),
		dib.SerialNumber[:],
		dib.RoutingMulticastAddress[:],
		[]byte(dib.HardwareAddr),
	); err != nil {
		return
	}

	nn, err := util.UnpackString(data[n:], friendlyNameMaxLen, &dib.FriendlyName)
	if err != nil {
		return n, err
	}
	n += nn

	if length != uint8(dib.Size()) {
		return n, errors.New("device info structure length is invalid")
	}

	return
}

// SupportedServicesDIB contains information about the supported services of a device.
type SupportedServicesDIB struct {
	Type     DescriptionType
	Families []ServiceFamily
}

// Size returns the packed size.
func (sdib SupportedServicesDIB) Size() uint {
	size := uint(2)
	for _, f := range sdib.Families {
		size += f.Size()
	}

	return size
}

// Pack assembles the supported services structure in the given buffer.
func (sdib *SupportedServicesDIB) Pack(buffer []byte) {
	util.PackSome(
		buffer,
		uint8(sdib.Size()), uint8(sdib.Type),
	)

	offset := uint(2)
	for _, f := range sdib.Families {
		f.Pack(buffer[offset:])
		offset += f.Size()
	}
}

// Unpack parses the given data in order to initialize the structure.
func (sdib *SupportedServicesDIB) Unpack(data []byte) (n uint, err error) {
	var length uint8
	if n, err = util.UnpackSome(
		data,
		&length, (*uint8)(&sdib.Type),
	); err != nil {
		return
	}

	for n < uint(length) {
		f := ServiceFamily{}
		nn, err := f.Unpack(data[n:])
		if err != nil {
			return n, errors.New("unable to unpack service family")
		}

		n += nn
		sdib.Families = append(sdib.Families, f)
	}

	if length != uint8(sdib.Size()) {
		return n, errors.New("invalid length for Supported Services structure")
	}

	return
}

// IPConfigDIB contains information about the IP configuration of a device.
type IPConfigDIB struct {
	Type           DescriptionType
	IP             Address
	Mask           Address
	Gateway        Address
	IPCapabilities uint8
	IPAssignment   uint8
}

// Size returns the packed size.
func (IPConfigDIB) Size() uint {
	return 16
}

// Pack assembles the IP configuration structure in the given buffer.
func (idib *IPConfigDIB) Pack(buffer []byte) {
	util.PackSome(
		buffer,
		uint8(idib.Size()), uint8(idib.Type),
		idib.IP[:], idib.Mask[:], idib.Gateway[:],
		idib.IPCapabilities, idib.IPAssignment,
	)
}

// Unpack parses the given data in order to initialize the structure.
func (idib *IPConfigDIB) Unpack(data []byte) (n uint, err error) {
	var length uint8
	if n, err = util.UnpackSome(
		data,
		&length, (*uint8)(&idib.Type),
		idib.IP[:], idib.Mask[:], idib.Gateway[:],
		&idib.IPCapabilities, &idib.IPAssignment,
	); err != nil {
		return
	}

	if length != uint8(idib.Size()) {
		return n, errors.New("invalid length for IP Config structure")
	}

	return
}

// IPCurrentConfigDIB contains information about the current IP configuration of a device.
type IPCurrentConfigDIB struct {
	Type         DescriptionType
	IP           Address
	Mask         Address
	Gateway      Address
	DHCPServer   Address
	IPAssignment uint8
	Reserved     byte
}

// Size returns the packed size.
func (IPCurrentConfigDIB) Size() uint {
	return 20
}

// Pack assembles the current IP configuration structure in the given buffer.
func (idib *IPCurrentConfigDIB) Pack(buffer []byte) {
	util.PackSome(
		buffer,
		uint8(idib.Size()), uint8(idib.Type),
		idib.IP[:], idib.Mask[:],
		idib.Gateway[:], idib.DHCPServer[:],
		idib.IPAssignment, idib.Reserved,
	)
}

// Unpack parses the given data in order to initialize the structure.
func (idib *IPCurrentConfigDIB) Unpack(data []byte) (n uint, err error) {
	var length uint8
	if n, err = util.UnpackSome(
		data,
		&length, (*uint8)(&idib.Type),
		idib.IP[:], idib.Mask[:],
		idib.Gateway[:], idib.DHCPServer[:],
		&idib.IPAssignment, &idib.Reserved,
	); err != nil {
		return
	}

	if length != uint8(idib.Size()) {
		return n, errors.New("invalid length for IP Current Config structure")
	}

	return
}

// KNXAddrsDIB contains information about the individual KNX addresses of a device.
type KNXAddrsDIB struct {
	Type     DescriptionType
	KNXAddrs []cemi.IndividualAddr
}

// Size returns the packed size.
func (kdib KNXAddrsDIB) Size() uint {
	return uint(2 + len(kdib.KNXAddrs)*2)
}

// Pack assembles the KNX addresses structure in the given buffer.
func (kdib *KNXAddrsDIB) Pack(buffer []byte) {
	util.PackSome(
		buffer, uint8(kdib.Size()), uint8(kdib.Type),
	)

	offset := uint(2)
	for _, addr := range kdib.KNXAddrs {
		util.PackSome(buffer[offset:], uint16(addr))
		offset += 2
	}
}

// Unpack parses the given data in order to initialize the structure.
func (kdib *KNXAddrsDIB) Unpack(data []byte) (n uint, err error) {
	var length uint8
	if n, err = util.UnpackSome(
		data,
		&length, (*uint8)(&kdib.Type),
	); err != nil {
		return
	}

	for n < uint(length) {
		var addr cemi.IndividualAddr
		nn, err := util.UnpackSome(data[n:], (*uint16)(&addr))
		if err != nil {
			return n, errors.New("unable to unpack individual address")
		}
		n += nn
		kdib.KNXAddrs = append(kdib.KNXAddrs, addr)
	}

	if length != uint8(kdib.Size()) {
		return n, errors.New("invalid length for KNX Addresses structure")
	}

	return
}

// ManufacturerDataDIB contains information about manufacturer-specific data.
type ManufacturerDataDIB struct {
	Type DescriptionType
	ID   uint16
	Data []byte
}

// Size returns the packed size.
func (mdib ManufacturerDataDIB) Size() uint {
	return uint(4 + len(mdib.Data))
}

// Pack assembles the manufacturer data structure in the given buffer.
func (mdib *ManufacturerDataDIB) Pack(buffer []byte) {
	util.PackSome(
		buffer,
		uint8(mdib.Size()), uint8(mdib.Type),
		mdib.ID,
		mdib.Data,
	)
}

// Unpack parses the given data in order to initialize the structure.
func (mdib *ManufacturerDataDIB) Unpack(data []byte) (n uint, err error) {
	var length uint8

	if n, err = util.UnpackSome(
		data,
		&length, (*uint8)(&mdib.Type),
		(*uint16)(&mdib.ID),
	); err != nil {
		return
	}

	mdib.Data = data[n:]

	if length != uint8(mdib.Size()) {
		return n, errors.New("invalid length for Manufacturer Data structure")
	}

	return
}

// SecuredServicesDIB contains information about the services that use KNX Secure.
type SecuredServicesDIB struct {
	Type     DescriptionType
	Families []ServiceFamily
}

// Size returns the packed size.
func (sdib SecuredServicesDIB) Size() uint {
	size := uint(2)
	for _, f := range sdib.Families {
		size += f.Size()
	}

	return size
}

// Pack assembles the supported services structure in the given buffer.
func (sdib *SecuredServicesDIB) Pack(buffer []byte) {
	util.PackSome(
		buffer,
		uint8(sdib.Size()), uint8(sdib.Type),
	)

	offset := uint(2)
	for _, f := range sdib.Families {
		f.Pack(buffer[offset:])
		offset += f.Size()
	}
}

// Unpack parses the given data in order to initialize the structure.
func (sdib *SecuredServicesDIB) Unpack(data []byte) (n uint, err error) {
	var length uint8
	if n, err = util.UnpackSome(
		data,
		&length, (*uint8)(&sdib.Type),
	); err != nil {
		return
	}

	for n < uint(length) {
		f := ServiceFamily{}
		nn, err := f.Unpack(data[n:])
		if err != nil {
			return n, errors.New("unable to unpack service family")
		}

		n += nn
		sdib.Families = append(sdib.Families, f)
	}

	if length != uint8(sdib.Size()) {
		return n, errors.New("invalid length for Supported Services structure")
	}

	return
}

// TunnellingSlot describes a tunneling slot of the TunnellingInformationDIB.
type TunnellingSlot struct {
	Addr   cemi.IndividualAddr
	Status uint16
}

// Size returns the packed size.
func (ts TunnellingSlot) Size() uint {
	return 4
}

// Pack assembles the tunneling slot structure in the given buffer.
func (ts *TunnellingSlot) Pack(buffer []byte) {
	util.PackSome(
		buffer,
		uint16(ts.Addr), ts.Status,
	)
}

// Unpack parses the given data in order to initialize the tunneling slot structure.
func (ts *TunnellingSlot) Unpack(data []byte) (n uint, err error) {
	if len(data) < 4 {
		return 0, fmt.Errorf("data too short to unpack TunnelingSlot: %d bytes", len(data))
	}
	n, err = util.UnpackSome(data, (*uint16)(&ts.Addr), &ts.Status)
	if err != nil {
		return n, fmt.Errorf("unable to unpack TunnelingSlot: %w", err)
	}
	if ts.Addr == 0 {
		return n, fmt.Errorf("invalid TunnelingSlot address: %d", ts.Addr)
	}
	return n, nil
}

// TunnellingInfoDIB contains information about the tunnelling capabilities of a device.
type TunnellingInfoDIB struct {
	Type     DescriptionType
	APDUSize uint16
	Slots    []TunnellingSlot
}

// Size returns the packed size.
func (tdib TunnellingInfoDIB) Size() uint {
	return uint(4 + len(tdib.Slots)*4)
}

// Pack assembles the tunnelling information structure in the given buffer.
func (tdib *TunnellingInfoDIB) Pack(buffer []byte) {
	util.PackSome(
		buffer,
		uint8(tdib.Size()), uint8(tdib.Type),
		tdib.APDUSize,
	)

	offset := uint(4)
	for _, s := range tdib.Slots {
		s.Pack(buffer[offset:])
		offset += s.Size()
	}
}

// Unpack parses the given data in order to initialize the structure.
func (tdib *TunnellingInfoDIB) Unpack(data []byte) (n uint, err error) {
	var length uint8
	if n, err = util.UnpackSome(
		data,
		&length, (*uint8)(&tdib.Type),
		&tdib.APDUSize,
	); err != nil {
		return
	}

	for n < uint(length) {
		s := TunnellingSlot{}
		nn, err := s.Unpack(data[n:])
		if err != nil {
			return n, fmt.Errorf("unable to unpack tunneling slot: %w", err)
		}

		n += nn
		if s.Addr == 0 {
			return n, fmt.Errorf("invalid tunneling slot address: %d", s.Addr)
		}
		tdib.Slots = append(tdib.Slots, s)
	}

	if length != uint8(tdib.Size()) {
		return n, errors.New("invalid length for Tunneling Information structure")
	}

	return
}

// ExtendedDeviceInfoDIB contains extended device information.
type ExtendedDeviceInfoDIB struct {
	Type             DescriptionType
	MediumStatus     uint8
	Reserved         uint8
	APDUSize         uint16
	DeviceDescriptor uint16
}

// Size returns the packed size.
func (ExtendedDeviceInfoDIB) Size() uint {
	return 8
}

// Pack assembles the extended device information structure in the given buffer.
func (edib *ExtendedDeviceInfoDIB) Pack(buffer []byte) {
	util.PackSome(
		buffer,
		uint8(edib.Size()), uint8(edib.Type),
		edib.MediumStatus, edib.Reserved,
		edib.APDUSize,
		edib.DeviceDescriptor,
	)
}

// Unpack parses the given data in order to initialize the structure.
func (edib *ExtendedDeviceInfoDIB) Unpack(data []byte) (n uint, err error) {
	var length uint8
	if n, err = util.UnpackSome(
		data,
		&length, (*uint8)(&edib.Type),
		&edib.MediumStatus, &edib.Reserved,
		&edib.APDUSize,
		&edib.DeviceDescriptor,
	); err != nil {
		return
	}

	if length != uint8(edib.Size()) {
		return n, errors.New("invalid length for Extended Device Info structure")
	}

	return
}

// ServiceFamilyType describes a KNXnet service family type.
type ServiceFamilyType uint8

const (
	// ServiceFamilyTypeIPCore is the KNXnet/IP Core family type.
	ServiceFamilyTypeIPCore = 0x02
	// ServiceFamilyTypeIPDeviceManagement is the KNXnet/IP Device Management family type.
	ServiceFamilyTypeIPDeviceManagement = 0x03
	// ServiceFamilyTypeIPTunnelling is the KNXnet/IP Tunnelling family type.
	ServiceFamilyTypeIPTunnelling = 0x04
	// ServiceFamilyTypeIPRouting is the KNXnet/IP Routing family type.
	ServiceFamilyTypeIPRouting = 0x05
	// ServiceFamilyTypeIPRemoteLogging is the KNXnet/IP Remote Logging family type.
	ServiceFamilyTypeIPRemoteLogging = 0x06
	// ServiceFamilyTypeIPRemoteConfigurationAndDiagnosis is the KNXnet/IP Remote Configuration and Diagnosis family type.
	ServiceFamilyTypeIPRemoteConfigurationAndDiagnosis = 0x07
	// ServiceFamilyTypeIPObjectServer is the KNXnet/IP Object Server family type.
	ServiceFamilyTypeIPObjectServer = 0x08
	// ServiceFamilyTypeIPSecure is the KNXnet/IP Secure family type.
	ServiceFamilyTypeIPSecure = 0x09
)

// ServiceFamily describes a KNXnet service supported by a device.
type ServiceFamily struct {
	Type    ServiceFamilyType
	Version uint8
}

// Size returns the packed size.
func (ServiceFamily) Size() uint {
	return 2
}

// Pack assembles the service family structure in the given buffer.
func (f *ServiceFamily) Pack(buffer []byte) {
	util.PackSome(
		buffer,
		uint8(f.Type), f.Version,
	)
}

// Unpack parses the given data in order to initialize the structure.
func (f *ServiceFamily) Unpack(data []byte) (n uint, err error) {
	return util.UnpackSome(data, (*uint8)(&f.Type), &f.Version)
}

// DescriptionBlock is returned by a Search Request, a Search Request Extended,
// a Description Request or a Diagnostic Request. DIBs other than the Device
// Information DIB and the Supported Service Families DIB are optional.
type DescriptionBlock struct {
	DeviceHardware     DeviceInformationBlock
	SupportedServices  SupportedServicesDIB
	IPConfig           IPConfigDIB
	IPCurrentConfig    IPCurrentConfigDIB
	KNXAddrs           KNXAddrsDIB
	SecuredServices    SecuredServicesDIB
	TunnellingInfo     TunnellingInfoDIB
	ExtendedDeviceInfo ExtendedDeviceInfoDIB
	ManufacturerData   ManufacturerDataDIB
	UnknownBlocks      []UnknownDescriptionBlock
}

// Unpack parses the given service payload in order to initialize the Description Block.
// It can cope with not in sequence and unknown Device Information Blocks (DIB).
func (di *DescriptionBlock) Unpack(data []byte) (n uint, err error) {
	var length uint8
	var ty DescriptionType

	n = 0
	for n < uint(len(data)) {
		// DIBs should always have a length and a type.
		_, err := util.UnpackSome(data[n:], &length, (*uint8)(&ty))
		if err != nil {
			return 0, err
		}

		switch ty {
		case DescriptionTypeDeviceInfo:
			_, err = di.DeviceHardware.Unpack(data[n : n+uint(length)])
			if err != nil {
				return 0, err
			}
			n += uint(length)

		case DescriptionTypeSupportedServiceFamilies:
			_, err = di.SupportedServices.Unpack(data[n : n+uint(length)])
			if err != nil {
				return 0, err
			}
			n += uint(length)

		case DescriptionTypeIPConfig:
			_, err = di.IPConfig.Unpack(data[n : n+uint(length)])
			if err != nil {
				return 0, err
			}
			n += uint(length)

		case DescriptionTypeIPCurrentConfig:
			_, err = di.IPCurrentConfig.Unpack(data[n : n+uint(length)])
			if err != nil {
				return 0, err
			}
			n += uint(length)

		case DescriptionTypeKNXAddresses:
			_, err = di.KNXAddrs.Unpack(data[n : n+uint(length)])
			if err != nil {
				return 0, err
			}
			n += uint(length)

		case DescriptionTypeSecuredServiceFamilies:
			_, err = di.SecuredServices.Unpack(data[n : n+uint(length)])
			if err != nil {
				return 0, err
			}
			n += uint(length)

		case DescriptionTypeTunnellingInfo:
			_, err = di.TunnellingInfo.Unpack(data[n : n+uint(length)])
			if err != nil {
				return 0, err
			}
			n += uint(length)

		case DescriptionTypeExtendedDeviceInfo:
			_, err = di.ExtendedDeviceInfo.Unpack(data[n : n+uint(length)])
			if err != nil {
				return 0, err
			}
			n += uint(length)

		case DescriptionTypeManufacturerData:
			_, err = di.ManufacturerData.Unpack(data[n : n+uint(length)])
			if err != nil {
				return 0, err
			}
			n += uint(length)

			// Original implementation did not handle these DIBs.
		// case DescriptionTypeIPConfig, DescriptionTypeIPCurrentConfig,
		// 	DescriptionTypeKNXAddresses, DescriptionTypeManufacturerData:
		// 	u := UnknownDescriptionBlock{Type: ty}
		//
		// 	// known DIBs without data will be silently ignored.
		// 	if length > 2 {
		// 		// _, err = u.Unpack(data[n+2 : n+uint(length)-2]) // wrong end index in original code
		// 		_, err = u.Unpack(data[n+2 : n+uint(length)])
		// 		if err != nil {
		// 			return 0, err
		// 		}
		// 		di.UnknownBlocks = append(di.UnknownBlocks, u)
		// 		util.Log(di, "DIB not parsed: 0x%02x", ty)
		// 	}
		// 	n += uint(length)

		default:
			util.Log(di, "Found unsupported DIB with code: 0x%02x", ty)
			n += uint(length)
		}
	}

	return n, err
}

// UnknownDescriptionBlock is a placeholder for unknown DIBs.
type UnknownDescriptionBlock struct {
	Type DescriptionType
	Data []byte
}

// Unpack Unknown Description Blocks into a buffer.
func (u *UnknownDescriptionBlock) Unpack(data []byte) (n uint, err error) {
	u.Data = make([]byte, len(data))
	return util.UnpackSome(data, u.Data)
}

type DIB interface {
	// Size returns the packed size of the DIB.
	Size() uint

	// Pack assembles the DIB structure in the given buffer.
	Pack(buffer []byte)

	// Unpack parses the given data in order to initialize the DIB structure.
	Unpack(data []byte) (n uint, err error)

	// Type returns the type of the DIB.
	// Type() DescriptionType
}
