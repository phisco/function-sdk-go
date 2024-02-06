/*
Copyright 2021 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package response contains utilities for working with RunFunctionResponses.
package response

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/crossplane/function-sdk-go/errors"
	"github.com/crossplane/function-sdk-go/proto/v1beta1"
	"github.com/crossplane/function-sdk-go/resource"
)

// DefaultTTL is the default TTL for which a response can be cached.
const DefaultTTL = 1 * time.Minute

// To bootstraps a response to the supplied request. It automatically copies the
// desired state from the request.
func To(req *v1beta1.RunFunctionRequest, ttl time.Duration) *v1beta1.RunFunctionResponse {
	return &v1beta1.RunFunctionResponse{
		Meta: &v1beta1.ResponseMeta{
			Tag: req.GetMeta().GetTag(),
			Ttl: durationpb.New(ttl),
		},
		Desired: req.GetDesired(),
		Context: req.GetContext(),
	}
}

// SetContextKey sets context to the supplied key.
func SetContextKey(rsp *v1beta1.RunFunctionResponse, key string, v *structpb.Value) {
	if rsp.GetContext().GetFields() == nil {
		rsp.Context = &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	}
	rsp.Context.Fields[key] = v
}

// SetDesiredCompositeResource sets the desired composite resource in the
// supplied response. The caller must be sure to avoid overwriting the desired
// state that may have been accumulated by previous Functions in the pipeline,
// unless they intend to.
func SetDesiredCompositeResource(rsp *v1beta1.RunFunctionResponse, xr *resource.Composite) error {
	if rsp.GetDesired() == nil {
		rsp.Desired = &v1beta1.State{}
	}
	s, err := resource.AsStruct(xr.Resource)
	rsp.Desired.Composite = &v1beta1.Resource{Resource: s, ConnectionDetails: xr.ConnectionDetails}
	return errors.Wrapf(err, "cannot convert %T to desired composite resource", xr.Resource)
}

// SetDesiredComposedResources sets the desired composed resources in the
// supplied response. The caller must be sure to avoid overwriting the desired
// state that may have been accumulated by previous Functions in the pipeline,
// unless they intend to.
func SetDesiredComposedResources(rsp *v1beta1.RunFunctionResponse, dcds map[resource.Name]*resource.DesiredComposed) error {
	if rsp.GetDesired() == nil {
		rsp.Desired = &v1beta1.State{}
	}
	if rsp.GetDesired().GetResources() == nil {
		rsp.Desired.Resources = map[string]*v1beta1.Resource{}
	}
	for name, dcd := range dcds {
		s, err := resource.AsStruct(dcd.Resource)
		if err != nil {
			return err
		}
		r := &v1beta1.Resource{Resource: s}
		switch dcd.Ready {
		case resource.ReadyUnspecified:
			r.Ready = v1beta1.Ready_READY_UNSPECIFIED
		case resource.ReadyFalse:
			r.Ready = v1beta1.Ready_READY_FALSE
		case resource.ReadyTrue:
			r.Ready = v1beta1.Ready_READY_TRUE
		}
		rsp.Desired.Resources[string(name)] = r
	}
	return nil
}

// RequestExtraResourceByName requests an extra resource by name.
func RequestExtraResourceByName(rsp *v1beta1.RunFunctionResponse, id, name string, gvk schema.GroupVersionKind) error {
	if gvk.Empty() {
		return errors.New("cannot request extra resource by name with empty GVK")
	}
	if id == "" {
		return errors.New("cannot request extra resource by name with empty ID")
	}
	if name == "" {
		return errors.New("cannot request extra resource by empty name with empty name")
	}
	if rsp.GetRequirements() == nil {
		rsp.Requirements = &v1beta1.Requirements{}
	}
	if rsp.GetRequirements().GetExtraResources() == nil {
		rsp.Requirements.ExtraResources = make(map[string]*v1beta1.ResourceSelector)
	}
	rsp.Requirements.ExtraResources[id] = &v1beta1.ResourceSelector{
		ApiVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Match: &v1beta1.ResourceSelector_MatchName{
			MatchName: name,
		},
	}
	return nil
}

// RequestExtraResourceByLabels requests an extra resource by labels.
func RequestExtraResourceByLabels(rsp *v1beta1.RunFunctionResponse, id string, labels map[string]string, gvk schema.GroupVersionKind) error {
	if gvk.Empty() {
		return errors.New("cannot request extra resource by name with empty GVK")
	}
	if id == "" {
		return errors.New("cannot request extra resource by name with empty ID")
	}
	if rsp.GetRequirements() == nil {
		rsp.Requirements = &v1beta1.Requirements{}
	}
	if rsp.GetRequirements().GetExtraResources() == nil {
		rsp.Requirements.ExtraResources = make(map[string]*v1beta1.ResourceSelector)
	}
	rsp.Requirements.ExtraResources[id] = &v1beta1.ResourceSelector{
		ApiVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Match: &v1beta1.ResourceSelector_MatchLabels{
			MatchLabels: &v1beta1.MatchLabels{
				Labels: labels,
			},
		},
	}
	return nil
}

// Fatal adds a fatal result to the supplied RunFunctionResponse.
func Fatal(rsp *v1beta1.RunFunctionResponse, err error) {
	if rsp.GetResults() == nil {
		rsp.Results = make([]*v1beta1.Result, 0, 1)
	}
	rsp.Results = append(rsp.GetResults(), &v1beta1.Result{
		Severity: v1beta1.Severity_SEVERITY_FATAL,
		Message:  err.Error(),
	})
}

// Warning adds a warning result to the supplied RunFunctionResponse.
func Warning(rsp *v1beta1.RunFunctionResponse, err error) {
	if rsp.GetResults() == nil {
		rsp.Results = make([]*v1beta1.Result, 0, 1)
	}
	rsp.Results = append(rsp.GetResults(), &v1beta1.Result{
		Severity: v1beta1.Severity_SEVERITY_WARNING,
		Message:  err.Error(),
	})
}

// Normal adds a normal result to the supplied RunFunctionResponse.
func Normal(rsp *v1beta1.RunFunctionResponse, message string) {
	if rsp.GetResults() == nil {
		rsp.Results = make([]*v1beta1.Result, 0, 1)
	}
	rsp.Results = append(rsp.GetResults(), &v1beta1.Result{
		Severity: v1beta1.Severity_SEVERITY_NORMAL,
		Message:  message,
	})
}

// Normalf adds a normal result to the supplied RunFunctionResponse.
func Normalf(rsp *v1beta1.RunFunctionResponse, format string, a ...any) {
	Normal(rsp, fmt.Sprintf(format, a...))
}
