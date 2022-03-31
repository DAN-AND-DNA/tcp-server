package methods

import "sync"

type Location UpdateLocationReq
type UpdateLocationReq struct {
	Username string  `json:"username,omitempty"`
	X        float32 `json:"x,omitempty"`
	Y        float32 `json:"y,omitempty"`
	Z        float32 `json:"z,omitempty"`
	Roll     float32 `json:"roll,omitempty"`
	Pitch    float32 `json:"pitch,omitempty"`
	Yaw      float32 `json:"yaw,omitempty"`
	Vx       float32 `json:"vx,omitempty"`
	Vy       float32 `json:"vy,omitempty"`
	Vz       float32 `json:"vz,omitempty"`
}

type UpdateLocationResp struct {
	Username   string     `json:"username,omitempty"`
	PlayerList []string   `json:"playerlist,omitempty"`
	Locations  []Location `json:"locations,omitempty"`
}

var (
	Locations sync.Map
	Players   sync.Map
)

func UpdateLocation(req UpdateLocationReq) (resp UpdateLocationResp, err error) {
	resp.Username = req.Username

	Locations.Range(func(key interface{}, value interface{}) bool {
		username := key.(string)
		if req.Username != username {
			resp.PlayerList = append(resp.PlayerList, username)
			resp.Locations = append(resp.Locations, value.(Location))
		}
		return true
	})

	Locations.Store(req.Username, (Location)(req))
	return
}
