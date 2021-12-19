package main

import (
	"fmt"
	"log"
	"path/filepath"

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
	Modules map[string]*tfconfig.ModuleCall `json:"modules,omitempty"`

	Files map[string]map[string]map[string]*Resource `json:"files,omitempty"`
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

func (r *rover) GenerateModuleMap(parent *Resource, parentModule string, parentPath string, mapObj *Map, module *tfconfig.Module, config *tfjson.ConfigModule, files map[string]map[string]map[string]*Resource) {

	fmt.Printf("Generating map for module \"%v\"...\n", module.Path)

	prefix := parentModule
	if parentModule != "" {
		prefix = fmt.Sprintf("%s.", prefix)
	}

	if _, ok := files[parentModule]; !ok {
		files[parentModule] = make(map[string]map[string]*Resource)
	}

	// Loop through each resource type and populate graph
	for _, variable := range module.Variables {

		fname := filepath.Base(variable.Pos.Filename)

		// Populate with file if doesn't exist
		r.AddFileIfNotExists(parent, parentModule, fname, files)

		id := fmt.Sprintf("%svar.%s", prefix, variable.Name)

		files[parentModule][fname][id] = &Resource{
			Type:     ResourceTypeVariable,
			Name:     variable.Name,
			Required: &variable.Required,
			Line:     &variable.Pos.Line,
		}

		// Get variable sensitivity
		if _, ok := config.Variables[variable.Name]; ok {
			files[parentModule][fname][id].Sensitive = config.Variables[variable.Name].Sensitive
		}

		if parent != nil {
			parent.Children[fname].Children[id] = files[parentModule][fname][id]
		}
	}

	for _, output := range module.Outputs {

		fname := filepath.Base(output.Pos.Filename)

		// Populate with file if doesn't exist
		r.AddFileIfNotExists(parent, parentModule, fname, files)

		fmt.Printf("%v\n", output.Name)

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

		files[parentModule][fname][id] = oo

		if parent != nil {
			parent.Children[fname].Children[id] = files[parentModule][fname][id]
		}
	}

	for _, resource := range module.ManagedResources {

		fname := filepath.Base(resource.Pos.Filename)

		// Populate with file if doesn't exist
		r.AddFileIfNotExists(parent, parentModule, fname, files)

		id := fmt.Sprintf("%s%s.%s", prefix, resource.Type, resource.Name)

		re := &Resource{
			Type:         ResourceTypeResource,
			Name:         resource.Name,
			ResourceType: resource.Type,
			Provider:     resource.Provider.Name,
			Line:         &resource.Pos.Line,
		}

		if _, ok := r.RSO.Resources[id]; ok {
			fmt.Printf("%v\n", id)
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

		files[parentModule][fname][id] = re
		if parent != nil {
			parent.Children[fname].Children[id] = files[parentModule][fname][id]
		}
	}

	for _, data := range module.DataResources {

		fname := filepath.Base(data.Pos.Filename)

		// Populate with file if doesn't exist
		r.AddFileIfNotExists(parent, parentModule, fname, files)

		id := fmt.Sprintf("%sdata.%s.%s", prefix, data.Type, data.Name)

		files[parentModule][fname][id] = &Resource{
			Type:         ResourceTypeData,
			Name:         data.Name,
			ResourceType: data.Type,
			Provider:     data.Provider.Name,
			Line:         &data.Pos.Line,
		}

		if r.RSO.Resources[id].Change.Actions != nil {
			files[parentModule][fname][id].ChangeAction = Action(string(r.RSO.Resources[id].Change.Actions[0]))

			if len(r.RSO.Resources[id].Change.Actions) > 1 {
				files[parentModule][fname][id].ChangeAction = ActionReplace
			}
		}

		if parent != nil {
			parent.Children[fname].Children[id] = files[parentModule][fname][id]
		}
	}

	for _, mc := range module.ModuleCalls {

		fname := filepath.Base(mc.Pos.Filename)

		// Populate with file if doesn't exist
		r.AddFileIfNotExists(parent, parentModule, fname, files)

		id := fmt.Sprintf("%smodule.%s", prefix, mc.Name)

		files[parentModule][fname][id] = &Resource{
			Type:     ResourceTypeModule,
			Name:     mc.Name,
			Source:   mc.Source,
			Version:  mc.Version,
			Line:     &mc.Pos.Line,
			Children: map[string]*Resource{},
		}

		if parent != nil {
			parent.Children[fname].Children[id] = files[parentModule][fname][id]
		}

		childPath := mc.Source
		if parentPath != "" {
			childPath = fmt.Sprintf("%s/%s", parentPath, childPath)
		}

		child, _ := tfconfig.LoadModule(childPath)

		r.GenerateModuleMap(files[parentModule][fname][id], id, childPath, mapObj, child, config.ModuleCalls[mc.Name].Module, files)

	}

}

func (r *rover) AddFileIfNotExists(module *Resource, parentModule string, fname string, files map[string]map[string]map[string]*Resource) {

	if _, ok := files[parentModule][fname]; !ok {
		files[parentModule][fname] = make(map[string]*Resource)

	}

	if _, ok := files[parentModule][fname][fname]; !ok && parentModule != "" {

		files[parentModule][fname][fname] = &Resource{
			Type:     ResourceTypeFile,
			Name:     fname,
			Children: map[string]*Resource{},
		}
	}

	if module != nil {
		module.Children[fname] = files[parentModule][fname][fname]
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

	mapObj := &Map{
		Path:              r.Config.Path,
		RequiredProviders: r.Config.RequiredProviders,
		RequiredCore:      r.Config.RequiredCore,
		// ProviderConfigs:   module.ProviderConfigs,
		Modules: make(map[string]*tfconfig.ModuleCall),
	}

	files := make(map[string]map[string]map[string]*Resource)

	r.GenerateModuleMap(nil, "", "", mapObj, r.Config, r.Plan.Config.RootModule, files)

	mapObj.Files = files

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
