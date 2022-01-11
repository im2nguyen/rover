package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
)

const (
	VARIABLE_COLOR  string = "#1d7ada"
	OUTPUT_COLOR    string = "#ffc107"
	DATA_COLOR      string = "#dc477d"
	MODULE_COLOR    string = "#8450ba"
	MODULE_BG_COLOR string = "white"
	FNAME_BG_COLOR  string = "white"
	RESOURCE_COLOR  string = "lightgray"
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
	ID          string       `json:"id"`
	Label       string       `json:"label,omitempty"`
	Type        ResourceType `json:"type,omitempty"`
	Parent      string       `json:"parent,omitempty"`
	ParentColor string       `json:"parentColor,omitempty"`
	Change      string       `json:"change,omitempty"`
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
	log.Println("Generating resource graph...")

	nodes := r.GenerateNodes()
	edges := r.GenerateEdges()

	// Edge case for terraform.workspace
	for _, e := range edges {
		if strings.Contains(e.Data.ID, "terraform.workspace") {
			nodes = append(nodes, Node{
				Data: NodeData{
					ID:    "terraform.workspace",
					Label: "terraform.workspace",
					Type:  "locals",
					// Parent is equal to basePath
					Parent: strings.ReplaceAll(r.Map.Path, "./", ""),
				},
				Classes: "locals",
			})
			break
		}
	}

	r.Graph = Graph{
		Nodes: nodes,
		Edges: edges,
	}

	return nil
}

func (r *rover) addNodes(base string, parent string, nodeMap map[string]Node, resources map[string]*Resource) []string {

	nmo := []string{}

	for id, re := range resources {

		if re.Type == ResourceTypeResource || re.Type == ResourceTypeData {

			pid := parent

			if nodeMap[parent].Data.Type == ResourceTypeFile {
				pid = strings.TrimSuffix(pid, nodeMap[parent].Data.Label)
				pid = strings.TrimSuffix(pid, ".")
				//qfmt.Printf("%v\n", pid)
			}

			mid := fmt.Sprintf("%v.%v", pid, re.ResourceType)
			mid = strings.TrimPrefix(mid, fmt.Sprintf("%v.", base))
			mid = strings.TrimPrefix(mid, ".")
			mid = strings.TrimSuffix(mid, ".")

			l := strings.Split(mid, ".")
			label := l[len(l)-1]

			midParent := parent
			if midParent == mid {
				midParent = strings.TrimSuffix(midParent, fmt.Sprintf(".%v", label))
			}

			// Append resource type
			nmo = append(nmo, mid)
			nodeMap[mid] = Node{
				Data: NodeData{
					ID:          mid,
					Label:       label,
					Type:        re.Type,
					Parent:      midParent,
					ParentColor: getResourceColor(nodeMap[parent].Data.Type),
				},
				Classes: fmt.Sprintf("%s-type", re.Type),
			}

			mrChange := string(re.ChangeAction)

			// Append resource name
			nmo = append(nmo, id)
			nodeMap[id] = Node{
				Data: NodeData{
					ID:          id,
					Label:       re.Name,
					Type:        re.Type,
					Parent:      mid,
					ParentColor: getResourceColor(nodeMap[parent].Data.Type),
					Change:      mrChange,
				},
				Classes: fmt.Sprintf("%s-name %s", re.Type, mrChange),
			}

			nmo = append(nmo, r.addNodes(base, id, nodeMap, re.Children)...)

		} else if re.Type == ResourceTypeFile {
			fid := id
			if parent != base {
				fid = fmt.Sprintf("%s.%s", parent, fid)
			}
			//fmt.Printf("%v\n", fid)
			nmo = append(nmo, fid)
			nodeMap[fid] = Node{
				Data: NodeData{
					ID:          fid,
					Label:       id,
					Type:        re.Type,
					Parent:      parent,
					ParentColor: getResourceColor(nodeMap[parent].Data.Type),
				},

				Classes: getResourceClass(re.Type),
			}
			nmo = append(nmo, r.addNodes(base, fid, nodeMap, re.Children)...)
		} else {

			pid := parent

			if nodeMap[parent].Data.Type == ResourceTypeFile {
				pid = strings.TrimSuffix(pid, nodeMap[parent].Data.Label)
				pid = strings.TrimSuffix(pid, ".")
			}

			ls := strings.Split(id, ".")
			label := ls[len(ls)-1]

			//fmt.Printf("%v - %v\n", id, re.Type)

			nmo = append(nmo, id)
			nodeMap[id] = Node{
				Data: NodeData{
					ID:          id,
					Label:       label,
					Type:        re.Type,
					Parent:      parent,
					ParentColor: getResourceColor(nodeMap[pid].Data.Type),
				},

				Classes: getResourceClass(re.Type),
			}

			nmo = append(nmo, r.addNodes(base, id, nodeMap, re.Children)...)

		}

	}

	return nmo

}

