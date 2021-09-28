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
	Outputs   map[string]*OutputOverview      `json:"output,omitempty"`
	Resources map[string]*ResourceOverview    `json:"resources,omitempty"`
}

type VariableOverview struct {
	Value interface{} `json:"value,omitempty"`
	Sensitive *bool `json:"sensitive,omitempty"`
}

// ResourceOverview is a modified tfjson.Plan
type ResourceOverview struct {
	// ChangeAction tfjson.Actions        `json:change_action`
	PriorState   map[string]interface{}       `json:"prior_state,omitempty"`
	PlannedState map[string]interface{}       `json:"planned_state,omitempty"`
	Change       tfjson.Change                `json:"change,omitempty"`
	Config       tfjson.ConfigResource        `json:"config,omitempty"`
	ModuleConfig *tfjson.ModuleCall           `json:"module_config,omitempty"`
	DependsOn    []string                     `json:"depends_on,omitempty"`
	Children     map[string]*ResourceOverview `json:"children,omitempty"`
}

// OutputOverview is a modified tfjson.Change with Outputs
type OutputOverview struct {
	// ChangeAction tfjson.Actions        `json:change_action`
	Change *tfjson.Change       `json:"change"`
	Config *tfjson.ConfigOutput `json:"config,omitempty"`
}

// GenerateResourceOverview - Overview of files and their resources
// Groups different resource types together
func (r *rover) GenerateResourceOverview() error {
	log.Println("Generating resource overview...")

	rso := &ResourcesOverview{}

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

	// Loop through outputs
	oo := make(map[string]*OutputOverview)
	// Loop through output configs
	for outputName, output := range r.Plan.Config.RootModule.Outputs {
		if _, ok := oo[outputName]; !ok {
			oo[outputName] = &OutputOverview{}
		}
		oo[outputName].Config = output
	}
	// Loop through output changes
	for outputName, output := range r.Plan.OutputChanges {
		if _, ok := oo[outputName]; !ok {
			oo[outputName] = &OutputOverview{}
		}

		// If before/after sensitive, set value to "Sensitive Value"
		if !r.ShowSensitive {
			if output.BeforeSensitive.(bool) {
				output.Before = "Sensitive Value"
			} 
			if output.AfterSensitive.(bool) {
				output.After = "Sensitive Value"
			} 
		}

		oo[outputName].Change = output
	}

	rso.Outputs = oo

	rs := make(map[string]*ResourceOverview)

	// reIsChild := regexp.MustCompile(`^\w+\.\w+[\.\[]`)
	// reGetParent := regexp.MustCompile(`^\w+\.\w+`)
	reIsChild := regexp.MustCompile(`^\w+\.[\w-]+[\.\[]`)
	reGetParent := regexp.MustCompile(`^\w+\.[\w-]+`)

	// Loop through each resource type and populate graph
	for _, rc := range r.Plan.Config.RootModule.Resources {
		if _, ok := rs[rc.Address]; !ok {
			rs[rc.Address] = &ResourceOverview{}
		}

		rs[rc.Address].Config = *rc
		rs[rc.Address].DependsOn = rc.DependsOn
	}

	// Add modules
	for moduleName, m := range r.Plan.Config.RootModule.ModuleCalls {
		mn := fmt.Sprintf("module.%s", moduleName)

		if _, ok := rs[mn]; !ok {
			rs[mn] = &ResourceOverview{}
		}

		rs[mn].ModuleConfig = m
	}

	// Loop through resource changes
	for _, rc := range r.Plan.ResourceChanges {
		id := rc.Address
		var parent string

		// Check if resource has parent
		// part of module, resource w/ count or for_each
		if reIsChild.MatchString(id) {
			parent = reGetParent.FindString(id)

			// If resource has parent, create parent if doesn't exist
			if _, ok := rs[parent]; !ok {
				rs[parent] = &ResourceOverview{}
			}

			if rs[parent].Children == nil {
				rs[parent].Children = make(map[string]*ResourceOverview)
			}
		}

		if rc.Change != nil {
			// Add resource to parent
			if parent != "" {
				// Create resource if doesn't exist
				if _, ok := rs[parent].Children[id]; !ok {
					rs[parent].Children[id] = &ResourceOverview{}
				}
				rs[parent].Children[id].Change = *rc.Change

				// Add type and name since it's missing
				// TODO: Find long term fix
				rs[parent].Children[id].Config.Name = strings.ReplaceAll(rc.Address, fmt.Sprintf("%s.%s.", parent, rc.Type), "")
				rs[parent].Children[id].Config.Type = rc.Type
			} else {
				rs[rc.Address].Change = *rc.Change
			}
		}
	}

	// Populate prior state
	if r.Plan.PriorState != nil {
		if r.Plan.PriorState.Values != nil {
			if r.Plan.PriorState.Values.RootModule != nil {
				for _, rst := range r.Plan.PriorState.Values.RootModule.Resources {
					id := rst.Address
					var parent string

					// Check if resource has parent
					// part of module, resource w/ count or for_each
					if reIsChild.MatchString(id) {
						parent = reGetParent.FindString(id)

						// If resource has parent, create parent if doesn't exist
						if _, ok := rs[parent]; !ok {
							rs[parent] = &ResourceOverview{}
						}

						if rs[parent].Children == nil {
							rs[parent].Children = make(map[string]*ResourceOverview)
						}
					}

					if rst.AttributeValues != nil {
						// Add resource to parent
						if parent != "" {
							// Create resource if doesn't exist
							if _, ok := rs[parent].Children[id]; !ok {
								rs[parent].Children[id] = &ResourceOverview{}
							}
							rs[parent].Children[id].PriorState = rst.AttributeValues

							// Add type and name since it's missing
							// TODO: Find long term fix
							rs[parent].Children[id].Config.Name = strings.ReplaceAll(rst.Address, fmt.Sprintf("%s.%s.", parent, rst.Type), "")
							rs[parent].Children[id].Config.Type = rst.Type
						} else {
							rs[rst.Address].PriorState = rst.AttributeValues
						}
					}
				}
			}
		}
	}

	// Populate planned state
	if r.Plan.PlannedValues != nil {
		if r.Plan.PlannedValues.RootModule != nil {
			for _, rps := range r.Plan.PlannedValues.RootModule.Resources {
				id := rps.Address
				var parent string

				// Check if resource has parent
				// part of module, resource w/ count or for_each
				if reIsChild.MatchString(id) {
					parent = reGetParent.FindString(id)

					// If resource has parent, create parent if doesn't exist
					if _, ok := rs[parent]; !ok {
						rs[parent] = &ResourceOverview{}
					}

					if rs[parent].Children == nil {
						rs[parent].Children = make(map[string]*ResourceOverview)
					}
				}

				if rps.AttributeValues != nil {
					// Add resource to parent
					if parent != "" {
						// Create resource if doesn't exist
						if _, ok := rs[parent].Children[id]; !ok {
							rs[parent].Children[id] = &ResourceOverview{}
						}
						rs[parent].Children[id].PlannedState = rps.AttributeValues

						// Add type and name since it's missing
						// TODO: Find long term fix
						rs[parent].Children[id].Config.Name = strings.ReplaceAll(rps.Address, fmt.Sprintf("%s.%s.", parent, rps.Type), "")
						rs[parent].Children[id].Config.Type = rps.Type
					} else {
						rs[rps.Address].PlannedState = rps.AttributeValues
					}
				}
			}
		}
	}

	rso.Resources = rs

	r.RSO = rso

	return nil
}
