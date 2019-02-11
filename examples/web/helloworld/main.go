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

// Package helloworld provides the quick start web application example
// main package
package main

// import web starter from hiboot
import (
	"hidevops.io/hiboot/pkg/app/web"
	"hidevops.io/hiboot/pkg/app/web/context"
	"hidevops.io/hiboot/pkg/at"
	"hidevops.io/hiboot/pkg/log"
	"hidevops.io/hiboot/pkg/utils/gotest"
)

// Controller Rest Controller with path /
// RESTful Controller, derived from web.Controller. The context mapping of this controller is '/' by default
type Controller struct {
	// at.RestController or web.Controller must be embedded here
	at.RestController
}

// Get GET /
func (c *Controller) Get(ctx context.Context) string {
	// response
	log.Debugf("Controller.Get() GID: %v ctx: %p", gotest.GetGID(), ctx)
	return "Hello world"
}


// Get GET /
func (c *Controller) GetById(id int, ctx context.Context) {
	// response
	log.Infof("Is ctx stopped: %v", ctx.IsStopped())
	log.Debugf("Controller.GetHello() reqID: %v GID: %v ctx: %p - %p", id, gotest.GetGID(), ctx, ctx.(*web.Context).Context)

	ctx.JSON(struct{Message string}{Message: "Hello world!"})
}

// main function
func main() {
	// create new web application and run it
	web.NewApplication(new(Controller)).
		SetAddCommandLineProperties(false).
		Run()
}
