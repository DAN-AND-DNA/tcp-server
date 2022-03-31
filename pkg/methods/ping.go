package methods

type PingReq struct {
	Body string `json:"body,omitempty"`
}

type PingResp struct {
	Body string `json:"body,omitempty"`
}

func Ping(Request PingReq) (PingResp, error) {
	return PingResp{Body: Request.Body}, nil
}
