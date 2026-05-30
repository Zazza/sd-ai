package serverclient

import (
	"context"
	"fmt"
	"time"

	"github.com/grandcat/zeroconf"
)

type DiscoveredServer struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	IPAddress string `json:"ip_address"`
}

func DiscoverServers(ctx context.Context, timeout time.Duration) ([]DiscoveredServer, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("mDNS resolver init: %w", err)
	}

	entries := make(chan *zeroconf.ServiceEntry)
	var servers []DiscoveredServer

	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			srv := DiscoveredServer{
				Name: entry.Instance,
				Host: entry.HostName,
				Port: entry.Port,
			}
			if len(entry.AddrIPv4) > 0 {
				srv.IPAddress = entry.AddrIPv4[0].String()
			} else if len(entry.AddrIPv6) > 0 {
				srv.IPAddress = entry.AddrIPv6[0].String()
			}
			servers = append(servers, srv)
		}
	}(entries)

	searchCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := resolver.Browse(searchCtx, "_sd-studio._tcp", "local.", entries); err != nil {
		return nil, fmt.Errorf("mDNS browse: %w", err)
	}

	<-searchCtx.Done()
	return servers, nil
}
