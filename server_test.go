package rpc

import (
	"bytes"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"io"
	"net/http"
	"testing"
)

type fakeParams struct {
	Foo string `json:"foo"`
	Bar int    `json:"bar"`
}

func (params *fakeParams) Validate() error {
	return nil
}

type fakeMethod struct {
	id string
}

func (method *fakeMethod) Params() ParametersInterface {
	return new(fakeParams)
}

func (method *fakeMethod) Action(r *http.Request, p ParametersInterface) (response interface{}, err error) {
	params := p.(*fakeParams)
	response = map[string]string{params.Foo: "foo", "bar": fmt.Sprintf("I LIKE %d BARS", params.Bar)}
	return
}

type fakeBody struct {
	io.Reader
}

func (fakeBody) Close() error { return nil }

type fakeBrokenBody struct {
	io.Reader
}

func (fakeBrokenBody) Close() error             { return nil }
func (fakeBrokenBody) Read([]byte) (int, error) { return 0, fmt.Errorf("test: it's broken!") }

type fakeResponse struct {
	headers http.Header
	body    []byte
	status  int
}

func (response *fakeResponse) Header() http.Header {
	return response.headers
}

func (response *fakeResponse) Write(content []byte) (int, error) {
	if response.status == 0 {
		response.WriteHeader(http.StatusOK)
	}
	_, ok := response.headers["Content-Type"]
	if !ok {
		response.headers["Content-Type"] = []string{"text/plain"}
	}

	response.body = append(response.body, content...)
	return len(content), nil
}

func (response *fakeResponse) WriteHeader(status int) {
	response.status = status
}

func (response *fakeResponse) Body() string {
	return string(response.body)
}

