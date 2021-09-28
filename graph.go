package main

import (
	"fmt"
	"strings"
	// tfjson "github.com/hashicorp/terraform-json"
)

const (
	VARIABLE_COLOR  string = "#1d7ada"
	OUTPUT_COLOR    string = "#ffc107"
	DATA_COLOR      string = "#dc477d"
	MODULE_COLOR    string = "#8450ba"
	MODULE_BG_COLOR string = "white"
	FNAME_BG_COLOR  string = "white"
	RESOURCE_COLOR  string = "#8450ba"
	LOCAL_COLOR     string = "black"
)

// ModuleGraph TODO
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Node TODO
type Node struct {
	Data    NodeData `json:"data"`
	Classes string   `json:"classes,omitempty"`
}

// NodeData TODO
type NodeData struct {
	ID          string `json:"id"`
	Label       string `json:"label,omitempty"`
	Type        string `json:"type,omitempty"`
	Parent      string `json:"parent,omitempty"`
	ParentColor string `json:"parentColor,omitempty"`
	Change      string `json:"change,omitempty"`
}

// Edge TODO
type Edge struct {
	Data    EdgeData `json:"data"`
	Classes string   `json:"classes,omitempty"`
}

// EdgeData TODO
type EdgeData struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Gradient string `json:"gradient,omitempty"`
}

// GenerateGraph -
func (r *rover) GenerateGraph() error {
	n := r.GenerateNodes()
	e := r.GenerateEdges()

	r.Graph = Graph{
		Nodes: n,
		Edges: e,
	}

	return nil
}

