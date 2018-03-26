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


// dependencies: ci -> pipeline -> impl

package ci

import (
	"fmt"
	"github.com/hidevopsio/hi/boot/pkg/log"
	"github.com/hidevopsio/hi/cicd/pkg/pipeline"
)

func Run(p pipeline.PipelineInterface) error {
	log.Debug("ci.Run()")
	err := p.EnsureParam()
	if err != nil {
		return fmt.Errorf("failed: %s", err)
	}

	err = p.Build()
	if err != nil {
		return fmt.Errorf("failed: %s", err)
	}

	err = p.RunUnitTest()
	if err != nil {
		return fmt.Errorf("failed: %s", err)
	}

	err = p.Deploy()
	if err != nil {
		return fmt.Errorf("failed: %s", err)
	}

	return nil
}
