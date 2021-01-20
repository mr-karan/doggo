package config

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// GAA_FLAG_INCLUDE_GATEWAYS Return the addresses of default gateways.
// This flag is supported on Windows Vista and later.
const GAA_FLAG_INCLUDE_GATEWAYS = 0x00000080

// IpAdapterWinsServerAddress structure in a linked list of Windows Internet Name Service (WINS) server addresses for the adapter.
type IpAdapterWinsServerAddress struct {
	Length  uint32
	_       uint32
	Next    *IpAdapterWinsServerAddress
	Address windows.SocketAddress
}

// IpAdapterGatewayAddress structure in a linked list of gateways for the adapter.
type IpAdapterGatewayAddress struct {
	Length  uint32
	_       uint32
	Next    *IpAdapterGatewayAddress
	Address windows.SocketAddress
}

// IpAdapterAddresses structure is the header node for a linked list of addresses for a particular adapter.
// This structure can simultaneously be used as part of a linked list of IP_ADAPTER_ADDRESSES structures.
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
	FirstWinsServerAddress *IpAdapterWinsServerAddress
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

func getDefaultDNSServers() ([]string, error) {
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

// GetDefaultServers get system default nameserver
func GetDefaultServers() ([]string, int, []string, error) {
	// TODO: DNS Suffix
	servers, err := getDefaultDNSServers()
	return servers, 0, nil, err
}
