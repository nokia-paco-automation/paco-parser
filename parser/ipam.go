package parser

import (
	"errors"
	"net"
	"strconv"

	"github.com/apparentlymart/go-cidr/cidr"
	log "github.com/sirupsen/logrus"
)

// Ipam struct with index isl or lo, etc
type Ipam struct {
	//
	IP map[string]*IpamAlloc
}

// IpamAlloc struct that holds the allocation of the prefix
type IpamAlloc struct {
	PrefixLength     *int
	AllocatedSubnets map[string]*Link
	NextFreeSubnet   *net.IPNet
	NextFreeAddress  *string
}

// Allocates a new IPAM struct per link Kind (network, loopback, access, etc)
func NewIPAM(netwInfo *NetworkInfo) (*Ipam, error) {
	log.Debugf("NewIPAM: %s ...", *netwInfo.Kind)

	var err error
	ipam := new(Ipam)
	// name is "loopback", "isl" or "access"
	ipam.IP = make(map[string]*IpamAlloc)

	// initialize prefix Length
	for _, Ipv4Cidr := range netwInfo.Ipv4Cidr {
		if *netwInfo.AddressingSchema == "dual-stack" || *netwInfo.AddressingSchema == "ipv4-only" {
			if err = initializeIPAM(netwInfo.Kind,
				StringPtr("ipv4"),
				Ipv4Cidr,                       // IPv4 CIDR
				netwInfo.Ipv4ItfcePrefixLength, // IPv4 prefix length
				ipam); err != nil {
				return nil, err
			}
		}
	}
	for _, Ipv6Cidr := range netwInfo.Ipv6Cidr {
		if *netwInfo.AddressingSchema == "dual-stack" || *netwInfo.AddressingSchema == "ipv6-only" {
			if err = initializeIPAM(netwInfo.Kind,
				StringPtr("ipv6"),
				Ipv6Cidr,                       // IPv6 CIDR
				netwInfo.Ipv6ItfcePrefixLength, // IPv6 prefix length
				ipam); err != nil {
				return nil, err
			}
		}
	}
	return ipam, nil
}

// IPAMInit initializes the ipam struct per link Kind; kind = network, acess, loopback
func initializeIPAM(kind, version, prefix *string, prefixLength *int, ipam *Ipam) error {
	log.Debugf("InitializeIPAM: %s, %s, %s ...", *kind, *version, *prefix)
	ipam.IP[*prefix] = new(IpamAlloc)

	if *kind == "loopback" {
		if *version == "ipv4" {
			ipam.IP[*prefix].PrefixLength = IntPtr(32)
		} else { // ipv6
			ipam.IP[*prefix].PrefixLength = IntPtr(128)
		}
	} else {
		ipam.IP[*prefix].PrefixLength = prefixLength
	}

	// initialize AllocatedSubnets
	ipam.IP[*prefix].AllocatedSubnets = make(map[string]*Link)

	// initialize NextFreeSubnet
	// get the base ip address and network
	ipAddr, ipNet, err := net.ParseCIDR(*prefix)
	if err != nil {
		return err
	}

	// initialize a new CIdr with the prefix length
	ipCidr := ipAddr.String() + "/" + strconv.Itoa(*ipam.IP[*prefix].PrefixLength)
	ipAddr, ipNet, err = net.ParseCIDR(ipCidr)
	if err != nil {
		return err
	}

	ipam.IP[*prefix].NextFreeSubnet = ipNet
	ipam.IP[*prefix].NextFreeAddress = StringPtr(ipAddr.String())
	return nil
}

// IPAMAllocateLinkPrefix allocates a link prefix
func (ipam *Ipam) IPAMAllocateLinkPrefix(link *Link, ipv4Cidr, ipv6Cidr *string) (err error) {
	log.Debugf("IPAMAllocateLinkPrefix: %s...", *link.Kind)

	log.Debugf("ipv4Prefix: %s...", *ipv4Cidr)
	log.Debugf("ipv6Prefix: %s...", *ipv6Cidr)
	if *ipv4Cidr != "" {
		err = ipam.IPAMAllocPrefixPerLink(StringPtr("ipv4"), ipv4Cidr, link)
		if err != nil {
			return err
		}
	}
	if *ipv6Cidr != "" {
		err = ipam.IPAMAllocPrefixPerLink(StringPtr("ipv6"), ipv6Cidr, link)
		if err != nil {
			return err
		}
	}

	return nil
}

