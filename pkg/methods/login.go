package methods

import "fmt"

type LoginReq struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type LoginResp struct {
	Username string `json:"username,omitempty"`
	HP       int    `json:"hp,omitempty"`
	Level    int    `json:"level,omitempty"`
}

func Login(req LoginReq) (resp LoginResp, err error) {
	if req.Password != "12345678" {
		err = fmt.Errorf("bad password")
		return
	}

	resp.Username = req.Username
	resp.Level = 99
	resp.HP = 2000

	Locations.Delete(req.Username)
	Players.Delete(req.Username)
	return
}
