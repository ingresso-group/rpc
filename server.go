package rpc

import (
	"net/http"
)

type request struct {
	Id      string      `json:"id"`
	Version string      `json:"jsonrpc"`
	Params  interface{} `json:"params"`
}

type response struct {
	Id      string      `json:"params"`
	Version string      `json:"params"`
	Params  interface{} `json:"params"`
}

type MethodInterface interface {
	Params() interface{}
	Action(request *http.Request, params interface{}) (response interface{}, err error)
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
	return
}

func (service *Service) ServeHTTP(response http.ResponseWriter, request *http.Request) {

}
