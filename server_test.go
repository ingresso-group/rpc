package rpc

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"testing"
)

type fakeMethod struct {
	id string
}
type fakeParams struct {
	Foo string `json:"foo"`
	Bar uint   `json:"bar"`
}

func (method *fakeMethod) Params() interface{} {
	return new(fakeParams)
}

func (method *fakeMethod) Action(r *http.Request, p interface{}) (response interface{}, err error) {
	params := p.(fakeParams)
	response = map[string]string{params.Foo: "foo", "bar": fmt.Sprintf("I LIKE %d BARS", params.Bar)}
	return
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
