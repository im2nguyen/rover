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
	Variables map[string]*VariableOverview `json:"variables,omitempty"`
	Outputs   map[string]*OutputOverview   `json:"output,omitempty"`
	Resources map[string]*ResourceOverview `json:"resources,omitempty"`
}

type VariableOverview struct {
	Value     interface{} `json:"value,omitempty"`
	Sensitive *bool       `json:"sensitive,omitempty"`
}

// ResourceOverview is a modified tfjson.Plan
type ResourceOverview struct {
	// ChangeAction tfjson.Actions        `json:change_action`
	PriorState   map[string]interface{}       `json:"prior_state,omitempty"`
	PlannedState map[string]interface{}       `json:"planned_state,omitempty"`
	Change       tfjson.Change                `json:"change,omitempty"`
	Config       tfjson.ConfigResource        `json:"config,omitempty"`
	ModuleConfig *tfjson.ModuleCall           `json:"module_config,omitempty"`
	Module       *tfjson.StateModule          `json:"module,omitempty"`
	DependsOn    []string                     `json:"depends_on,omitempty"`
	Children     map[string]*ResourceOverview `json:"children,omitempty"`
}

// OutputOverview is a modified tfjson.Change with Outputs
type OutputOverview struct {
	// ChangeAction tfjson.Actions        `json:change_action`
	Change *tfjson.Change       `json:"change"`
	Config *tfjson.ConfigOutput `json:"config,omitempty"`
}

func (r *rover) PopulateConfigs(parent string, rso *ResourcesOverview, rs map[string]*ResourceOverview, oo map[string]*OutputOverview, config *tfjson.ConfigModule) {

	prefix := parent
	if prefix != "" {
		prefix = fmt.Sprintf("%s.", prefix)
	}

	// Loop through output configs
	for outputName, output := range config.Outputs {
		outputName = fmt.Sprintf("%soutput.%s", prefix, outputName)
		if _, ok := oo[outputName]; !ok {
			oo[outputName] = &OutputOverview{}
		}
		oo[outputName].Config = output
	}

	// Loop through each resource type and populate graph
	for _, rc := range config.Resources {

		address := fmt.Sprintf("%v%v", prefix, rc.Address)

		if _, ok := rs[address]; !ok {
			rs[address] = &ResourceOverview{}
		}

		rs[address].Config = *rc
		rs[address].DependsOn = rc.DependsOn
		rs[address].Children = make(map[string]*ResourceOverview)

		if _, ok := rs[parent]; !ok {
			rs[parent] = &ResourceOverview{}
			rs[parent].Children = make(map[string]*ResourceOverview)
		}

		rs[parent].Children[address] = rs[address]
	}

	// Add modules
	for moduleName, m := range config.ModuleCalls {

		mn := fmt.Sprintf("module.%s", moduleName)
		if prefix != "" {
			mn = fmt.Sprintf("%s%s", prefix, mn)
		}

		if _, ok := rs[mn]; !ok {
			rs[mn] = &ResourceOverview{}
			rs[mn].Children = make(map[string]*ResourceOverview)
		}

		rs[mn].ModuleConfig = m

		if _, ok := rs[parent]; !ok {
			rs[parent] = &ResourceOverview{}
			rs[parent].Children = make(map[string]*ResourceOverview)
		}

		rs[parent].Children[mn] = rs[mn]

		r.PopulateConfigs(mn, rso, rs, oo, m.Module)
	}
}

