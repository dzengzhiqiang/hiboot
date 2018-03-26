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


package k8s

import (
	"k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"fmt"
	"k8s.io/apimachinery/pkg/util/intstr"
	"github.com/hidevopsio/hi/boot/pkg/log"
	"github.com/hidevopsio/hi/cicd/pkg/pipeline"
)

func int32Ptr(i int32) *int32 { return &i }


// @Title Deploy
// @Description deploy application
// @Param pipeline
// @Return error
func Deploy(pipeline *pipeline.Pipeline) (string, error) {

	log.Debug(pipeline)

	deploySpec := &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: pipeline.App,
		},
		Spec: v1beta1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Strategy: v1beta1.DeploymentStrategy{
				Type: v1beta1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &v1beta1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: int32(0),
					},
					MaxSurge: &intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: int32(1),
					},
				},
			},
			RevisionHistoryLimit: int32Ptr(10),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: pipeline.App,
					Labels: map[string]string{
						"app": pipeline.App,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  pipeline.App,
							Image: pipeline.DockerRegistry + "/" + pipeline.Project + "/" + pipeline.App + ":" + pipeline.ImageTag,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 8080,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "APP_PROFILES_ACTIVE",
									Value: pipeline.Profile,
								},
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
				},
			},
		},
	}
	log.Debug(deploySpec)

	// Create Deployment
	//Client.ClientSet.ExtensionsV1beta1().Deployments()
	deployments := ClientSet.AppsV1beta1().Deployments(pipeline.Project)
	log.Info("Update or Create Deployment...")
	result, err := deployments.Update(deploySpec)
	var retVal string
	switch {
	case err == nil:
		log.Info("Deployment updated")
	case !errors.IsNotFound(err):
		_, err = deployments.Create(deploySpec)
		retVal = fmt.Sprintf("Created deployment %q.\n", result.GetObjectMeta().GetName())
		log.Info(retVal)
	default:
		return retVal, fmt.Errorf("could not update deployment controller: %s", err)
	}

	return retVal, err
}
