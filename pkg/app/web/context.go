// Copyright 2018 John Deng (hi.devops.io@gmail.com).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package web

import (
	"github.com/kataras/iris"
	"sync"

	ctx "github.com/kataras/iris/context"
	"github.com/kataras/iris/middleware/i18n"
	"hidevops.io/hiboot/pkg/app/web/context"
	"hidevops.io/hiboot/pkg/model"
	"hidevops.io/hiboot/pkg/utils/mapstruct"
	"hidevops.io/hiboot/pkg/utils/validator"
	"net/http"
)

// Context Create your own custom Context, put any fields you wanna need.
type Context struct {
	iris.Context
}

//NewContext constructor of context.Context
func NewContext(app ctx.Application) context.Context {
	return &Context{
		Context: ctx.NewContext(app),
	}
}

var contextPool = sync.Pool{New: func() interface{} {
	return &Context{}
}}

func acquire(original iris.Context) *Context {
	c := contextPool.Get().(*Context)
	c.Context = original // set the context to the original one in order to have access to iris's implementation.
	return c
}

func release(c *Context) {
	contextPool.Put(c)
}

// Handler will convert our handler of func(*Context) to an iris Handler,
// in order to be compatible with the HTTP API.
func Handler(h func(context.Context)) iris.Handler {
	return func(original iris.Context) {
		c := acquire(original)
		h(c)
		release(c)
	}
}

// Next The second one important if you will override the Context
// with an embedded context.Context inside it.
// Required in order to run the chain of handlers via this "*Context".
func (c *Context) Next() {
	ctx.Next(c)
}

// HTML Override any context's method you want...
// [...]
func (c *Context) HTML(htmlContents string) (int, error) {
	c.Application().Logger().Infof("Executing .HTML function from Context")

	c.ContentType("text/html")
	return c.WriteString(htmlContents)
}

// handle i18n
func (c *Context) translate(message string) string {

	message = i18n.Translate(c, message)

	return message
}

// Translate override base context method Translate to return format if i18n is not enabled
func (c *Context) Translate(format string, args ...interface{}) string {

	msg := c.Context.Translate(format, args...)

	if msg == "" {
		msg = format
	}

	return msg
}

// ResponseString set response
func (c *Context) ResponseString(data string) {
	c.WriteString(c.translate(data))
}

// ResponseBody set response
func (c *Context) ResponseBody(message string, data interface{}) {

	// TODO: check if data is a string, should we translate it?
	response := new(model.BaseResponse)
	response.SetCode(c.GetStatusCode())
	response.SetMessage(c.translate(message))
	response.SetData(data)

	c.JSON(response)
}

// ResponseError response with error
func (c *Context) ResponseError(message string, code int) {

	response := new(model.BaseResponse)
	response.SetCode(code)
	response.SetMessage(c.translate(message))
	if c.ResponseWriter() != nil {
		c.StatusCode(code)
		c.JSON(response)
	}
}

// RequestEx get RequestBody
func requestEx(c context.Context, data interface{}, cb func() error) error {
	if cb != nil {
		err := cb()
		if err != nil {
			c.ResponseError(err.Error(), http.StatusInternalServerError)
			return err
		}

		err = validator.Validate.Struct(data)
		if err != nil {
			c.ResponseError(err.Error(), http.StatusBadRequest)
			return err
		}
	}
	return nil
}

// RequestBody get RequestBody
func RequestBody(c context.Context, data interface{}) error {

	return requestEx(c, data, func() error {
		return c.ReadJSON(data)
	})
}

// RequestForm get RequestFrom
func RequestForm(c context.Context, data interface{}) error {

	return requestEx(c, data, func() error {
		return c.ReadForm(data)
	})
}

// RequestParams get RequestParams
func RequestParams(c context.Context, data interface{}) error {

	return requestEx(c, data, func() error {

		values := c.URLParams()
		if len(values) != 0 {
			return mapstruct.Decode(data, values)
		}
		return nil
	})
}