// GenerateNodes -
func (r *rover) GenerateNodes() []Node {
	nodeMap := make(map[string]Node)
	nmo := []string{}

	moduleMap := make(map[string]string)

	basePath := strings.ReplaceAll(r.Map.Path, "./", "")

	nmo = append(nmo, basePath)
	nodeMap[basePath] = Node{
		Data: NodeData{
			ID:    basePath,
			Label: basePath,
			Type:  "basename",
		},
		Classes: "basename",
	}

	for file := range r.Map.Files {
		// remove suffix and file path
		fname := strings.ReplaceAll(file, fmt.Sprintf("%s/", basePath), "")

		nmo = append(nmo, fname)
		nodeMap[fname] = Node{
			Data: NodeData{
				ID:     fname,
				Label:  fname,
				Type:   "fname",
				Parent: basePath,
			},
			Classes: "fname",
		}

		for id, rdata := range r.Map.Files[file] {
			rid := strings.Split(id, ".")

			rtype := getPrimitiveType(rid[0])

			switch rtype {
			case "module":
				moduleMap[id] = fname

				for id, crdata := range rdata.Children {
					if string(crdata.ChangeAction) != "" {
						nodeMap[id] = Node{
							Data: NodeData{
								ID:     id,
								Change: string(crdata.ChangeAction),
							},
						}
					}
				}

				// You don't want to parse module because it will
				// parse the module configuration and display everything
				// even unused resources/data sources which may be confusing

				// nm, tempNmo := parseModule(rid[0], fname, id, rdata)

				// for _, i := range tempNmo {
				// 	nmo = append(nmo, i)
				// 	nodeMap[i] = nm[i]
				// }
			case "var":
				nm := parseVariable(id, fname, id, rdata)

				for k, v := range nm {
					nmo = append(nmo, k)
					nodeMap[k] = v
				}
			case "output":
				nm := parseOutput(fname, id)

				for k, v := range nm {
					nmo = append(nmo, k)
					nodeMap[k] = v
				}
			case "data":
				nm, tempNmo := parseData(rid[1], rid[1], fname, id, rdata)

				for _, i := range tempNmo {
					nmo = append(nmo, i)
					nodeMap[i] = nm[i]
				}
			default:
				nm, tempNmo := parseResource(rid[0], rid[0], fname, id, rdata)

				for _, i := range tempNmo {
					nmo = append(nmo, i)
					nodeMap[i] = nm[i]
				}
			}
		}

	}

	// Go through all module calls, add module resources
	planned := r.Plan.PlannedValues.RootModule
	for _, module := range planned.ChildModules {
		fname := moduleMap[module.Address]

		// Append resource name
		nmo = append(nmo, module.Address)
		nodeMap[module.Address] = Node{
			Data: NodeData{
				ID:     module.Address,
				Label:  strings.TrimPrefix(module.Address, "module."),
				Type:   "module",
				Parent: fname,
			},
			Classes: "module",
		}

		for _, mr := range module.Resources {
			resourceNameSuffix := "name"

			mid := fmt.Sprintf("%s.%s", module.Address, mr.Type)

			// Append resource type
			nmo = append(nmo, mid)
			nodeMap[mid] = Node{
				Data: NodeData{
					ID:          mid,
					Label:       mr.Type,
					Type:        "resource",
					Parent:      module.Address,
					ParentColor: getResourceColor(module.Address),
				},
				Classes: "resource-type",
			}

			mrChange := string(ActionNoop)
			if _, ok := nodeMap[mr.Address]; ok {
				mrChange = string(nodeMap[mr.Address].Data.Change)
			}

			// Append resource name
			nmo = append(nmo, mr.Address)
			nodeMap[mr.Address] = Node{
				Data: NodeData{
					ID:          mr.Address,
					Label:       mr.Name,
					Type:        getPrimitiveType(mr.Type),
					Parent:      mid,
					ParentColor: getResourceColor(module.Address),
					Change:      mrChange,
				},
				Classes: fmt.Sprintf("resource-%s %s", resourceNameSuffix, mrChange),
			}
		}
	}

	// Get module outputs
	config := r.Plan.Config.RootModule
	// for mName, mValue := range config.ModuleCalls {
	// 	if mValue.Module != nil {
	// 		mid := fmt.Sprintf("module.%s", mName)
	// 		for oName, _ := range mValue.Module.Outputs {
	// 			oid := fmt.Sprintf("module.%s.%s", mName, oName)

	// 			nm := parseOutput(oid, mid, oid)

	// 			for k, v := range nm {
	// 				nmo = append(nmo, k)
	// 				nodeMap[k] = v
	// 			}
	// 		}
	// 	}
	// }

	// Check for locals
	for _, v := range config.Outputs {
		if v.Expression != nil {
			for _, dependsOnR := range v.Expression.References {
				if strings.HasPrefix(dependsOnR, "local.") {
					// Append local variable
					nmo = append(nmo, dependsOnR)
					nodeMap[dependsOnR] = Node{
						Data: NodeData{
							ID:     dependsOnR,
							Label:  strings.TrimPrefix(dependsOnR, "local."),
							Type:   "locals",
							Parent: basePath,
						},
						Classes: "locals",
					}
				}
			}
		}
	}
	for _, r := range config.Resources {
		// fmt.Printf("%+v - %+v\n", oName, oValue)
		for _, reValues := range r.Expressions {
			for _, dependsOnR := range reValues.References {
				if strings.HasPrefix(dependsOnR, "local.") {
					// Append local variable
					nmo = append(nmo, dependsOnR)
					nodeMap[dependsOnR] = Node{
						Data: NodeData{
							ID:     dependsOnR,
							Label:  strings.TrimPrefix(dependsOnR, "local."),
							Type:   "locals",
							Parent: basePath,
						},
						Classes: "locals",
					}
				}
			}
		}
	}

	nodes := make([]Node, 0, len(nodeMap))
	exists := make(map[string]bool)

	for _, i := range nmo {
		if _, ok := exists[i]; !ok {
			nodes = append(nodes, nodeMap[i])
			exists[i] = true
		}
	}

	return nodes
}

