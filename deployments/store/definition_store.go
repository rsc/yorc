// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package store

import (
	"context"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"golang.org/x/sync/errgroup"
	"path"
	"sync"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/ystia/yorc/v4/deployments/internal"
	"github.com/ystia/yorc/v4/helper/collections"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tosca"
)

// BuiltinOrigin is the origin for Yorc builtin
const BuiltinOrigin = "builtin"

const yorcOriginConsulKey = "yorc_origin"

var lock sync.Mutex

var builtinTypes = make([]string, 0)

// getLatestCommonsTypesKeyPaths() returns all the path keys corresponding to the last version of a type
// that is stored under the consulutil.CommonsTypesKVPrefix.
// For example, one path key could be _yorc/commons_types/some_type/2.0.0 if the last version
// of the some_type type is 2.0.0
func getLatestCommonsTypesKeyPaths() ([]string, error) {
	keys, err := storage.GetStore(types.StoreTypeDeployment).Keys(consulutil.CommonsTypesKVPrefix)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	paths := make([]string, 0, len(keys))
	for _, builtinTypesPath := range keys {
		versions, err := storage.GetStore(types.StoreTypeDeployment).Keys(builtinTypesPath)
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}

		if len(versions) == 0 {
			continue
		}
		var maxVersion semver.Version
		for _, v := range versions {
			version, err := semver.Make(path.Base(v))
			if err == nil && version.GTE(maxVersion) {
				maxVersion = version
			}
		}
		typePath := path.Join(builtinTypesPath, maxVersion.String())

		paths = append(paths, typePath)
	}
	return paths, nil
}

// GetCommonsTypesKeyPaths returns the path of builtin types supported by this instance of Yorc
//
// Returned keys are formatted as <consulutil.CommonsTypesKVPrefix>/<name>/<version>
// If this is used from outside a Yorc instance typically a plugin or another app then the latest
// version of each builtin type stored in Consul is assumed
func GetCommonsTypesKeyPaths() []string {
	lock.Lock()
	defer lock.Unlock()
	if len(builtinTypes) == 0 {
		// Not provided at system startup we are probably in an external application used as a lib
		// So let use latest values of each stored builtin types in Consul
		builtinTypes, _ = getLatestCommonsTypesKeyPaths()
	}
	res := make([]string, len(builtinTypes))
	copy(res, builtinTypes)
	return res
}

// CommonDefinition stores a TOSCA definition to the common place
func CommonDefinition(ctx context.Context, definitionName, origin string, definitionContent []byte) error {
	topology := tosca.Topology{}
	err := yaml.Unmarshal(definitionContent, &topology)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal TOSCA definition %q", definitionName)
	}
	errGroup, ctx := errgroup.WithContext(ctx)
	name := topology.Metadata["template_name"]
	if name == "" {
		return errors.Errorf("Can't store builtin TOSCA definition %q, template_name is missing", definitionName)
	}
	version := topology.Metadata["template_version"]
	if version == "" {
		return errors.Errorf("Can't store builtin TOSCA definition %q, template_version is missing", definitionName)
	}

	topology.Metadata[yorcOriginConsulKey] = origin

	topologyPrefix := path.Join(consulutil.CommonsTypesKVPrefix, name, version)

	func() {
		lock.Lock()
		defer lock.Unlock()
		if !collections.ContainsString(builtinTypes, topologyPrefix) {
			builtinTypes = append(builtinTypes, topologyPrefix)
		}
	}()

	keys, err := storage.GetStore(types.StoreTypeDeployment).Keys(topologyPrefix)
	if err != nil {
		return errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	if len(keys) > 0 {
		log.Printf("Do not storing existing topology definition: %q version: %q", name, version)
		return nil
	}
	errGroup.Go(func() error {
		return internal.StoreTopologyTopLevelKeyNames(ctx, topology, topologyPrefix)
	})
	errGroup.Go(func() error {
		return internal.StoreRepositories(ctx, topology, topologyPrefix)
	})
	errGroup.Go(func() error {
		return internal.StoreAllTypes(ctx, topology, topologyPrefix, "")
	})
	return errGroup.Wait()
}

// Deployment stores a whole deployment.
func Deployment(ctx context.Context, topology tosca.Topology, deploymentID, rootDefPath string) error {
	errGroup, ctx := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		return internal.StoreTopology(ctx, errGroup, topology, deploymentID, path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology"), "", "", rootDefPath)
	})

	return errGroup.Wait()
}

// Definition is TOSCA Definition registered in the Yorc as builtin could be comming from Yorc itself or a plugin
type Definition struct {
	Name    string `json:"name"`
	Origin  string `json:"origin"`
	Version string `json:"version"`
}

// GetCommonsDefinitionsList returns the list of commons definitions within Yorc
func GetCommonsDefinitionsList() ([]Definition, error) {
	lock.Lock()
	defer lock.Unlock()
	if len(builtinTypes) == 0 {
		// Not provided at system startup we are probably in an external application used as a lib
		// So let use latest values of each stored builtin types in Consul
		builtinTypes, _ = getLatestCommonsTypesKeyPaths()
	}
	res := make([]Definition, len(builtinTypes))
	for _, p := range builtinTypes {
		d := Definition{}
		d.Version = path.Base(p)
		p = path.Dir(p)
		d.Name = path.Base(p)
		res = append(res, d)
		exist, value, err := consulutil.GetStringValue(path.Join(p, "metadata", yorcOriginConsulKey))
		if err != nil {
			return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
		}
		if exist {
			d.Version = value
		}
	}
	return res, nil
}