// IPAMLinkPrefixAlloc function allocates the ipv4 or ipv6 prefixes per link
func (ipam *Ipam) IPAMAllocPrefixPerLink(version, prefix *string, link *Link) (err error) {
	log.Debugf("IPAMLinkPrefixAlloc: %s, %s, %s ...", *link.Kind, *version, *prefix)
	//p.IPAM[*link.Kind].IP[*prefix].NextFreeSubnet

	ipNet := ipam.IP[*prefix].NextFreeSubnet
	ipMask, length := ipNet.Mask.Size()

	if length != 32 && *version == "ipv4" {
		log.Error("Not an IPv4 address")
	}

	if length != 128 && *version == "ipv6" {
		log.Error("Not an IPv6 address")
	}

	switch ipMask {
	case 31:
		// for a /31 ipv4 address the A address starts with .0
		link.A.IPv4Address = StringPtr(ipNet.IP.To4().String())
		link.B.IPv4Address, err = incrementIP(StringPtr(ipNet.IP.To4().String()), StringPtr(ipNet.String()))
		if err != nil {
			return err
		}
	case 127:
		// for a /127 ipv6 address the A address starts with .0
		link.A.IPv6Address = StringPtr(ipNet.IP.To16().String())
		link.B.IPv6Address, err = incrementIP(StringPtr(ipNet.IP.To16().String()), StringPtr(ipNet.String()))
		if err != nil {
			return err
		}
	default:
		// for a non /31 ipv4 or /127 ipv6 the A address starts with .1
		if *version == "ipv4" {
			link.A.IPv4Address, err = incrementIP(StringPtr(ipNet.IP.To4().String()), StringPtr(ipNet.String()))
			link.B.IPv4Address, err = incrementIP(link.A.IPv4Address, StringPtr(ipNet.String()))
			if err != nil {
				return err
			}
		} else { // version is ipv6
			link.A.IPv6Address, err = incrementIP(StringPtr(ipNet.IP.To16().String()), StringPtr(ipNet.String()))
			link.B.IPv6Address, err = incrementIP(link.A.IPv6Address, StringPtr(ipNet.String()))
			if err != nil {
				return err
			}
		}
	}
	link.A.IPv4NeighborAddress = link.B.IPv4Address
	link.B.IPv4NeighborAddress = link.A.IPv4Address
	link.A.IPv6NeighborAddress = link.B.IPv6Address
	link.B.IPv6NeighborAddress = link.A.IPv6Address

	if *version == "ipv4" {
		//ipNet.Mask.String() // returns fffffffe iso /31
		link.A.IPv4PrefixLength = IntPtr(ipMask)
		link.A.IPv4Prefix = StringPtr(ipNet.String())
		//ipNet.Mask.String() // returns fffffffe iso /31
		link.B.IPv4PrefixLength = IntPtr(ipMask)
		link.B.IPv4Prefix = StringPtr(ipNet.String())
		link.B.IPv4NeighborPrefix = link.A.IPv4Prefix
		link.A.IPv4NeighborPrefix = link.B.IPv4Prefix
	} else { // version ipv6
		link.A.IPv6PrefixLength = IntPtr(ipMask)
		link.A.IPv6Prefix = StringPtr(ipNet.String())
		link.B.IPv6PrefixLength = IntPtr(ipMask)
		link.B.IPv6Prefix = StringPtr(ipNet.String())
		link.B.IPv6NeighborPrefix = link.A.IPv6Prefix
		link.A.IPv6NeighborPrefix = link.B.IPv6Prefix
	}
	var b bool
	ipam.IP[*prefix].NextFreeSubnet, b = cidr.NextSubnet(ipNet, ipMask)
	if b == true {
		log.Fatal("Next IP cannot be allocated")
	}
	_, ipPrefixNet, err := net.ParseCIDR(*prefix)
	if !ipPrefixNet.Contains(ipam.IP[*prefix].NextFreeSubnet.IP) {
		log.Fatal("Next IP cannot be allocated")
	}
	return nil
}