// GenerateEdges -
func (r *rover) GenerateEdges() []Edge {
	edgeMap := make(map[string]Edge)
	emo := []string{}

	config := r.Plan.Config.RootModule

	// Loop through outputs
	for oName, oValue := range config.Outputs {
		// fmt.Printf("%+v - %+v\n", oName, oValue)
		if oValue.Expression != nil {
			oid := fmt.Sprintf("output.%s", oName)
			for _, dependsOnR := range oValue.Expression.References {
				// ignore each.
				if !strings.HasPrefix(dependsOnR, "each.") {
					if strings.HasPrefix(dependsOnR, "module.") {
						id := strings.Split(dependsOnR, ".")
						dependsOnR = fmt.Sprintf("%s.%s", id[0], id[1])
					}
					id := fmt.Sprintf("%s->%s", oid, dependsOnR)

					targetType := RESOURCE_COLOR

					if strings.HasPrefix(dependsOnR, "output.") {
						targetType = OUTPUT_COLOR
					} else if strings.HasPrefix(dependsOnR, "var.") {
						targetType = VARIABLE_COLOR
					} else if strings.HasPrefix(dependsOnR, "module.") {
						targetType = MODULE_COLOR
					} else if strings.HasPrefix(dependsOnR, "data.") {
						targetType = DATA_COLOR
					} else if strings.HasPrefix(dependsOnR, "local.") {
						targetType = LOCAL_COLOR
					}

					emo = append(emo, id)
					edgeMap[id] = Edge{
						Data: EdgeData{
							ID:       id,
							Source:   oid,
							Target:   dependsOnR,
							Gradient: fmt.Sprintf("%s %s", OUTPUT_COLOR, targetType),
						},
						Classes: "edge",
					}
				}
			}
		}
	}

	// Loop through resources
	for _, resource := range config.Resources {
		// fmt.Printf("%+v - %+v\n", oName, oValue)
		for _, reValues := range resource.Expressions {
			for _, dependsOnR := range reValues.References {
				if !strings.HasPrefix(dependsOnR, "each.") {
					if strings.HasPrefix(dependsOnR, "module.") {
						id := strings.Split(dependsOnR, ".")
						dependsOnR = fmt.Sprintf("%s.%s", id[0], id[1])
					}
					sourceType := RESOURCE_COLOR
					targetType := RESOURCE_COLOR

					if strings.HasPrefix(resource.Address, "output.") {
						sourceType = OUTPUT_COLOR
					} else if strings.HasPrefix(resource.Address, "var.") {
						sourceType = VARIABLE_COLOR
					} else if strings.HasPrefix(resource.Address, "module.") {
						sourceType = MODULE_COLOR
					} else if strings.HasPrefix(resource.Address, "data.") {
						sourceType = DATA_COLOR
					}

					if strings.HasPrefix(dependsOnR, "output.") {
						targetType = OUTPUT_COLOR
					} else if strings.HasPrefix(dependsOnR, "var.") {
						targetType = VARIABLE_COLOR
					} else if strings.HasPrefix(dependsOnR, "module.") {
						targetType = MODULE_COLOR
					} else if strings.HasPrefix(dependsOnR, "data.") {
						targetType = DATA_COLOR
					}

					// For Terraform 1.0, resource references point to specific resource attributes
					// Skip if the target is a resource and reference points to an attribute
					if targetType == RESOURCE_COLOR && len(strings.Split(dependsOnR, ".")) != 2 {
						continue
					} else if targetType == DATA_COLOR && len(strings.Split(dependsOnR, ".")) != 3 {
						continue
					}

					id := fmt.Sprintf("%s->%s", resource.Address, dependsOnR)
					emo = append(emo, id)
					edgeMap[id] = Edge{
						Data: EdgeData{
							ID:       id,
							Source:   resource.Address,
							Target:   dependsOnR,
							Gradient: fmt.Sprintf("%s %s", sourceType, targetType),
						},
						Classes: "edge",
					}
				}
			}
		}
	}

	// Loop through modules
	for mid, module := range config.ModuleCalls {
		// fmt.Printf("%+v - %+v\n", oName, oValue)
		for _, mExpressions := range module.Expressions {
			for _, dependsOnR := range mExpressions.References {
				if !strings.HasPrefix(dependsOnR, "each.") {
					if strings.HasPrefix(dependsOnR, "module.") {
						id := strings.Split(dependsOnR, ".")
						dependsOnR = fmt.Sprintf("%s.%s", id[0], id[1])
					}
					sourceType := MODULE_COLOR
					targetType := RESOURCE_COLOR

					if strings.HasPrefix(dependsOnR, "output.") {
						targetType = OUTPUT_COLOR
					} else if strings.HasPrefix(dependsOnR, "var.") {
						targetType = VARIABLE_COLOR
					} else if strings.HasPrefix(dependsOnR, "module.") {
						targetType = MODULE_COLOR
					} else if strings.HasPrefix(dependsOnR, "data.") {
						targetType = DATA_COLOR
					}

					// For Terraform 1.0, resource references point to specific resource attributes
					// Skip if the target is a resource and reference points to an attribute
					if targetType == RESOURCE_COLOR && len(strings.Split(dependsOnR, ".")) != 2 {
						continue
					} else if targetType == DATA_COLOR && len(strings.Split(dependsOnR, ".")) != 3 {
						continue
					}

					id := fmt.Sprintf("%s->%s", fmt.Sprintf("module.%s", mid), dependsOnR)
					emo = append(emo, id)
					edgeMap[id] = Edge{
						Data: EdgeData{
							ID:       id,
							Source:   fmt.Sprintf("module.%s", mid),
							Target:   dependsOnR,
							Gradient: fmt.Sprintf("%s %s", sourceType, targetType),
						},
						Classes: "edge",
					}
				}
			}
		}
	}

	edges := make([]Edge, 0, len(edgeMap))
	exists := make(map[string]bool)

	for _, i := range emo {
		if _, ok := exists[i]; !ok {
			edges = append(edges, edgeMap[i])
			exists[i] = true
		}
	}

	return edges
}

