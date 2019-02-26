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
	"errors"
	"fmt"
	"hidevops.io/hiboot/pkg/app/web/context"
	"hidevops.io/hiboot/pkg/at"
	"hidevops.io/hiboot/pkg/factory"
	"hidevops.io/hiboot/pkg/log"
	"hidevops.io/hiboot/pkg/model"
	"hidevops.io/hiboot/pkg/utils/reflector"
	"hidevops.io/hiboot/pkg/utils/replacer"
	"hidevops.io/hiboot/pkg/utils/str"
	"net/http"
	"reflect"
	"strings"
)

const (
	success = "success"
	failed  = "failed"
)

var (
	ErrCanNotInterface = errors.New("response can not interface")
)

type request struct {
	typeName string
	name     string
	fullName string
	kind     reflect.Kind
	genKind  reflect.Kind // e.g. convert int16 to int
	typ      reflect.Type
	iTyp     reflect.Type
	val      reflect.Value
	iVal     reflect.Value
	pathIdx  int
	callback func(ctx context.Context, data interface{}) error
}

type response struct {
	typeName string
	name     string
	kind     reflect.Kind
	typ      reflect.Type
}

type handler struct {
	controller      interface{}
	method          reflect.Method
	ctlVal          reflect.Value
	numIn           int
	numOut          int
	pathParams      []string
	requests        []request
	responses       []response
	lenOfPathParams int
	hasCtxField     bool
	factory         factory.ConfigurableFactory
	runtimeInstance factory.Instance
	contextName     string
	dependencies    []*factory.MetaData
}

type requestSet struct {
	name     string
	callback func(ctx context.Context, data interface{}) error
}

var requestSets []requestSet

func newRequestTypeName(in interface{}) string {
	return reflector.GetName(in)
}

func init() {
	requestSets = []requestSet{
		{newRequestTypeName(new(at.RequestForm)), RequestForm},
		{newRequestTypeName(new(at.RequestParams)), RequestParams},
		{newRequestTypeName(new(at.RequestBody)), RequestBody},
	}
}

func newHandler(factory factory.ConfigurableFactory) *handler {
	return &handler{
		contextName: reflector.GetLowerCamelFullName(new(context.Context)),
		factory:     factory,
	}
}

func clean(in string) (out string) {
	out = strings.Replace(in, "//", "/", -1)
	if strings.Contains(out, "//") {
		out = clean(out)
	}
	lenOfOut := len(out) - 1
	if lenOfOut > 1 && out[lenOfOut:] == "/" {
		out = out[:lenOfOut]
	}
	return
}

func (h *handler) parse(method reflect.Method, object interface{}, path string) {
	//log.Debug("NumIn: ", method.Type.NumIn())
	h.controller = object
	h.method = method
	h.numIn = method.Type.NumIn()
	h.numOut = method.Type.NumOut()
	//h.inputs = make([]reflect.Value, h.numIn)
	h.ctlVal = reflect.ValueOf(object)

	//log.Debugf("method: %v", method.Name)

	// TODO: should parse all of below request and response during router register to improve performance
	path = clean(path)
	//log.Debugf("path: %v", path)
	pps := strings.SplitN(path, "/", -1)
	//log.Debug(pps)
	pp := replacer.ParseVariables(path, compiledRegExp)
	h.pathParams = make([]string, len(pp))
	for i, pathParam := range pp {
		//log.Debugf("pathParm: %v", pathParam[1])
		h.pathParams[i] = pathParam[1]
	}

	h.requests = make([]request, h.numIn)
	objVal := reflect.ValueOf(object)
	idv := reflect.Indirect(objVal)
	objTyp := idv.Type()
	h.requests[0].typeName = objTyp.Name()
	h.requests[0].typ = objTyp
	h.requests[0].val = objVal
	h.hasCtxField = reflector.HasEmbeddedFieldType(object, Controller{})

	lenOfPathParams := len(h.pathParams)
	for i := 1; i < h.numIn; i++ {
		typ := method.Type.In(i)
		iTyp := reflector.IndirectType(typ)

		// parse embedded annotation at.ContextAware
		// append at.ContextAware dependencies
		dp := h.factory.GetInstance(iTyp, factory.MetaData{})
		if dp != nil {
			cdp := dp.(*factory.MetaData)
			if cdp.ContextAware {
				h.dependencies = append(h.dependencies, dp.(*factory.MetaData))
			}
		}

		h.requests[i].typ = typ
		h.requests[i].iTyp = iTyp
		if typ.Kind() == reflect.Slice {
			h.requests[i].kind = reflect.Slice
		} else {
			h.requests[i].kind = iTyp.Kind()
		}
		h.requests[i].val = reflect.New(typ)
		h.requests[i].iVal = reflect.New(iTyp)
		h.requests[i].genKind = reflector.GetKindByValue(h.requests[i].iVal) // TODO:

		pi := i - 1
		if pi < lenOfPathParams {
			h.requests[i].name = pp[pi][1]
			for idx, pv := range pps {
				if pv == pp[pi][0] {
					h.requests[i].pathIdx = idx
					break
				}
			}
		}
		h.requests[i].typeName = iTyp.Name()
		if iTyp.Kind() == reflect.Struct {
			for _, tn := range requestSets {
				if field, ok := iTyp.FieldByName(tn.name); ok && field.Anonymous {
					h.requests[i].typeName = tn.name
					h.requests[i].callback = tn.callback
					break
				}
			}
		}
		h.requests[i].fullName = reflector.GetLowerCamelFullNameByType(iTyp)
	}
	h.lenOfPathParams = lenOfPathParams

	h.responses = make([]response, h.numOut)
	for i := 0; i < h.numOut; i++ {
		typ := method.Type.Out(i)

		h.responses[i].typ = typ
		h.responses[i].kind = typ.Kind()
		h.responses[i].typeName = typ.Name()
		//log.Debug(h.responses[i])
	}
	if path != "" {
		log.Infof("Mapped \"%v\" onto %v.%v()", path, idv.Type(), method.Name)
	}
}

