package rpc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
)

type responseError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Data    string `json:"data,omitempty"`
}

type requestData struct {
	Id      int         `json:"id"`
	Version string      `json:"jsonrpc"`
	Params  interface{} `json:"params"`
	Method  string      `json:"method"`
}

type responseData struct {
	Id      int            `json:"id,omitempty"`
	Version string         `json:"jsonrpc"`
	Result  interface{}    `json:"result,omitempty"`
	Error   *responseError `json:"error,omitempty"`
}

type ParametersInterface interface {
	Validate() error
}

type MethodInterface interface {
	Params() ParametersInterface
	Action(request *http.Request, params ParametersInterface) (response interface{}, err error)
}

type Service struct {
	methods map[string]MethodInterface
	accept  []string
}

func NewService() (service *Service) {
	service = new(Service)
	service.methods = map[string]MethodInterface{}
	service.accept = []string{"application/json", "text/json"}
	return
}

func (service *Service) RegisterMethod(name string, method MethodInterface) (err error) {
	_, exists := service.methods[name]
	if exists {
		return fmt.Errorf("rpc: method name `%s` has already been registered on this service", name)
	}
	service.methods[name] = method
	return
}

func (service *Service) handleRequest(request requestData, response *responseData, rawRequest *http.Request, wg *sync.WaitGroup) {
	defer wg.Done()
	method, ok := service.methods[request.Method]

	if !ok {
		response.Error = &responseError{
			Code:    -32601,
			Message: fmt.Sprintf("rpc: Method name `%s` does not exist", request.Method),
		}
		return
	}

	params := method.Params()
	paramData, err := json.Marshal(request.Params)
	err = json.Unmarshal(paramData, params)

	if err != nil {
		response.Error = &responseError{
			Code:    -32602,
			Message: err.Error(),
		}
		return
	}

	err = params.Validate()

	if err != nil {
		response.Error = &responseError{
			Code:    -32602,
			Message: err.Error(),
		}
		return
	}

	result, err := method.Action(rawRequest, params)

	if err != nil {
		response.Error = &responseError{
			Code:    -32603,
			Message: err.Error(),
		}
		return
	}

	response.Result = result
	return
}

func (service *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		errR := responseData{
			Version: "2.0",
			Error: &responseError{
				Code:    -32700,
				Message: "rpc: invalid HTTP method",
			},
		}
		data, _ := json.Marshal(errR)
		w.Write(data)
		return
	}

	requests := []requestData{}
	responses := []*responseData{}

	data, err := ioutil.ReadAll(r.Body)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errR := responseData{
			Version: "2.0",
			Error: &responseError{
				Code:    -32700,
				Message: err.Error(),
			},
		}
		data, _ = json.Marshal(errR)
		w.Write(data)
		return
	}

	err = json.Unmarshal(data, &requests)

	var single bool

	if err != nil {
		singleRequest := requestData{}
		sErr := json.Unmarshal(data, &singleRequest)
		if sErr != nil {
			w.WriteHeader(http.StatusBadRequest)
			errR := responseData{
				Version: "2.0",
				Error: &responseError{
					Code:    -32700,
					Message: err.Error(),
				},
			}
			data, _ = json.Marshal(errR)
			w.Write(data)
			return
		}
		single = true
		requests = append(requests, singleRequest)
	}

	var wg sync.WaitGroup

	for _, request := range requests {
		response := new(responseData)
		response.Id = request.Id
		response.Version = "2.0"
		responses = append(responses, response)
		wg.Add(1)
		go service.handleRequest(request, response, r, &wg)
	}

	wg.Wait()

	if single && len(responses) == 1 {
		data, err = json.Marshal(responses[0])
	} else {
		data, err = json.Marshal(responses)
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errR := responseData{
			Version: "2.0",
			Error: &responseError{
				Code:    -32603,
				Message: err.Error(),
			},
		}
		data, _ = json.Marshal(errR)
		w.Write(data)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
