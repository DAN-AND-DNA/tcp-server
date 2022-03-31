package jsonrpc

import (
	"encoding/binary"
	"tcp-server/pkg/easyjson"
	"tcp-server/pkg/methods"

	jsoniter "github.com/json-iterator/go"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type JsonRpcRequestS struct {
	Id     string `json:"id,omitempty"`
	Method string `json:"method,omitempty"`
}

type JsonRpcRequest struct {
	Id     string      `json:"id,omitempty"`
	Method string      `json:"method,omitempty"`
	Params interface{} `json:"params,omitempty"`
}

type JsonRpcResponseError struct {
	Message string `json:"message,omitempty"`
	Code    int32  `json:"code,omitempty"`
}

type JsonRpcResponse struct {
	Id     string                `json:"id,omitempty"`
	Result interface{}           `json:"result,omitempty"`
	Error  *JsonRpcResponseError `json:"error,omitempty"`
}

func Service(conn *easyjson.EasyTcpConn, inputBuffer []byte) (consumed int, outputBuffer []byte, err error) {
	var bodyLen uint32

	// 1. get total request
	inputBuffer, ok := encodeUint32(inputBuffer, &bodyLen)
	if !ok {
		return 0, nil, nil
	}

	if len(inputBuffer) < (int)(bodyLen) {
		return 0, nil, nil
	}

	bytesRequest := inputBuffer[:bodyLen]

	//log.Println(string(bytesRequest))

	// 2. ecode request
	var tmpRequest JsonRpcRequestS
	err = json.Unmarshal(bytesRequest, &tmpRequest)
	if err != nil {
		return 0, nil, err
	}

	method := tmpRequest.Method
	Id := tmpRequest.Id
	var request JsonRpcRequest
	var response JsonRpcResponse
	response.Id = Id
	consumed = 4 + (int)(bodyLen)

	realRequest, err := methods.NewRequest(method)
	if err != nil {
		response.Result = nil
		// TODO error code
		response.Error = &JsonRpcResponseError{err.Error(), -1}
	} else {
		request.Params = realRequest
		err = json.Unmarshal(bytesRequest, &request)
		if err != nil {
			return 0, nil, err
		}

		// 3. execute reuqest callback
		realResponse, err := methods.ProcessRequest(method, realRequest)
		if err != nil {
			response.Result = nil
			// TODO error code
			response.Error = &JsonRpcResponseError{err.Error(), -2}
		} else {
			response.Id = request.Id
			response.Result = realResponse
		}
	}

	// 4. decode response
	bytesResponse, err := json.Marshal(&response)
	if err != nil {
		return 0, nil, err
	}

	//log.Println(string(bytesResponse))

	outputBuffer = make([]byte, len(bytesResponse)+4)
	copy(decodeUint32(outputBuffer, (uint32)(len(bytesResponse))), bytesResponse)

	return consumed, outputBuffer, nil
}

func encodeUint32(buffer []byte, outInt *uint32) ([]byte, bool) {
	if len(buffer) < 4 {
		return buffer, false
	}

	*outInt = binary.LittleEndian.Uint32(buffer)
	buffer = buffer[4:]
	return buffer, true
}

func decodeUint32(buffer []byte, intInt uint32) []byte {
	binary.LittleEndian.PutUint32(buffer, intInt)
	return buffer[4:]
}

func encodeString(buffer []byte, outString *string, strLen uint32) ([]byte, bool) {

	if len(buffer) < (int)(strLen) {
		return buffer, false
	}

	*outString = string(buffer[:strLen])

	return buffer[strLen:], true
}

func decodeString(buffer []byte, inString string) []byte {
	src := []byte(inString)
	copy(buffer, src)

	return buffer[:len(src)]
}

func encodeBytes(buffer []byte, outBytes []byte, bytesLen uint32) ([]byte, bool) {
	if len(buffer) < (int)(bytesLen) {
		return buffer, false
	}

	outBytes = buffer[:bytesLen]

	return buffer[bytesLen:], true
}
