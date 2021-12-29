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

type Action string
type ResourceType string

const (
	ResourceTypeFile     ResourceType = "file"
	ResourceTypeLocal    ResourceType = "locals"
	ResourceTypeVariable ResourceType = "variable"
	ResourceTypeOutput   ResourceType = "output"
	ResourceTypeResource ResourceType = "resource"
	ResourceTypeData     ResourceType = "data"
	ResourceTypeModule   ResourceType = "module"
	DefaultFileName      string       = "unknown"
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

func (r *rover) GenerateModuleMap(parent *Resource, parentModule string) {

	childIndex := regexp.MustCompile(`\[[^[\]]*\]$`)
	matchBrackets := regexp.MustCompile(`\[[^\[\]]*\]`)

	states := r.RSO.States
	configs := r.RSO.Configs

	prefix := parentModule
	if parentModule != "" {
		prefix = fmt.Sprintf("%s.", prefix)
	}

	parentConfig := matchBrackets.ReplaceAllString(parentModule, "")
	parentConfigured := configs[parentConfig] != nil

	if parentConfigured && !states[parentModule].IsParent {
		for oName, o := range configs[parentConfig].Module.Outputs {
			fname := filepath.Base(o.Pos.Filename)
			oid := fmt.Sprintf("%soutput.%s", prefix, oName)
			out := &Resource{
				Type:      ResourceTypeOutput,
				Name:      oName,
				Sensitive: o.Sensitive,
				Line:      &o.Pos.Line,
			}
			r.AddFileIfNotExists(parent, parentModule, fname)

			parent.Children[fname].Children[oid] = out
		}

		for vName, v := range configs[parentConfig].Module.Variables {
			fname := filepath.Base(v.Pos.Filename)
			vid := fmt.Sprintf("%svar.%s", prefix, vName)
			va := &Resource{
				Type:     ResourceTypeVariable,
				Name:     vName,
				Required: &v.Required,
				Line:     &v.Pos.Line,
			}

			r.AddFileIfNotExists(parent, parentModule, fname)

			parent.Children[fname].Children[vid] = va

		}
	}

	for id, rs := range states[parentModule].Children {

		configId := matchBrackets.ReplaceAllString(id, "")
		configured := configs[parentConfig] != nil && configs[configId] != nil

		re := &Resource{
			Type:     rs.Type,
			Children: map[string]*Resource{},
		}

		if states[id].Change.Actions != nil {

			re.ChangeAction = Action(string(states[id].Change.Actions[0]))
			if len(states[id].Change.Actions) > 1 {
				re.ChangeAction = ActionReplace
			}
		}

		if rs.Type == ResourceTypeResource || rs.Type == ResourceTypeData {
			re.ResourceType = configs[configId].ResourceConfig.Type
			re.Name = configs[configId].ResourceConfig.Name

			for crName, cr := range states[id].Children {

				if re.Children == nil {
					re.Children = make(map[string]*Resource)
				}

				tcr := &Resource{
					Type: rs.Type,
					Name: strings.TrimPrefix(crName, fmt.Sprintf("%s%s.", prefix, re.ResourceType)),
				}

				if cr.Change.Actions != nil {
					tcr.ChangeAction = Action(string(cr.Change.Actions[0]))

					if len(cr.Change.Actions) > 1 {
						tcr.ChangeAction = ActionReplace
					}
				}

				re.Children[crName] = tcr
			}

			if configured {

				var fname string
				ind := fmt.Sprintf("%s.%s", re.ResourceType, re.Name)

				if rs.Type == ResourceTypeData {
					ind = fmt.Sprintf("data.%s", ind)
					fname = filepath.Base(configs[parentConfig].Module.DataResources[ind].Pos.Filename)
					re.Line = &configs[parentConfig].Module.DataResources[ind].Pos.Line
				} else if rs.Type == ResourceTypeResource {
					fname = filepath.Base(configs[parentConfig].Module.ManagedResources[ind].Pos.Filename)
					re.Line = &configs[parentConfig].Module.ManagedResources[ind].Pos.Line
				}

				r.AddFileIfNotExists(parent, parentModule, fname)

				parent.Children[fname].Children[id] = re

			} else {

				r.AddFileIfNotExists(parent, parentModule, DefaultFileName)

				parent.Children[DefaultFileName].Children[id] = re
			}

		} else if rs.Type == ResourceTypeModule {
			re.Name = strings.Split(id, ".")[len(strings.Split(id, "."))-1]

			if configured && !childIndex.MatchString(id) {
				fmt.Printf("%v - %v\n", re.Name, configs[parentConfig].Module.ModuleCalls)
				fname := filepath.Base(configs[parentConfig].Module.ModuleCalls[matchBrackets.ReplaceAllString(re.Name, "")].Pos.Filename)
				re.Line = &configs[parentConfig].Module.ModuleCalls[matchBrackets.ReplaceAllString(re.Name, "")].Pos.Line

				r.AddFileIfNotExists(parent, parentModule, fname)

				parent.Children[fname].Children[id] = re

			} else {

				r.AddFileIfNotExists(parent, parentModule, DefaultFileName)

				parent.Children[DefaultFileName].Children[id] = re
			}

			r.GenerateModuleMap(re, id)

		}

		if configured && !(re.Type == ResourceTypeModule && childIndex.MatchString(id)) {
			expressions := map[string]*tfjson.Expression{}

			if re.Type == ResourceTypeResource {
				expressions = configs[configId].ResourceConfig.Expressions
			} else if re.Type == ResourceTypeModule {
				expressions = configs[configId].ModuleConfig.Expressions
			} else if re.Type == ResourceTypeOutput {
				expressions["exp"] = configs[configId].OutputConfig.Expression
			}

			// Add locals
			for _, reValues := range expressions {
				for _, dependsOnR := range reValues.References {
					ref := &Resource{}
					if strings.HasPrefix(dependsOnR, "local.") {
						// Append local variable
						ref.Type = ResourceTypeLocal
						ref.Name = strings.TrimPrefix(dependsOnR, "local.")
						rid := fmt.Sprintf("%s%s", prefix, dependsOnR)

						r.AddFileIfNotExists(parent, parentModule, DefaultFileName)

						parent.Children[DefaultFileName].Children[rid] = ref
					}
				}
			}

		}
	}
}

func (r *rover) AddFileIfNotExists(module *Resource, parentModule string, fname string) {

	if _, ok := module.Children[fname]; !ok {

		module.Children[fname] = &Resource{
			Type:     ResourceTypeFile,
			Name:     fname,
			Source:   fmt.Sprintf("%s/%s", module.Source, fname),
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

	r.GenerateModuleMap(rootModule, "")

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
