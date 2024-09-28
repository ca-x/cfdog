package myip

import (
	"strings"
	"testing"
)

func ipIsNotEmpty(ip string) bool {
	return strings.Contains(ip, ".") || strings.Contains(ip, ":")
}

func TestGetIPv4Address(t *testing.T) {
	tests := []struct {
		name    string
		matchFn func(string) bool
		wantErr bool
	}{
		{name: "basic", matchFn: ipIsNotEmpty, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetIPv4Address()
			t.Logf("got ip =%v ", got)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIPv4Address() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.matchFn(got) {
				t.Errorf("GetIPv4Address() got = %v", got)
			}
		})
	}
}

func TestGetIPv6Address(t *testing.T) {
	tests := []struct {
		name    string
		matchFn func(string) bool
		wantErr bool
	}{
		{name: "basic", matchFn: ipIsNotEmpty, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetIPv6Address()
			t.Logf("got ip =%v ", got)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIPv6Address() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.matchFn(got) {
				t.Errorf("GetIPv6Address() got = %v", got)
			}
		})
	}
}
