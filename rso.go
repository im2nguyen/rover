package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
)

// ResourcesOverview represents the root module
type ResourcesOverview struct {
	States  map[string]*StateOverview  `json:"states,omitempty"`
	Configs map[string]*ConfigOverview `json:"configs,omitempty"`
}

// ResourceOverview is a modified tfjson.Plan
type StateOverview struct {
	// ChangeAction tfjson.Actions        `json:change_action`
	ConfigId  string                    `json:"config_id,omitempty"`
	Change    tfjson.Change             `json:"change,omitempty"`
	Module    *tfjson.StateModule       `json:"module,omitempty"`
	DependsOn []string                  `json:"depends_on,omitempty"`
	Children  map[string]*StateOverview `json:"children,omitempty"`
}

type ConfigOverview struct {
	ResourceConfig *tfjson.ConfigResource `json:"resource_config,omitempty"`
	ModuleConfig   *tfjson.ModuleCall     `json:"module_config,omitempty"`
	VariableConfig *tfjson.ConfigVariable `json:"variable_config,omitempty"`
	OutputConfig   *tfjson.ConfigOutput   `json:"output_config,omitempty"`
}

func (r *rover) PopulateConfigs(parent string, rso *ResourcesOverview, config *tfjson.ConfigModule) {

	rc := rso.Configs

	prefix := parent
	if prefix != "" {
		prefix = fmt.Sprintf("%s.", prefix)
	}

	// Loop through variable configs
	for variableName, variable := range config.Variables {
		variableName = fmt.Sprintf("%svar.%s", prefix, variableName)
		if _, ok := rc[variableName]; !ok {
			rc[variableName] = &ConfigOverview{}
		}
		rc[variableName].VariableConfig = variable
	}

	// Loop through output configs
	for outputName, output := range config.Outputs {
		outputName = fmt.Sprintf("%soutput.%s", prefix, outputName)
		if _, ok := rc[outputName]; !ok {
			rc[outputName] = &ConfigOverview{}
		}
		rc[outputName].OutputConfig = output
	}

	// Loop through each resource type and populate graph
	for _, resource := range config.Resources {

		address := fmt.Sprintf("%v%v", prefix, resource.Address)

		if _, ok := rc[address]; !ok {
			rc[address] = &ConfigOverview{}
		}

		rc[address].ResourceConfig = resource
		//rc[address].DependsOn = resource.DependsOn

		if _, ok := rc[parent]; !ok {
			rc[parent] = &ConfigOverview{}
		}
	}

	// Add modules
	for moduleName, m := range config.ModuleCalls {

		mn := fmt.Sprintf("module.%s", moduleName)
		if prefix != "" {
			mn = fmt.Sprintf("%s%s", prefix, mn)
		}

		if _, ok := rc[mn]; !ok {
			rc[mn] = &ConfigOverview{}
		}

		rc[mn].ModuleConfig = m

		if _, ok := rc[parent]; !ok {
			rc[parent] = &ConfigOverview{}
		}

		r.PopulateConfigs(mn, rso, m.Module)
	}
}

func (r *rover) PopulateModuleState(rso *ResourcesOverview, module *tfjson.StateModule, config *tfjson.ConfigModule, prior bool) {
	matchBrackets := regexp.MustCompile(`\[[^\[\]]*\]`)
	childIndex := regexp.MustCompile(`\[[^[\]]*\]$`)

	rs := rso.States

	// Loop through each resource type and populate states
	for _, rst := range module.Resources {
		id := rst.Address
		parent := module.Address
		//fmt.Printf("ID: %v\n", id)
		if rst.AttributeValues != nil {
			configId := matchBrackets.ReplaceAllString(id, "")

			// Add resource to parent
			// Create resource if doesn't exist
			if _, ok := rs[id]; !ok {
				rs[id] = &StateOverview{}
				rs[id].ConfigId = configId
			}

			// Check if resource has parent
			// part of, resource w/ count or for_each
			if childIndex.MatchString(id) {
				parent = childIndex.ReplaceAllString(id, "")
				// If resource has parent, create parent if doesn't exist
				if _, ok := rs[parent]; !ok {
					rs[parent] = &StateOverview{}
					rs[parent].Children = make(map[string]*StateOverview)
				}

				rs[module.Address].Children[parent] = rs[parent]
				rs[parent].ConfigId = matchBrackets.ReplaceAllString(parent, "")
			}

			//fmt.Printf("%v - %v\n", id, parent)
			rs[parent].Children[id] = rs[id]

			if prior {
				rs[id].Change.Before = rst.AttributeValues
			} else {
				rs[id].Change.After = rst.AttributeValues
			}

		} else {
			if prior {
				rs[id].Change.Before = rst.AttributeValues
			} else {
				rs[id].Change.After = rst.AttributeValues

			}
		}

	}

	for _, childModule := range module.ChildModules {

		matchBrackets := regexp.MustCompile(`\[[^\[\]]*\]`)

		parent := module.Address
		id := childModule.Address
		configId := matchBrackets.ReplaceAllString(id, "")
		cid := strings.Split(configId, ".")[len(strings.Split(configId, "."))-1]

		if childIndex.MatchString(id) {
			parent = childIndex.ReplaceAllString(id, "")
		}

		//fmt.Printf("'%v' '%v' '%v'\n", parent, id, configId)
		// If module has parent, create parent if doesn't exist
		if _, ok := rs[parent]; !ok {
			rs[parent] = &StateOverview{}
			rs[parent].Children = make(map[string]*StateOverview)
			rs[parent].ConfigId = configId
		}

		if rs[parent].Module == nil {
			rs[parent].Module = module
		}

		if _, ok := rs[id]; !ok {
			rs[id] = &StateOverview{}
			rs[id].Children = make(map[string]*StateOverview)
			rs[id].ConfigId = configId
		}

		rs[id].Module = childModule

		rs[parent].Children[id] = rs[id]

		r.PopulateModuleState(rso, childModule, config.ModuleCalls[cid].Module, prior)
	}

}

