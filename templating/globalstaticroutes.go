package templating

import "github.com/nokia-paco-automation/paco-parser/types"

type GlobalStaticRoutes struct {
	Data map[string]map[string][]*types.StaticRouteNHG // nodename, networkinstance -> []*staticrouteNHG
}

func NewGlobalStaticRoutes() *GlobalStaticRoutes {
	return &GlobalStaticRoutes{
		Data: map[string]map[string][]*types.StaticRouteNHG{},
	}
}

func (gsr *GlobalStaticRoutes) addEntry(nodename string, networkinstance string, sr *types.StaticRouteNHG) {
	if _, ok := gsr.Data[nodename]; !ok {
		gsr.Data[nodename] = map[string][]*types.StaticRouteNHG{}
	}
	if _, ok := gsr.Data[nodename][networkinstance]; !ok {
		gsr.Data[nodename][networkinstance] = []*types.StaticRouteNHG{}
	}
	gsr.Data[nodename][networkinstance] = append(gsr.Data[nodename][networkinstance], sr)
}