func (r *rover) PopulateModuleState(rs map[string]*ResourceOverview, module *tfjson.StateModule, config *tfjson.ConfigModule, prior bool) {
	childIndex := regexp.MustCompile(`\[[^[\]]*\]$`)

	// Loop through each resource type and populate states
	for _, rst := range module.Resources {
		id := rst.Address
		parent := module.Address
		// Check if resource has parent
		// part of module, resource w/ count or for_each
		if childIndex.MatchString(id) {
			parent = childIndex.ReplaceAllString(id, "")
			// If resource has parent, create parent if doesn't exist
			if _, ok := rs[parent]; !ok {
				rs[parent] = &ResourceOverview{}
				rs[parent].Children = make(map[string]*ResourceOverview)
			}

			if rs[parent].Module == nil {
				rs[parent].Module = module
			}
		}

		if rst.AttributeValues != nil {
			// Add resource to parent
			// Create resource if doesn't exist
			if _, ok := rs[id]; !ok {
				rs[id] = &ResourceOverview{}
			}

			//fmt.Printf("%v - %v\n", id, parent)
			rs[parent].Children[id] = rs[id]

			if prior {
				rs[id].PriorState = rst.AttributeValues
			} else {
				rs[id].PlannedState = rst.AttributeValues
			}
			// Add type and name since it's missing
			// TODO: Find long term fix
			rs[id].Config.Name = strings.ReplaceAll(rst.Address, fmt.Sprintf("%s.%s.", parent, rst.Type), "")
			rs[id].Config.Type = rst.Type
		} else {
			if prior {
				rs[id].PriorState = rst.AttributeValues
			} else {
				rs[id].PlannedState = rst.AttributeValues

			}
		}

	}

	for _, childModule := range module.ChildModules {

		matchBrackets := regexp.MustCompile(`\[[^\[\]]*\]`)

		parent := module.Address
		id := childModule.Address
		configId := matchBrackets.ReplaceAllString(id, "")
		configId = strings.Split(configId, ".")[len(strings.Split(configId, "."))-1]

		if childIndex.MatchString(id) {
			parent = childIndex.ReplaceAllString(id, "")
			fmt.Printf("Parent: %v Child: %v\n", parent, id)
		}

		//fmt.Printf("'%v' '%v' '%v'\n", parent, id, configId)
		// If module has parent, create parent if doesn't exist
		if _, ok := rs[parent]; !ok {
			rs[parent] = &ResourceOverview{}
			rs[parent].Children = make(map[string]*ResourceOverview)
		}

		if rs[parent].Module == nil {
			rs[parent].Module = module
		}

		if _, ok := rs[id]; !ok {
			rs[id] = &ResourceOverview{}
			rs[id].Children = make(map[string]*ResourceOverview)
		}

		rs[id].Module = childModule

		rs[parent].Children[id] = rs[id]

		r.PopulateModuleState(rs, childModule, config.ModuleCalls[configId].Module, prior)
	}

}

// GenerateResourceOverview - Overview of files and their resources
// Groups different resource types together
func (r *rover) GenerateResourceOverview() error {
	log.Println("Generating resource overview...")

	rso := &ResourcesOverview{}

	rs := make(map[string]*ResourceOverview)

	// Loop through variables
	vars := make(map[string]*VariableOverview)
	for varName, variable := range r.Plan.Variables {
		if _, ok := vars[varName]; !ok {
			vars[varName] = &VariableOverview{}
		}

		vars[varName].Value = variable.Value
	}

	// If variable is sensitive and show sensitive is off, replace value with "Sensitive Value"
	for varName, variable := range r.Plan.Config.RootModule.Variables {
		vars[varName].Sensitive = &variable.Sensitive

		if !r.ShowSensitive && variable.Sensitive {
			vars[varName].Value = "Sensitive Value"
		}
	}
	rso.Variables = vars

	oo := make(map[string]*OutputOverview)
	r.PopulateConfigs("", rso, rs, oo, r.Plan.Config.RootModule)

	// Populate prior state
	if r.Plan.PriorState != nil {
		if r.Plan.PriorState.Values != nil {
			if r.Plan.PriorState.Values.RootModule != nil {
				r.PopulateModuleState(rs, r.Plan.PriorState.Values.RootModule, r.Plan.Config.RootModule, true)
			}
		}
	}

	// Populate planned state
	if r.Plan.PlannedValues != nil {
		if r.Plan.PlannedValues.RootModule != nil {
			r.PopulateModuleState(rs, r.Plan.PlannedValues.RootModule, r.Plan.Config.RootModule, false)
		}
	}

	// Loop through output changes
	for outputName, output := range r.Plan.OutputChanges {
		if _, ok := oo[outputName]; !ok {
			oo[outputName] = &OutputOverview{}

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

		oo[outputName].Change = output
	}

	// reIsChild := regexp.MustCompile(`^\w+\.\w+[\.\[]`)
	// reGetParent := regexp.MustCompile(`^\w+\.\w+`)
	//reIsChild := regexp.MustCompile(`^\w+\.[\w-]+[\.\[]`)

	// Loop through resource changes
	for _, rc := range r.Plan.ResourceChanges {
		id := rc.Address
		parent := rc.ModuleAddress

		if rc.Change != nil {

			// If module has parent, create parent if doesn't exist
			if _, ok := rs[parent]; !ok {
				rs[parent] = &ResourceOverview{}
				rs[parent].Children = make(map[string]*ResourceOverview)
			}

			// Add resource to parent
			// Create resource if doesn't exist
			if _, ok := rs[id]; !ok {
				rs[id] = &ResourceOverview{}
				rs[parent].Children[id] = rs[id]
			}
			rs[id].Change = *rc.Change

			// Add type and name since it's missing
			// TODO: Find long term fix
			rs[id].Config.Name = strings.ReplaceAll(rc.Address, fmt.Sprintf("%s.%s.", parent, rc.Type), "")
			rs[id].Config.Type = rc.Type

		}
	}

	rso.Resources = rs
	rso.Outputs = oo

	r.RSO = rso

	return nil
}