// GenerateNodes -
func (r *rover) GenerateNodes() []Node {

	nodeMap := make(map[string]Node)
	nmo := []string{}

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

	nmo = append(nmo, r.addNodes(basePath, basePath, nodeMap, r.Map.Root)...)

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

func (r *rover) addEdges(base string, parent string, edgeMap map[string]Edge, resources map[string]*Resource) []string {
	emo := []string{}
	for id, re := range resources {
		matchBrackets := regexp.MustCompile(`\[[^\[\]]*\]`)

		configId := matchBrackets.ReplaceAllString(id, "")

		if _, ok := r.RSO.States[id]; ok {
			configId = r.RSO.States[id].ConfigId
		}

		var expressions map[string]*tfjson.Expression

		if r.RSO.Configs[configId] != nil {
			// If Resource
			if r.RSO.Configs[configId].ResourceConfig != nil {
				expressions = r.RSO.Configs[configId].ResourceConfig.Expressions
				// If Module
			} else if r.RSO.Configs[configId].ModuleConfig != nil {
				expressions = r.RSO.Configs[configId].ModuleConfig.Expressions
				// If Output
			} else if r.RSO.Configs[configId].OutputConfig != nil {
				expressions = make(map[string]*tfjson.Expression)
				expressions["output"] = r.RSO.Configs[configId].OutputConfig.Expression
			}
		}
		// fmt.Printf("%+v - %+v\n", oName, oValue)
		for _, reValues := range expressions {
			for _, dependsOnR := range reValues.References {
				if !strings.HasPrefix(dependsOnR, "each.") {

					/*if strings.HasPrefix(dependsOnR, "module.") {
						id := strings.Split(dependsOnR, ".")
						dependsOnR = fmt.Sprintf("%s.%s", id[0], id[1])
					}*/

					sourceColor := getResourceColor(re.Type)
					targetId := dependsOnR
					if parent != "" {
						targetId = fmt.Sprintf("%s.%s", parent, dependsOnR)
					}

					targetColor := RESOURCE_COLOR

					if strings.Contains(dependsOnR, "output.") {
						targetColor = OUTPUT_COLOR
					} else if strings.Contains(dependsOnR, "var.") {
						targetColor = VARIABLE_COLOR
					} else if strings.HasPrefix(dependsOnR, "module.") {
						targetColor = MODULE_COLOR
					} else if strings.Contains(dependsOnR, "data.") {
						targetColor = DATA_COLOR
					} else if strings.Contains(dependsOnR, "local.") {
						targetColor = LOCAL_COLOR
					}

					// For Terraform 1.0, resource references point to specific resource attributes
					// Skip if the target is a resource and reference points to an attribute
					if targetColor == RESOURCE_COLOR && len(strings.Split(dependsOnR, ".")) != 2 {
						continue
					} else if targetColor == DATA_COLOR && len(strings.Split(dependsOnR, ".")) != 3 {
						continue
					}

					edgeId := fmt.Sprintf("%s->%s", id, targetId)
					emo = append(emo, edgeId)
					edgeMap[edgeId] = Edge{
						Data: EdgeData{
							ID:       edgeId,
							Source:   id,
							Target:   targetId,
							Gradient: fmt.Sprintf("%s %s", sourceColor, targetColor),
						},
						Classes: "edge",
					}
				}
			}
		}

		// Ignore files in edge generation
		if re.Type == ResourceTypeFile {
			emo = append(emo, r.addEdges(base, parent, edgeMap, re.Children)...)
		} else {
			emo = append(emo, r.addEdges(base, id, edgeMap, re.Children)...)
		}
	}

	return emo
}

// GenerateEdges -
func (r *rover) GenerateEdges() []Edge {
	edgeMap := make(map[string]Edge)
	emo := []string{}

	//config := r.Plan.Config.RootModule

	emo = append(emo, r.addEdges("", "", edgeMap, r.Map.Root)...)

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

func getResourceColor(t ResourceType) string {
	switch t {
	case ResourceTypeModule:
		return MODULE_COLOR
	case ResourceTypeData:
		return DATA_COLOR
	case ResourceTypeOutput:
		return OUTPUT_COLOR
	case ResourceTypeVariable:
		return VARIABLE_COLOR
	case ResourceTypeLocal:
		return LOCAL_COLOR
	}
	return RESOURCE_COLOR
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

func getResourceClass(resourceType ResourceType) string {
	switch resourceType {

	case ResourceTypeData:
		return "data-type"
	case ResourceTypeOutput:
		return "output"
	case ResourceTypeVariable:
		return "variable"
	case ResourceTypeFile:
		return "fname"
	case ResourceTypeLocal:
		return "locals"
	case ResourceTypeModule:
		return "module"
	}
	return "resource-type"
}