// GenerateResourceOverview - Overview of files and their resources
// Groups different resource types together
func (r *rover) GenerateResourceOverview() error {
	log.Println("Generating resource overview...")

	matchBrackets := regexp.MustCompile(`\[[^\[\]]*\]`)
	rso := &ResourcesOverview{}

	rso.Configs = make(map[string]*ConfigOverview)
	rso.States = make(map[string]*StateOverview)

	rc := rso.Configs
	rs := rso.States

	r.PopulateConfigs("", rso, r.Plan.Config.RootModule)

	// Populate prior state
	if r.Plan.PriorState != nil {
		if r.Plan.PriorState.Values != nil {
			if r.Plan.PriorState.Values.RootModule != nil {
				r.PopulateModuleState(rso, r.Plan.PriorState.Values.RootModule, r.Plan.Config.RootModule, true)
			}
		}
	}

	// Populate planned state
	if r.Plan.PlannedValues != nil {
		if r.Plan.PlannedValues.RootModule != nil {
			r.PopulateModuleState(rso, r.Plan.PlannedValues.RootModule, r.Plan.Config.RootModule, false)
		}
	}

	// reIsChild := regexp.MustCompile(`^\w+\.\w+[\.\[]`)
	// reGetParent := regexp.MustCompile(`^\w+\.\w+`)
	//reIsChild := regexp.MustCompile(`^\w+\.[\w-]+[\.\[]`)

	// Loop through output changes
	for outputName, output := range r.Plan.OutputChanges {
		if _, ok := rs[outputName]; !ok {
			rs[outputName] = &StateOverview{}
		}

		// If before/after sensitive, set value to "Sensitive Value"
		if !r.ShowSensitive {
			if output.BeforeSensitive != nil {
				if output.BeforeSensitive.(bool) {
					output.Before = "Sensitive Value"
				}
			}
			if output.AfterSensitive != nil {
				if output.AfterSensitive.(bool) {
					output.After = "Sensitive Value"
				}
			}
		}

		rs[outputName].Change = *output
	}

	// Loop through resource changes
	for _, resource := range r.Plan.ResourceChanges {
		id := resource.Address
		configId := matchBrackets.ReplaceAllString(id, "")
		parent := resource.ModuleAddress

		if resource.Change != nil {

			// If module has parent, create parent if doesn't exist
			if _, ok := rs[parent]; !ok {
				rs[parent] = &StateOverview{}
				rs[parent].Children = make(map[string]*StateOverview)
			}

			// Add resource to parent
			// Create resource if doesn't exist
			if _, ok := rs[id]; !ok {
				rs[id] = &StateOverview{}
				rs[parent].Children[id] = rs[id]
			}
			rs[id].Change = *resource.Change

			// Create resource config if doesn't exist
			if _, ok := rc[configId]; !ok {
				rc[configId] = &ConfigOverview{}
				rc[configId].ResourceConfig = &tfjson.ConfigResource{}

				// Add type and name since it's missing
				// TODO: Find long term fix
				rc[configId].ResourceConfig.Name = resource.Name
				rc[configId].ResourceConfig.Type = resource.Type
			}

		}
	}

	r.RSO = rso

	return nil
}
