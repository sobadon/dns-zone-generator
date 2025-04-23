package zoneutil

import (
	"fmt"
	"net/netip"
	"strings"
)

// https://github.com/scottlaird/netbox2dns/blob/4b6e5e502001c6f9849b9a0104ed791372b21be3/zones.go#L245-L257
func ReverseName4(addr netip.Addr) string {
	b := addr.As4()
	return fmt.Sprintf("%d.%d.%d.%d.in-addr.arpa.", b[3], b[2], b[1], b[0])
}
func ReverseZone4(prefix netip.Prefix) string {
	if prefix.Bits() != 24 {
		panic("invalid prefix size for ipv4. only /24 are supported")
	}
	name32 := ReverseName4(prefix.Addr())
	// 雑にやってしまう 頭の 0. 消す
	zoneName := strings.Split(name32, ".")[1:] // /24 決めうち
	return MustWithDot(strings.Join(zoneName, "."))
}

func ReverseName6(addr netip.Addr) string {
	ret := ""
	b := addr.As16()
	for i := 15; i >= 0; i-- {
		ret += fmt.Sprintf("%x.%x.", b[i]&0xf, (b[i]&0xf0)>>4)
	}
	return ret + "ip6.arpa."
}

func ReverseZone6(prefix netip.Prefix) string {
	idx := -1
	// 雑！
	switch prefix.Bits() {
	case 32:
		idx = 24
	case 40:
		idx = 22
	case 48:
		idx = 20
	default:
		panic("invalid prefix size for ipv6. only /32, /40 and /48 are supported")
	}
	name128 := ReverseName6(prefix.Addr())
	return MustWithDot(strings.Join(strings.Split(name128, ".")[idx:], "."))
}

func MustWithDot(s string) string {
	if strings.HasSuffix(s, ".") {
		return s
	}
	return s + "."
}

func MustWithNoDot(s string) string {
	return strings.TrimSuffix(s, ".")
}

func BuildZoneFileNameForwardZone(zoneName string) string {
	return MustWithNoDot(zoneName) + ".zone"
}

func BuildZoneFileNameReverseZone(zoneIPPrefix netip.Prefix) string {
	if zoneIPPrefix.Addr().Is4() {
		return MustWithNoDot(ReverseZone4(zoneIPPrefix)) + ".zone"
	}
	return MustWithNoDot(ReverseZone6(zoneIPPrefix)) + ".zone"
}