func TestServiceServeHTTP(t *testing.T) {
	Convey("Given a service with two methods `FooBar` and `BarFoo`", t, func() {
		service := Service{
			methods: map[string]MethodInterface{
				"FooBar": &fakeMethod{id: "foo"},
				"BarFoo": &fakeMethod{id: "foo"},
			},
		}
		Convey("Given an http request containing an invalid body", func() {
			request := http.Request{
				Method: "POST",
				Body:   fakeBrokenBody{bytes.NewBufferString("")},
			}

			Convey("When the http request is served", func() {
				response := fakeResponse{
					headers: map[string][]string{},
				}
				service.ServeHTTP(&response, &request)

				Convey("Then the response should have the 200 status code", func() {
					So(response.status, ShouldEqual, http.StatusBadRequest)
				})

				Convey("Then the response should have the json content type", func() {
					So(response.Header().Get("Content-Type"), ShouldEqual, "application/json")
				})

				Convey("Then the response body should be the results of the request", func() {
					expected := `{"jsonrpc":"2.0","error":{"code":-32700,"message":"test: it's broken!"}}`
					So(response.Body(), ShouldEqual, expected)
				})

			})
		})
		Convey("Given an http request containing an invalid JSONRPC request", func() {
			request := http.Request{
				Method: "POST",
				Body: fakeBody{bytes.NewBufferString(`
				ASDKLASDJLAKSJDLKASJADS
				`)},
			}

			Convey("When the http request is served", func() {
				response := fakeResponse{
					headers: map[string][]string{},
				}
				service.ServeHTTP(&response, &request)

				Convey("Then the response should have the 200 status code", func() {
					So(response.status, ShouldEqual, http.StatusBadRequest)
				})

				Convey("Then the response should have the json content type", func() {
					So(response.Header().Get("Content-Type"), ShouldEqual, "application/json")
				})

				Convey("Then the response body should be the results of the request", func() {
					expected := `{"jsonrpc":"2.0","error":{"code":-32700,"message":"invalid character 'A' looking for beginning of value"}}`
					So(response.Body(), ShouldEqual, expected)
				})

			})
		})
		Convey("Given a non post request", func() {
			request := http.Request{
				Method: "GET",
			}

			Convey("When the http request is served", func() {
				response := fakeResponse{
					headers: map[string][]string{},
				}
				service.ServeHTTP(&response, &request)

				Convey("Then the response should have the 400 status code", func() {
					So(response.status, ShouldEqual, http.StatusBadRequest)
				})

				Convey("Then the response should have the json content type", func() {
					So(response.Header().Get("Content-Type"), ShouldEqual, "application/json")
				})

				Convey("Then the response body should be the results of the request", func() {
					expected := `{"jsonrpc":"2.0","error":{"code":-32700,"message":"rpc: invalid HTTP method"}}`
					So(response.Body(), ShouldEqual, expected)
				})

			})
		})
		Convey("Given an http request containing a single JSONRPC request", func() {
			request := http.Request{
				Method: "POST",
				Body: fakeBody{bytes.NewBufferString(`{
					"id": 1,
					"jsonrpc": "2.0",
					"method": "FooBar",
					"params": {
						"foo": "I LIKE BEANS",
						"bar": 10
					}
				}
				`)},
			}

			Convey("When the http request is served", func() {
				response := fakeResponse{
					headers: map[string][]string{},
				}
				service.ServeHTTP(&response, &request)

				Convey("Then the response should have the 200 status code", func() {
					So(response.status, ShouldEqual, http.StatusOK)
				})

				Convey("Then the response should have the json content type", func() {
					So(response.Header().Get("Content-Type"), ShouldEqual, "application/json")
				})

				Convey("Then the response body should be the results of the request", func() {
					expected := `{"id":1,"jsonrpc":"2.0","result":{"I LIKE BEANS":"foo","bar":"I LIKE 10 BARS"}}`
					So(response.Body(), ShouldEqual, expected)
				})

			})
		})
		Convey("Given an http request containing multiple JSONRPC requests", func() {
			request := http.Request{
				Method: "POST",
				Body: fakeBody{bytes.NewBufferString(`[
						{
							"id": 1,
							"jsonrpc": "2.0",
							"method": "FooBar",
							"params": {
								"foo": "I LIKE BEANS",
								"bar": 10
							}
						},
						{
							"id": 2,
							"jsonrpc": "2.0",
							"method": "BarFoo",
							"params": {
								"foo": "BEANS I LIKE",
								"bar": 99
							}
						}
					]
				`)},
			}

			Convey("When the http request is served", func() {
				response := fakeResponse{
					headers: map[string][]string{},
				}
				service.ServeHTTP(&response, &request)

				Convey("Then the response should have the 200 status code", func() {
					So(response.status, ShouldEqual, http.StatusOK)
				})

				Convey("Then the response should have the json content type", func() {
					So(response.Header().Get("Content-Type"), ShouldEqual, "application/json")
				})

				Convey("Then the response body should be the results of the request", func() {
					expected := `[{"id":1,"jsonrpc":"2.0","result":{"I LIKE BEANS":"foo","bar":"I LIKE 10 BARS"}},{"id":2,"jsonrpc":"2.0","result":{"BEANS I LIKE":"foo","bar":"I LIKE 99 BARS"}}]`
					So(response.Body(), ShouldEqual, expected)
				})

			})
		})
	})
}

func TestNewService(t *testing.T) {
	Convey("When a new service is initialised", t, func() {
		service := NewService()
		Convey("Then the method map should exist", func() {
			So(service.methods, ShouldNotBeNil)
		})
		Convey("Then the service should only accept json by default", func() {
			So(service.accept, ShouldContain, "application/json")
			So(service.accept, ShouldContain, "text/json")
		})
	})
}

func TestServiceRegister(t *testing.T) {
	Convey("Given a service with a method named `FooBar`", t, func() {
		service := Service{
			methods: map[string]MethodInterface{
				"FooBar": &fakeMethod{id: "foo"},
			},
		}
		Convey("When a new method with the name `BarFoo` is registered", func() {
			err := service.RegisterMethod("BarFoo", &fakeMethod{id: "bar"})
			Convey("Then `BarFoo` should be regsitered", func() {
				So(err, ShouldBeNil)
				So(service.methods, ShouldContainKey, "BarFoo")
			})
		})
		Convey("When a new method with the name `FooBar` is registered", func() {
			err := service.RegisterMethod("FooBar", &fakeMethod{id: "bar"})
			Convey("Then an error should be returned", func() {
				So(err, ShouldNotBeNil)
			})
			Convey("Then the original method should not be replaced", func() {
				method, ok := service.methods["FooBar"]
				So(ok, ShouldBeTrue)
				So(method.(*fakeMethod).id, ShouldEqual, "foo")
			})
		})
	})
}
