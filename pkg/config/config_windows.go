package config

import (
	"net"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const GAA_FLAG_INCLUDE_GATEWAYS = 0x00000080

type IpAdapterWinsServerAddress struct {
	Length  uint32
	_       uint32
	Next    *IpAdapterWinsServerAddress
	Address windows.SocketAddress
}

type IpAdapterGatewayAddress struct {
	Length  uint32
	_       uint32
	Next    *IpAdapterGatewayAddress
	Address windows.SocketAddress
}

type IpAdapterAddresses struct {
	Length                uint32
	IfIndex               uint32
	Next                  *IpAdapterAddresses
	AdapterName           *byte
	FirstUnicastAddress   *windows.IpAdapterUnicastAddress
	FirstAnycastAddress   *windows.IpAdapterAnycastAddress
	FirstMulticastAddress *windows.IpAdapterMulticastAddress
	FirstDnsServerAddress *windows.IpAdapterDnsServerAdapter
	DnsSuffix             *uint16
	Description           *uint16
	FriendlyName          *uint16
	PhysicalAddress       [syscall.MAX_ADAPTER_ADDRESS_LENGTH]byte
	PhysicalAddressLength uint32
	Flags                 uint32
	Mtu                   uint32
	IfType                uint32
	OperStatus            uint32
	Ipv6IfIndex           uint32
	ZoneIndices           [16]uint32
	FirstPrefix           *windows.IpAdapterPrefix
	/* more fields might be present here. */
	TransmitLinkSpeed      uint64
	ReceiveLinkSpeed       uint64
	FirstWINSServerAddress *IpAdapterWinsServerAddress
	FirstGatewayAddress    *IpAdapterGatewayAddress
}

func adapterAddresses() ([]*IpAdapterAddresses, error) {
	var b []byte
	// https://docs.microsoft.com/en-us/windows/win32/api/iphlpapi/nf-iphlpapi-getadaptersaddresses
	// #define WORKING_BUFFER_SIZE 15000
	l := uint32(15000)
	for {
		b = make([]byte, l)
		err := windows.GetAdaptersAddresses(syscall.AF_UNSPEC, GAA_FLAG_INCLUDE_GATEWAYS|windows.GAA_FLAG_INCLUDE_PREFIX, 0, (*windows.IpAdapterAddresses)(unsafe.Pointer(&b[0])), &l)
		if err == nil {
			if l == 0 {
				return nil, nil
			}
			break
		}
		if err.(syscall.Errno) != syscall.ERROR_BUFFER_OVERFLOW {
			return nil, os.NewSyscallError("getadaptersaddresses", err)
		}
		if l <= uint32(len(b)) {
			return nil, os.NewSyscallError("getadaptersaddresses", err)
		}
	}
	aas := make([]*IpAdapterAddresses, 0, uintptr(l)/unsafe.Sizeof(IpAdapterAddresses{}))
	for aa := (*IpAdapterAddresses)(unsafe.Pointer(&b[0])); aa != nil; aa = aa.Next {
		aas = append(aas, aa)
	}
	return aas, nil
}

/// As per [RFC 3879], the whole `FEC0::/10` prefix is
/// deprecated. New software must not support site-local
/// addresses.
///
/// [RFC 3879]: https://tools.ietf.org/html/rfc3879
func isUnicastLinkLocal(ip net.IP) bool {
	return len(ip) == net.IPv6len && ip[0] == 0xfe && ip[1] == 0xc0
}

func GetDefaultDnsServers() ([]string, error) {
	ifs, err := adapterAddresses()
	if err != nil {
		return nil, err
	}
	dnsServers := make([]string, 0)
	for _, ifi := range ifs {
		if ifi.OperStatus != windows.IfOperStatusUp {
			continue
		}

		if ifi.FirstGatewayAddress == nil {
			continue
		}

		for dnsServer := ifi.FirstDnsServerAddress; dnsServer != nil; dnsServer = dnsServer.Next {
			ip := dnsServer.Address.IP()
			if isUnicastLinkLocal(ip) {
				continue
			}
			dnsServers = append(dnsServers, ip.String())
		}
	}
	return dnsServers, nil
}

func GetDefaultServers() ([]string, int, []string, error) {
	// TODO: DNS Suffix
	servers, err := GetDefaultDnsServers()
	return servers, 0, nil, err
}