// IPAMAllocateAddress - allocated address kind = network, access, loopback
// typically being used for loopback address allocation of the network elements
func (ipam *Ipam) IPAMAllocateAddress(kind *string, ipv4Cidrs, ipv6Cidrs []*string) (*Endpoint, error) {
	log.Debug("AllocateIPEndpoint ...")
	var err error
	e := new(Endpoint)

	log.Debugf("Kind: %s, ipv4Cidrs: %v", kind, ipv4Cidrs)
	log.Debugf("Kind: %s, ipv6Cidrs: %v", kind, ipv4Cidrs)
	for _, ipv4Prefix := range ipv4Cidrs {
		if *ipv4Prefix != "" {
			err = ipam.IPEndpointAlloc(*kind, "ipv4", *ipv4Prefix, e)
			if err != nil {
				return nil, err
			}
		}
	}

	for _, ipv6Prefix := range ipv6Cidrs {
		if *ipv6Prefix != "" {
			err = ipam.IPEndpointAlloc(*kind, "ipv6", *ipv6Prefix, e)
			if err != nil {
				return nil, err
			}
		}
	}
	return e, nil
}

// IPEndpointAlloc function allocates the ipv4 or ipv6 addresses; kind = network, access, loopback
func (ipam *Ipam) IPEndpointAlloc(kind, version, prefix string, e *Endpoint) error {
	log.Debug("IPEndpointAlloc ...")
	var err error
	ipAddr := ipam.IP[prefix].NextFreeAddress

	if version == "ipv4" {
		e.IPv4Prefix = StringPtr(*ipAddr + "/" + "32")
		e.IPv4Address = ipAddr
		e.IPv4PrefixLength = IntPtr(32)
	} else {
		e.IPv6Prefix = StringPtr(*ipAddr + "/" + "128")
		e.IPv6Address = ipAddr
		e.IPv6PrefixLength = IntPtr(128)
	}

	_, ipNet, err := net.ParseCIDR(prefix)
	ipam.IP[prefix].NextFreeAddress, err = incrementIP(ipAddr, StringPtr(ipNet.String()))
	if err != nil {
		return err
	}

	return nil
}

func incrementIP(origIP, cidr *string) (*string, error) {
	ip := net.ParseIP(*origIP)
	_, ipNet, err := net.ParseCIDR(*cidr)
	if err != nil {
		return origIP, err
	}
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
	if !ipNet.Contains(ip) {
		return origIP, errors.New("overflowed CIDR while incrementing IP")
	}
	return StringPtr(ip.String()), nil
}

// decrementIP returns the given IP - 1
func decrementIP(ip net.IP) (result net.IP) {
	result = make([]byte, len(ip)) // start off with a nice empty ip of proper length

	borrow := true
	for i := len(ip) - 1; i >= 0; i-- {
		result[i] = ip[i]
		if borrow {
			result[i]--
			if result[i] != 255 { // if we overflowed, we'd end up here
				borrow = false
			}
		}
	}
	return
}

func getLastIPPrefixInCidr(cidr *string) (*string, error) {
	ipAddr, ipNet, err := net.ParseCIDR(*cidr)
	if err != nil {
		return nil, err
	}
	ipMask, _ := ipNet.Mask.Size()
	ipAddr, err = GetLastIP(ipNet)
	if err != nil {
		return nil, err
	}
	ipAddr = decrementIP(ipAddr)

	return StringPtr(ipAddr.String() + "/" + strconv.Itoa(ipMask)), nil
}

func getLastIPPrefixInIPnet(ipNet net.IPNet) (*string, error) {
	ipMask, _ := ipNet.Mask.Size()
	ipAddr, err := GetLastIP(&ipNet)
	if err != nil {
		return nil, err
	}
	ipAddr = decrementIP(ipAddr)

	return StringPtr(ipAddr.String() + "/" + strconv.Itoa(ipMask)), nil
}

func ip4or6(s string) string {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.':
			return "ipv4"
		case ':':
			return "ipv6"
		}
	}
	return "unknown"
}
