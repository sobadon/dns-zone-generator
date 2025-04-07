package zoneutil

import (
	"net/netip"
	"testing"
)

func Test_reverseZone4(t *testing.T) {
	tests := []struct {
		name   string
		prefix netip.Prefix
		want   string
	}{
		{name: "a", prefix: netip.MustParsePrefix("192.0.2.0/24"), want: "2.0.192.in-addr.arpa."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReverseZone4(tt.prefix); got != tt.want {
				t.Errorf("reverseZone4() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_reverseZone6(t *testing.T) {
	tests := []struct {
		name   string
		prefix netip.Prefix
		want   string
	}{
		{name: "/32", prefix: netip.MustParsePrefix("2001:db8::/32"), want: "8.b.d.0.1.0.0.2.ip6.arpa."},
		{name: "/48", prefix: netip.MustParsePrefix("2001:db8:1234::/48"), want: "4.3.2.1.8.b.d.0.1.0.0.2.ip6.arpa."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReverseZone6(tt.prefix); got != tt.want {
				t.Errorf("reverseZone6() = %v, want %v", got, tt.want)
			}
		})
	}
}
