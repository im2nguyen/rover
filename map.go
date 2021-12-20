package main

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	tfjson "github.com/hashicorp/terraform-json"
)

type ResourceType string
type Action string

const (
	ResourceTypeFile     ResourceType = "file"
	ResourceTypeVariable ResourceType = "variable"
	ResourceTypeOutput   ResourceType = "output"
	ResourceTypeResource ResourceType = "resource"
	ResourceTypeData     ResourceType = "data"
	ResourceTypeModule   ResourceType = "module"
	DefaultFileName      string       = "Resources"
)

const (
	// ActionNoop denotes a no-op operation.
	ActionNoop Action = "no-op"

	// ActionCreate denotes a create operation.
	ActionCreate Action = "create"

	// ActionRead denotes a read operation.
	ActionRead Action = "read"

	// ActionUpdate denotes an update operation.
	ActionUpdate Action = "update"

	// ActionDelete denotes a delete operation.
	ActionDelete Action = "delete"

	// ActionReplace denotes a replace operation.
	ActionReplace Action = "replace"
)

// Map represents the root module
type Map struct {
	Path              string                                   `json:"path"`
	RequiredCore      []string                                 `json:"required_core,omitempty"`
	RequiredProviders map[string]*tfconfig.ProviderRequirement `json:"required_providers,omitempty"`
	// ProviderConfigs   map[string]*tfconfig.ProviderConfig      `json:"provider_configs,omitempty"`
	Root map[string]*Resource `json:"root,omitempty"`
}

// Resource is a modified tfconfig.Resource
type Resource struct {
	Type ResourceType `json:"type"`
	Name string       `json:"name"`
	Line *int         `json:"line,omitempty"`

	Children map[string]*Resource `json:"children,omitempty"`

	// Resource
	ChangeAction Action `json:"change_action,omitempty"`
	// Variable and Output
	Required  *bool `json:"required,omitempty"`
	Sensitive bool  `json:"sensitive,omitempty"`
	// Provider and Data
	Provider     string `json:"provider,omitempty"`
	ResourceType string `json:"resource_type,omitempty"`
	// ModuleCall
	Source  string `json:"source,omitempty"`
	Version string `json:"version,omitempty"`
}

// ModuleCall is a modified tfconfig.ModuleCall
type ModuleCall struct {
	Name    string `json:"name"`
	Source  string `json:"source"`
	Version string `json:"version,omitempty"`
	Line    int    `json:"line,omitempty"`
}

