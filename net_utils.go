package uidgenerator

import (
	"net"
	"sync"
)

var netInfo netUtils

type netUtils struct {
	LocalAddress string
}

func init() {
	once := sync.Once{}
	once.Do(func() {
		getLocalInetAddress()
	})
}

func getLocalInetAddress() {
	interfaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			panic(err)
		}
		for _, addr := range addresses {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP
			if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
				continue
			}
			netInfo.LocalAddress = ip.String()
			return
		}
	}
	panic("No validated local address")
}
