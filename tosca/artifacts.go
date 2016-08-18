package tosca

import (
	"fmt"
	"novaforge.bull.com/starlings-janus/janus/log"
)

type ArtifactDefMap map[string]ArtifactDefinition

func (adm *ArtifactDefMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Either a map or a seq
	*adm = make(ArtifactDefMap)
	var m map[string]ArtifactDefinition
	if err := unmarshal(&m); err == nil {
		for k, v := range m {
			(*adm)[k] = v
		}
		return nil
	}
	log.Debugf("Resolving in artifacts in Alien format")
	//var l []map[string]interface{}
	var l []ArtifactDefinition
	if err := unmarshal(&l); err != nil {
		return err
	}
	log.Debugf("list: %v", l)
	for _, a := range l {
		(*adm)[a.name] = a
	}
	return nil
}

type ArtifactDefinition struct {
	Type        string `yaml:"type,omitempty"`
	File        string `yaml:"file,omitempty"`
	Description string `yaml:"description,omitempty"`
	Repository  string `yaml:"repository,omitempty"`
	DeployPath  string `yaml:"deploy_path,omitempty"`
	// Extra types used in list (A4C) mode
	name string
}

func (a *ArtifactDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err == nil {
		a.File = s
		return nil
	}
	var str struct {
		Type        string `yaml:"type"`
		File        string `yaml:"file"`
		Description string `yaml:"description,omitempty"`
		Repository  string `yaml:"repository,omitempty"`
		DeployPath  string `yaml:"deploy_path,omitempty"`

		// Extra types
		MimeType string                 `yaml:"mime_type,omitempty"`
		XXX      map[string]interface{} `yaml:",inline"`
	}
	if err := unmarshal(&str); err != nil {
		return err
	}
	log.Debugf("Unmarshalled complex ArtifactDefinition %#v", str)
	a.Type = str.Type
	a.File = str.File
	a.Description = str.Description
	a.Repository = str.Repository
	a.DeployPath = str.DeployPath
	if str.File == "" && len(str.XXX) == 1 {
		for k, v := range str.XXX {
			a.name = k
			a.File = fmt.Sprint(v)
		}
	}
	return nil
}