func (r *rover) GenerateModuleMap(parent *Resource, parentModule string, parentPath string, module *tfconfig.Module, config *tfjson.ConfigModule) {

	prefix := parentModule
	if parentModule != "" {
		prefix = fmt.Sprintf("%s.", prefix)
	}

	// Loop through each resource type and populate graph
	for _, variable := range module.Variables {

		fname := filepath.Base(variable.Pos.Filename)

		// Populate with file if doesn't exist
		r.AddFileIfNotExists(parent, parentModule, fname)

		id := fmt.Sprintf("%svar.%s", prefix, variable.Name)

		parent.Children[fname].Children[id] = &Resource{
			Type:     ResourceTypeVariable,
			Name:     variable.Name,
			Required: &variable.Required,
			Line:     &variable.Pos.Line,
		}

		// Get variable sensitivity
		if _, ok := config.Variables[variable.Name]; ok {
			parent.Children[fname].Children[id].Sensitive = config.Variables[variable.Name].Sensitive
		}

	}

	for _, output := range module.Outputs {

		fname := filepath.Base(output.Pos.Filename)

		// Populate with file if doesn't exist
		r.AddFileIfNotExists(parent, parentModule, fname)

		id := fmt.Sprintf("%soutput.%s", prefix, output.Name)

		oo := &Resource{
			Type:      ResourceTypeOutput,
			Name:      output.Name,
			Sensitive: output.Sensitive,
			Line:      &output.Pos.Line,
		}

		if _, ok := r.RSO.Outputs[output.Name]; ok {
			if r.RSO.Outputs[output.Name].Change != nil {
				if r.RSO.Outputs[output.Name].Change.Actions != nil {
					oo.ChangeAction = Action(string(r.RSO.Outputs[output.Name].Change.Actions[0]))

					if len(r.RSO.Outputs[output.Name].Change.Actions) > 1 {
						oo.ChangeAction = ActionReplace
					}
				}
			}
		}

		parent.Children[fname].Children[id] = oo
	}

	for _, resource := range module.ManagedResources {

		fname := filepath.Base(resource.Pos.Filename)

		// Populate with file if doesn't exist
		r.AddFileIfNotExists(parent, parentModule, fname)

		id := fmt.Sprintf("%s%s.%s", prefix, resource.Type, resource.Name)

		re := &Resource{
			Type:         ResourceTypeResource,
			Name:         resource.Name,
			ResourceType: resource.Type,
			Provider:     resource.Provider.Name,
			Line:         &resource.Pos.Line,
		}

		if _, ok := r.RSO.Resources[id]; ok {

			if r.RSO.Resources[id].Change.Actions != nil {

				re.ChangeAction = Action(string(r.RSO.Resources[id].Change.Actions[0]))

				if len(r.RSO.Resources[id].Change.Actions) > 1 {
					re.ChangeAction = ActionReplace
				}
			}
		}

		parent.Children[fname].Children[id] = re

	}

	for _, data := range module.DataResources {

		fname := filepath.Base(data.Pos.Filename)

		// Populate with file if doesn't exist
		r.AddFileIfNotExists(parent, parentModule, fname)

		id := fmt.Sprintf("%sdata.%s.%s", prefix, data.Type, data.Name)

		dr := &Resource{
			Type:         ResourceTypeData,
			Name:         data.Name,
			ResourceType: data.Type,
			Provider:     data.Provider.Name,
			Line:         &data.Pos.Line,
		}

		if _, ok := r.RSO.Resources[id]; ok {

			if r.RSO.Resources[id].Change.Actions != nil {
				dr.ChangeAction = Action(string(r.RSO.Resources[id].Change.Actions[0]))

				if len(r.RSO.Resources[id].Change.Actions) > 1 {
					dr.ChangeAction = ActionReplace
				}
			}
		}

		parent.Children[fname].Children[id] = dr
	}

	fmt.Printf("%v\n", r.RSO.Resources[parentModule])
	for _, childModule := range r.RSO.Resources[parentModule].Module.ChildModules {

		matchBrackets := regexp.MustCompile(`\[[^\[\]]*\]`)
		configId := matchBrackets.ReplaceAllString(childModule.Address, "")
		configId = strings.Split(configId, ".")[len(strings.Split(configId, "."))-1]

		mc := module.ModuleCalls[configId]

		fname := filepath.Base(mc.Pos.Filename)

		// Populate with file if doesn't exist
		r.AddFileIfNotExists(parent, parentModule, fname)

		id := childModule.Address
		name := strings.Split(id, ".")[len(strings.Split(id, "."))-1]

		fmt.Printf("%v\n", id)

		m := &Resource{
			Type:     ResourceTypeModule,
			Name:     name,
			Source:   mc.Source,
			Version:  mc.Version,
			Line:     &mc.Pos.Line,
			Children: map[string]*Resource{},
		}

		parent.Children[fname].Children[id] = m

		childPath := mc.Source
		if parentPath != "" {
			childPath = fmt.Sprintf("%s/%s", parentPath, childPath)
		}

		child, _ := tfconfig.LoadModule(childPath)

		fmt.Printf("%v\n", id)
		if r.RSO.Resources[id].ModuleConfig != nil {
			fmt.Printf("%v\n", r.RSO.Resources[id].ModuleConfig)
		}

		r.GenerateModuleMap(m, id, childPath, child, config.ModuleCalls[mc.Name].Module)

	}

}

func (r *rover) AddFileIfNotExists(module *Resource, parentModule string, fname string) {

	if _, ok := module.Children[fname]; !ok {

		module.Children[fname] = &Resource{
			Type:     ResourceTypeFile,
			Name:     fname,
			Children: map[string]*Resource{},
		}
	}
}

// Generates Map - Overview of files and their resources
// Groups different resource types together
// Defaults to config
func (r *rover) GenerateMap() error {
	log.Println("Generating resource map...")

	if !r.TFConfigExists {
		return r.GenerateMapNoConfig()
	}

	// Root module
	rootModule := &Resource{
		Type:     ResourceTypeModule,
		Name:     "",
		Source:   r.Config.Path,
		Children: map[string]*Resource{},
	}

	r.GenerateModuleMap(rootModule, "", "", r.Config, r.Plan.Config.RootModule)

	mapObj := &Map{
		Path:              r.Config.Path,
		RequiredProviders: r.Config.RequiredProviders,
		RequiredCore:      r.Config.RequiredCore,
		// ProviderConfigs:   module.ProviderConfigs,
		Root: rootModule.Children,
	}

	r.Map = mapObj

	return nil
}