func (h *handler) responseData(ctx context.Context, numOut int, results []reflect.Value) (err error) {
	if numOut == 0 {
		ctx.StatusCode(http.StatusOK)
		return
	}

	result := results[0]
	if !result.CanInterface() {
		err = ErrCanNotInterface
		ctx.ResponseError(err.Error(), http.StatusInternalServerError)
		return
	}

	respVal := result.Interface()
	if respVal == nil {
		//log.Warn("response is nil")
		err = fmt.Errorf("response is nil")
		return
	}

	switch respVal.(type) {
	case string:
		ctx.ResponseString(result.Interface().(string))
	case error:
		respErr := result.Interface().(error)
		ctx.ResponseError(respErr.Error(), http.StatusInternalServerError)
	case model.Response:
		response := respVal.(model.Response)
		if numOut >= 2 {
			var respErr error
			errVal := results[1]
			if errVal.IsNil() {
				respErr = nil
			} else if errVal.Type().Name() == "error" {
				respErr = results[1].Interface().(error)
			}

			if respErr == nil {
				response.SetCode(http.StatusOK)
				response.SetMessage(ctx.Translate(success))
			} else {
				if response.GetCode() == 0 {
					response.SetCode(http.StatusInternalServerError)
				}
				// TODO: output error message directly? how about i18n
				response.SetMessage(ctx.Translate(respErr.Error()))

				// TODO: configurable status code in application.yml
				ctx.StatusCode(response.GetCode())
			}
		}
		ctx.JSON(response)
	case map[string]interface{}:
		ctx.JSON(respVal)
	default:
		ctx.ResponseError("response type is not implemented!", http.StatusInternalServerError)
	}
	return
}

func (h *handler) call(ctx context.Context) {

	var request interface{}
	var reqErr error
	var path string
	var pvs []string
	var runtimeInstance factory.Instance
	//var err error

	if h.lenOfPathParams != 0 {
		path = ctx.Path()
		pvs = strings.SplitN(path, "/", -1)
	}

	if len(h.dependencies) > 0 {
		runtimeInstance, _ = h.factory.InjectContextAwareObjects(ctx, h.dependencies)
	}
	inputs := make([]reflect.Value, h.numIn)
	inputs[0] = h.ctlVal
	for i := 1; i < h.numIn; i++ {
		req := h.requests[i]
		request = req.iVal.Interface()

		if req.callback != nil {
			reqErr = req.callback(ctx, request)
			inputs[i] = reflect.ValueOf(request)
		} else if req.kind == reflect.Interface && model.Context == req.typeName {
			request = ctx
			inputs[i] = reflect.ValueOf(request)
		} else if h.lenOfPathParams != 0 {
			strVal := pvs[req.pathIdx]
			val := str.Convert(strVal, req.kind)
			inputs[i] = reflect.ValueOf(val)
		} else {
			// inject instances
			var inst interface{}
			if runtimeInstance != nil {
				inst = runtimeInstance.Get(req.fullName)
			}
			if inst == nil {
				inst = h.factory.GetInstance(req.fullName)
			}
			if inst != nil {
				inputs[i] = reflect.ValueOf(inst)
			} else {
				msg := fmt.Sprintf("input type: %v is not supported!", req.typ)
				ctx.ResponseError(msg, http.StatusInternalServerError)
				return
			}
		}
	}

	//var respErr error
	var results []reflect.Value
	if reqErr == nil {
		if h.hasCtxField {
			reflector.SetFieldValue(h.controller, "Ctx", ctx)
		}
		// call controller method
		results = h.method.Func.Call(inputs)

		h.responseData(ctx, h.numOut, results)
	}
}
