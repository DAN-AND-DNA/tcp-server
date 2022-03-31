package methods

import "fmt"

var (
	Error_NoSuchMethod = fmt.Errorf("no such method")
)

func NewRequest(method string) (interface{}, error) {
	switch method {
	case "Ping":
		return &PingReq{}, nil
	case "Login":
		return &LoginReq{}, nil
	case "UpdateLocation":
		return &UpdateLocationReq{}, nil
	}

	return nil, Error_NoSuchMethod
}

func ProcessRequest(method string, rawRequest interface{}) (interface{}, error) {
	switch method {
	case "Ping":
		Copyed := *(rawRequest.(*PingReq))
		return Ping(Copyed)
	case "Login":
		Copyed := *(rawRequest.(*LoginReq))
		return Login(Copyed)
	case "UpdateLocation":
		Copyed := *(rawRequest.(*UpdateLocationReq))
		return UpdateLocation(Copyed)
	default:
		return nil, Error_NoSuchMethod
	}
}
