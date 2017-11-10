package deployments

import (
	"fmt"
	"path"

	"novaforge.bull.com/starlings-janus/janus/helper/collections"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
)

type typeMissingError struct {
	name         string
	deploymentID string
}

func (e typeMissingError) Error() string {
	return fmt.Sprintf("Looking for a type %q that do not exists in deployment %q.", e.name, e.deploymentID)
}

// IsTypeMissingError checks if the given error is a TypeMissing error
func IsTypeMissingError(err error) bool {
	cause := errors.Cause(err)
	_, ok := cause.(typeMissingError)
	return ok
}

// GetParentType returns the direct parent type of a given type using the 'derived_from' attributes
//
// An empty string denotes a root type
func GetParentType(kv *api.KV, deploymentID, typeName string) (string, error) {
	typePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", typeName)
	// Check if node type exist
	if kvps, _, err := kv.List(typePath+"/", nil); err != nil {
		return "", errors.Wrap(err, "Consul access error: ")
	} else if kvps == nil || len(kvps) == 0 {
		return "", errors.WithStack(typeMissingError{name: typeName, deploymentID: deploymentID})
	}

	kvp, _, err := kv.Get(path.Join(typePath, "derived_from"), nil)
	if err != nil {
		return "", errors.Wrap(err, "Consul access error: ")
	}
	if kvp == nil || len(kvp.Value) == 0 {
		return "", nil
	}
	return string(kvp.Value), nil
}

// IsTypeDerivedFrom traverses 'derived_from' to check if type derives from another type
func IsTypeDerivedFrom(kv *api.KV, deploymentID, nodeType, derives string) (bool, error) {
	if nodeType == derives {
		return true, nil
	}
	parent, err := GetParentType(kv, deploymentID, nodeType)
	if err != nil || parent == "" {
		return false, err
	}
	return IsTypeDerivedFrom(kv, deploymentID, parent, derives)
}

// GetTypes returns the names of the different types for a given deployment.
func GetTypes(kv *api.KV, deploymentID string) ([]string, error) {
	names := make([]string, 0)
	types, _, err := kv.Keys(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types")+"/", "/", nil)
	if err != nil {
		return names, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for _, t := range types {
		names = append(names, path.Base(t))
	}
	return names, nil
}

// GetTypeProperties returns the list of properties defined in a given type
//
// It lists only properties defined in the given type not in its parent types.
func GetTypeProperties(kv *api.KV, deploymentID, typeName string) ([]string, error) {
	return getTypeAttributesOrProperties(kv, deploymentID, typeName, "properties")
}

// GetTypeAttributes returns the list of attributes defined in a given type
//
// It lists only attributes defined in the given type not in its parent types.
func GetTypeAttributes(kv *api.KV, deploymentID, typeName string) ([]string, error) {
	return getTypeAttributesOrProperties(kv, deploymentID, typeName, "attributes")
}

func getTypeAttributesOrProperties(kv *api.KV, deploymentID, typeName, paramType string) ([]string, error) {
	result, _, err := kv.Keys(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology/types", typeName, paramType)+"/", "/", nil)
	if err != nil {
		return nil, errors.Wrap(err, consulutil.ConsulGenericErrMsg)
	}
	for i := range result {
		result[i] = path.Base(result[i])
	}
	return result, nil
}

// TypeHasProperty returns true if the type has a property named propertyName defined
func TypeHasProperty(kv *api.KV, deploymentID, typeName, propertyName string) (bool, error) {
	props, err := GetTypeProperties(kv, deploymentID, typeName)
	if err != nil {
		return false, err
	}
	return collections.ContainsString(props, propertyName), nil
}

// TypeHasAttribute returns true if the type has a attribute named attributeName defined
func TypeHasAttribute(kv *api.KV, deploymentID, typeName, attributeName string) (bool, error) {
	attrs, err := GetTypeAttributes(kv, deploymentID, typeName)
	if err != nil {
		return false, err
	}
	return collections.ContainsString(attrs, attributeName), nil
}

// getTypeDefaultProperty checks if a type has a default value for a given property.
//
// It returns true if a default value is found false otherwise as first return parameter.
// If no default value is found in a given type then the derived_from hierarchy is explored to find the default value.
// The second boolean result indicates if the result is a TOSCA Function that should be evaluated in the caller context.
func getTypeDefaultProperty(kv *api.KV, deploymentID, typeName, propertyName string, nestedKeys ...string) (bool, string, bool, error) {
	return getTypeDefaultAttributeOrProperty(kv, deploymentID, typeName, propertyName, true, nestedKeys...)
}

// getTypeDefaultAttribute checks if a type has a default value for a given attribute.
//
// It returns true if a default value is found false otherwise as first return parameter.
// If no default value is found in a given type then the derived_from hierarchy is explored to find the default value.
// The second boolean result indicates if the result is a TOSCA Function that should be evaluated in the caller context.
func getTypeDefaultAttribute(kv *api.KV, deploymentID, typeName, attributeName string, nestedKeys ...string) (bool, string, bool, error) {
	return getTypeDefaultAttributeOrProperty(kv, deploymentID, typeName, attributeName, false, nestedKeys...)
}

// getTypeDefaultProperty checks if a type has a default value for a given property or attribute.
// It returns true if a default value is found false otherwise as first return parameter.
// If no default value is found in a given type then the derived_from hierarchy is explored to find the default value.
// The second boolean result indicates if the result is a TOSCA Function that should be evaluated in the caller context.
func getTypeDefaultAttributeOrProperty(kv *api.KV, deploymentID, typeName, propertyName string, isProperty bool, nestedKeys ...string) (bool, string, bool, error) {

	// If this type doesn't contains the property lets continue to explore the type hierarchy
	var hasProp bool
	var err error
	if isProperty {
		hasProp, err = TypeHasProperty(kv, deploymentID, typeName, propertyName)
	} else {
		hasProp, err = TypeHasAttribute(kv, deploymentID, typeName, propertyName)
	}
	if err != nil {
		return false, "", false, err
	}
	if hasProp {
		typePath := path.Join(consulutil.DeploymentKVPrefix, deploymentID, "topology", "types", typeName)
		var t string
		if isProperty {
			t = "properties"
		} else {
			t = "attributes"
		}
		defaultPath := path.Join(typePath, t, propertyName, "default")

		baseDataType, err := getTypePropertyOrAttributeDataType(kv, deploymentID, typeName, propertyName, isProperty)
		if err != nil {
			return false, "", false, err
		}

		found, result, isFunction, err := getValueAssignmentWithoutResolve(kv, deploymentID, defaultPath, baseDataType, nestedKeys...)
		if err != nil {
			return false, "", false, errors.Wrapf(err, "Failed to get default %s %q for type %q", t, propertyName, typeName)
		}
		if found {
			return true, result, isFunction, nil
		}
	}
	// No default in this type
	// Lets look at parent type
	parentType, err := GetParentType(kv, deploymentID, typeName)
	if err != nil {
		return false, "", false, err
	}
	if parentType == "" {
		return false, "", false, nil
	}
	return getTypeDefaultAttributeOrProperty(kv, deploymentID, parentType, propertyName, isProperty, nestedKeys...)
}