func getResourceColor(resourceID string) string {
	rID := strings.Split(resourceID, ".")
	switch rID[0] {
	case "module":
		return MODULE_BG_COLOR
	case "data":
		return DATA_COLOR
	case "output":
		return OUTPUT_COLOR
	case "var":
		return VARIABLE_COLOR
	}
	// return RESOURCE_COLOR
	return FNAME_BG_COLOR
}

func getPrimitiveType(resourceType string) string {
	switch resourceType {
	case
		"module",
		"data",
		"output",
		"var",
		"local":
		return resourceType
	}
	return "resource"
}

func getResourceClass(resourceType string) string {
	switch resourceType {
	case
		"data",
		"output",
		"var",
		"local":
		return resourceType
	}
	return "resource"
}

func parseResource(rid string, rtype string, parentID string, id string, rdata *Resource) (map[string]Node, []string) {
	nodeMap := make(map[string]Node)
	nmo := []string{}

	resourceNameSuffix := "name"
	if rdata.Children != nil {
		resourceNameSuffix = "parent"
	}

	// Append resource type
	nmo = append(nmo, rid)
	nodeMap[rid] = Node{
		Data: NodeData{
			ID:          rid,
			Label:       rtype,
			Type:        "data",
			Parent:      parentID,
			ParentColor: getResourceColor(parentID),
		},
		Classes: "resource-type",
	}

	// Append resource name
	nmo = append(nmo, id)
	nodeMap[id] = Node{
		Data: NodeData{
			ID:          id,
			Label:       rdata.Name,
			Type:        string(rdata.Type),
			Parent:      rid,
			ParentColor: getResourceColor(parentID),
			Change:      string(rdata.ChangeAction),
		},
		Classes: fmt.Sprintf("resource-%s %s", resourceNameSuffix, string(rdata.ChangeAction)),
	}

	for cid, crdata := range rdata.Children {
		nmo = append(nmo, cid)
		nodeMap[cid] = Node{
			Data: NodeData{
				ID:          cid,
				Label:       crdata.Name,
				Type:        string(crdata.Type),
				Parent:      id,
				ParentColor: getResourceColor(parentID),
				Change:      string(crdata.ChangeAction),
			},
			Classes: fmt.Sprintf("resource-name %s", string(crdata.ChangeAction)),
		}
	}

	return nodeMap, nmo
}

