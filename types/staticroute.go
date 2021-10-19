package types

type StaticRouteNHG struct {
	Prefix     string
	RType      string
	CnfName    string
	IpVersion  string // "v4" or "v6"
	WlName     string
	VlanID     int
	TargetLeaf string
	NHGroup    *NHGroup
}

type NHGroup struct {
	Name    string
	Entries map[int]*NHGroupEntry
}

type NHGroupEntry struct {
	Index     int
	NHIp      string
	LocalAddr string
}

func NewStaticRouteNHG(prefix string) *StaticRouteNHG {
	return &StaticRouteNHG{
		Prefix:  prefix,
		NHGroup: NewNHGroup(),
	}
}
func NewNHGroup() *NHGroup {
	return &NHGroup{
		Name:    "",
		Entries: map[int]*NHGroupEntry{},
	}
}
func (sr *StaticRouteNHG) SetNHGroupName(name string) {
	sr.NHGroup.Name = name
}
func (sr *StaticRouteNHG) AddNHGroupEntry(nhge *NHGroupEntry) {
	sr.NHGroup.Entries[nhge.Index] = nhge
}
