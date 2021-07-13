package types

import (
	log "github.com/sirupsen/logrus"
)

type AppLbIpResult struct {
	BgpIp  map[string]map[string][]*IPInfo // app, workloadname -> IPInfo
	LinkIp map[string]map[string][]*IPInfo // app, workloadname -> IPInfo
}

type IPInfo struct {
	Ipv6 string
	Ipv4 string
}

func NewAppLbIpResult() *AppLbIpResult {
	return &AppLbIpResult{
		BgpIp:  map[string]map[string][]*IPInfo{},
		LinkIp: map[string]map[string][]*IPInfo{},
	}
}

func NewIPInfo(ipv4 string, ipv6 string) *IPInfo {
	return &IPInfo{
		Ipv4: ipv4,
		Ipv6: ipv6,
	}
}

func (i *IPInfo) ToString() string {
	return "Ipv4: " + i.Ipv4 + " IPv6: " + i.Ipv6
}

func (a *AppLbIpResult) AddBgpIP(appname string, workloadname string, ipinfo *IPInfo) {
	if _, ok := a.BgpIp[appname]; !ok {
		a.BgpIp[appname] = map[string][]*IPInfo{}
	}
	if _, ok := a.BgpIp[appname][workloadname]; !ok {
		a.BgpIp[appname][workloadname] = []*IPInfo{}
	}
	if len(a.BgpIp[appname][workloadname]) > 1 {
		log.Fatalf("BGPIP - Duplicate add")
	}
	a.BgpIp[appname][workloadname] = append(a.BgpIp[appname][workloadname], ipinfo)
}

func (a *AppLbIpResult) AddLinkIP(appname string, workloadname string, ipinfo *IPInfo) {
	if _, ok := a.LinkIp[appname]; !ok {
		a.LinkIp[appname] = map[string][]*IPInfo{}
	}
	if _, ok := a.LinkIp[appname][workloadname]; !ok {
		a.LinkIp[appname][workloadname] = []*IPInfo{}
	}
	a.LinkIp[appname][workloadname] = append(a.LinkIp[appname][workloadname], ipinfo)
}