func (r *rover) GenerateMapNoConfig() error {
	/*mapObj := &Map{
		Path:    "Rover Visualization",
		Modules: make(map[string]*tfconfig.ModuleCall),
	}

	files := make(map[string]map[string]*Resource)

	// Loop through each resource type and populate map
	for varName, variable := range r.Plan.Config.RootModule.Variables {
		// Populate with file if doesn't exist
		if _, ok := files[DefaultFileName]; !ok {
			files[DefaultFileName] = make(map[string]*Resource)
		}

		id := fmt.Sprintf("var.%s", varName)

		files[DefaultFileName][id] = &Resource{
			Type:      ResourceTypeVariable,
			Name:      varName,
			Sensitive: variable.Sensitive,
		}
	}

	for outputName, output := range r.Plan.Config.RootModule.Outputs {
		// Populate with file if doesn't exist
		if _, ok := files[DefaultFileName]; !ok {
			files[DefaultFileName] = make(map[string]*Resource)
		}

		id := fmt.Sprintf("output.%s", outputName)

		oo := &Resource{
			Type:      ResourceTypeOutput,
			Name:      outputName,
			Sensitive: output.Sensitive,
		}

		if _, ok := r.RSO.Outputs[outputName]; ok {
			if r.RSO.Outputs[outputName].Change != nil {
				if r.RSO.Outputs[outputName].Change.Actions != nil {
					oo.ChangeAction = Action(string(r.RSO.Outputs[outputName].Change.Actions[0]))

					if len(r.RSO.Outputs[outputName].Change.Actions) > 1 {
						oo.ChangeAction = ActionReplace
					}
				}
			}
		}

		files[DefaultFileName][id] = oo
	}

	// Data resources are included in r.Plan.Config.RootModule.Resources
	for _, resource := range r.Plan.Config.RootModule.Resources {
		// Populate with file if doesn't exist
		if _, ok := files[DefaultFileName]; !ok {
			files[DefaultFileName] = make(map[string]*Resource)
		}

		id := fmt.Sprintf("%s.%s", resource.Type, resource.Name)

		re := &Resource{
			Type:         ResourceTypeResource,
			Name:         resource.Name,
			ResourceType: resource.Type,
			Provider:     resource.ProviderConfigKey,
		}

		// If resource is a data resource...
		if resource.Mode == "data" {
			re.Type = ResourceTypeData
			id = fmt.Sprintf("data.%s.%s", resource.Type, resource.Name)
		}

		if _, ok := r.RSO.Resources[id]; ok {
			if r.RSO.Resources[id].Change.Actions != nil {
				re.ChangeAction = Action(string(r.RSO.Resources[id].Change.Actions[0]))

				if len(r.RSO.Resources[id].Change.Actions) > 1 {
					re.ChangeAction = ActionReplace
				}
			}

			for crName, cr := range r.RSO.Resources[id].Children {
				if re.Children == nil {
					re.Children = make(map[string]*Resource)
				}

				tcr := &Resource{
					Type: ResourceTypeResource,
					Name: crName,
				}

				if cr.Change.Actions != nil {
					tcr.ChangeAction = Action(string(cr.Change.Actions[0]))

					if len(cr.Change.Actions) > 1 {
						tcr.ChangeAction = ActionReplace
					}
				}

				re.Children[crName] = tcr
			}
		}

		files[DefaultFileName][id] = re
	}

	for modName, mc := range r.Plan.Config.RootModule.ModuleCalls {
		// Populate with file if doesn't exist
		if _, ok := files[DefaultFileName]; !ok {
			files[DefaultFileName] = make(map[string]*Resource)
		}

		id := fmt.Sprintf("module.%s", modName)

		m := &Resource{
			Type:   ResourceTypeModule,
			Name:   modName,
			Source: mc.Source,
		}

		m.Children = make(map[string]*Resource)

		if _, ok := r.RSO.Resources[id]; ok {
			tempChildren := make(map[string]*Resource)

			// Filter through and add configuration
			for _, cr := range r.RSO.Resources[id].ModuleConfig.Module.Resources {
				crName := fmt.Sprintf("%s.%s", id, cr.Address)

				tcr := &Resource{
					Type:         ResourceTypeResource,
					Name:         cr.Name,
					ResourceType: cr.Type,
				}

				if cr.Mode == tfjson.DataResourceMode {
					tcr.Type = ResourceTypeData
				}

				tempChildren[crName] = tcr
			}
			// Filter through and add change action
			for crName, cr := range r.RSO.Resources[id].Children {
				tcr := tempChildren[crName]

				if tcr == nil {
					tcr = &Resource{}
				}

				if tcr.Name == "" {
					tcr.Type = ResourceTypeResource
					tcr.Name = cr.Config.Name
					tcr.ResourceType = cr.Config.Type
				}

				if cr.Change.Actions != nil {
					tcr.ChangeAction = Action(string(cr.Change.Actions[0]))

					if len(cr.Change.Actions) > 1 {
						tcr.ChangeAction = ActionReplace
					}
				}

				// Add resource to module children
				m.Children[crName] = tcr

				// Add parent resource to module children
				parentId := strings.Split(crName, "[")[0]
				if _, ok := tempChildren[parentId]; ok {
					m.Children[parentId] = tempChildren[parentId]
				}
			}
		}

		files[DefaultFileName][id] = m
	}

	mapObj.Files = files

	r.Map = mapObj*/

	return nil
}