func parseData(rid string, rtype string, parentID string, id string, rdata *Resource) (map[string]Node, []string) {
	nodeMap := make(map[string]Node)
	nmo := []string{}

	resourceNameSuffix := "name"
	if rdata.Children != nil {
		resourceNameSuffix = "parent"
	}

	// Append resource type
	nmo = append(nmo, rid)
	nodeMap[rid] = Node{
		Data: NodeData{
			ID:          rid,
			Label:       rtype,
			Type:        "data",
			Parent:      parentID,
			ParentColor: getResourceColor(parentID),
		},
		Classes: "data-type",
	}

	// Append resource name
	nmo = append(nmo, id)
	nodeMap[id] = Node{
		Data: NodeData{
			ID:          id,
			Label:       rdata.Name,
			Type:        "data",
			Parent:      rid,
			ParentColor: getResourceColor(parentID),
			Change:      string(rdata.ChangeAction),
		},
		Classes: fmt.Sprintf("data-%s %s", resourceNameSuffix, string(rdata.ChangeAction)),
	}

	for cid, crdata := range rdata.Children {
		nmo = append(nmo, cid)
		nodeMap[cid] = Node{
			Data: NodeData{
				ID:          cid,
				Label:       crdata.Name,
				Type:        "data",
				Parent:      id,
				ParentColor: getResourceColor(parentID),
			},
			Classes: fmt.Sprintf("data-name %s", string(rdata.ChangeAction)),
		}
	}

	return nodeMap, nmo
}

func parseVariable(rid string, parentID string, id string, rdata *Resource) map[string]Node {
	nodeMap := make(map[string]Node)

	// Append resource type
	nodeMap[rid] = Node{
		Data: NodeData{
			ID:     id,
			Label:  strings.TrimPrefix(id, "var."),
			Type:   "variable",
			Parent: parentID,
		},
		Classes: "variable",
	}

	return nodeMap
}

func parseOutput(parentID string, id string) map[string]Node {
	nodeMap := make(map[string]Node)

	label := strings.TrimPrefix(id, fmt.Sprintf("%s.", parentID))
	label = strings.TrimPrefix(label, "output.")

	// Append resource type
	nodeMap[id] = Node{
		Data: NodeData{
			ID:     id,
			Label:  label,
			Type:   "output",
			Parent: parentID,
		},
		Classes: "output",
	}

	return nodeMap
}

func parseModule(rtype string, basePath string, mID string, rdata *Resource) (map[string]Node, []string) {
	nodeMap := make(map[string]Node)
	nmo := []string{}

	// Append resource name
	nmo = append(nmo, mID)
	nodeMap[mID] = Node{
		Data: NodeData{
			ID:     mID,
			Label:  rdata.Name,
			Type:   "module",
			Parent: basePath,
		},
		Classes: "module",
	}

	for id, crdata := range rdata.Children {
		cID := strings.TrimRight(id, fmt.Sprintf(".%s", crdata.Name))

		nm, tempNmo := parseResource(cID, crdata.ResourceType, mID, id, crdata)

		for _, i := range tempNmo {
			nmo = append(nmo, i)
			nodeMap[i] = nm[i]
		}
	}

	// fmt.Printf("%+v\n", nodeMap)

	return nodeMap, nmo
}
