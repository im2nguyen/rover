package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

//go:embed ui/dist
var frontend embed.FS

type arrayFlags []string

func (i arrayFlags) String() string {
	var ts []string
	for _, el := range i {
		ts = append(ts, el)
	}
	return strings.Join(ts, ",")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	log.Println("Starting Rover...")

	var tfPath, workingDir, name, zipFileName, planFileName string
	var standalone bool
	var tfVarsFiles, tfVars arrayFlags
	flag.StringVar(&tfPath, "tfPath", "/usr/local/bin/terraform", "Path to Terraform binary")
	flag.StringVar(&workingDir, "workingDir", ".", "Path to Terraform configuration")
	flag.StringVar(&name, "name", "rover", "Configuration name")
	flag.StringVar(&zipFileName, "zipFileName", "rover", "Standalone zip file name")
	flag.BoolVar(&standalone, "standalone", false, "Generate standalone HTML files")
	flag.Var(&tfVarsFiles, "tfVarsFile", "Path to *.tfvars files")
	flag.Var(&tfVars, "tfVar", "Terraform variable (key=value)")
	flag.StringVar(&planFileName, "planFileName", "plan.out", "Plan file name")
	flag.Parse()

	parsedTfVarsFiles := strings.Split(tfVarsFiles.String(), ",")
	parsedTfVars := strings.Split(tfVars.String(), ",")

	// Generate assets
	plan, rso, mapDM, graph := generateAssets(name, workingDir, tfPath, parsedTfVarsFiles, parsedTfVars, planFileName)
	log.Println("Done generating assets.")

	// Save to file (debug)
	// saveJSONToFile(name, "plan", "output", plan)
	// saveJSONToFile(name, "rso", "output", rso)
	// saveJSONToFile(name, "map", "output", mapDM)
	// saveJSONToFile(name, "graph", "output", graph)

	// Embed frontend
	fe, err := fs.Sub(frontend, "ui/dist")
	if err != nil {
		log.Fatalln(err)
	}
	frontendFS := http.FileServer(http.FS(fe))

	if standalone {
		err = generateZip(fe,
			fmt.Sprintf("%s.zip", zipFileName),
			plan, rso, mapDM, graph,
		)
		if err != nil {
			log.Fatalln(err)
		}

		log.Printf("Generated zip file: %s.zip\n", zipFileName)
		return
	}

	err = startServer(frontendFS, plan, rso, mapDM, graph)
	if err != nil {
		log.Fatalf("Could not start server: %s\n", err.Error())
	}
}

func generateAssets(name string, workingDir string, tfPath string, tfVarsFiles []string, tfVars []string, planFileName string) (*tfjson.Plan, *ResourcesOverview, *Map, Graph) {
	
	var plan *tfjson.Plan

	log.Println("Use plan")
	// Get Plan
	plan, err := getPlan(workingDir, tfPath, planFileName)
	if err != nil {
		log.Printf("Unable to get Plan")
		// Generate Plan
		generatedPlan, err := generatePlan(name, workingDir, tfPath, tfVarsFiles, tfVars)
		if err != nil {
			log.Printf(fmt.Sprintf("Unable to parse Plan: %s", err))
			os.Exit(2)
		}
		plan = generatedPlan
	}
	
	// Parse Configuration
	log.Println("Parsing configuration...")
	// Get current directory file
	config, _ := tfconfig.LoadModule(workingDir)
	if config.Diagnostics.HasErrors() {
		os.Exit(1)
	}

	// Generate RSO
	log.Println("Generating resource overview...")
	rso := GenerateResourceOverview(plan)

	// Generate Map
	log.Println("Generating resource map...")
	mapDM := GenerateMap(config, rso)

	// Generate Graph
	log.Println("Generating resource graph...")
	graph := GenerateGraph(plan, mapDM)

	return plan, rso, mapDM, graph
}

func getPlan(workingDir string, tfPath string, planFileName string) (*tfjson.Plan, error) {
	tf, err := tfexec.NewTerraform(workingDir, tfPath)
	if err != nil {
		return nil, err
	}

	log.Println("Initializing Terraform...")
	// err = tf.Init(context.Background(), tfexec.Upgrade(true), tfexec.LockTimeout("60s"))
	err = tf.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		return nil, err
	}

	plan, err := tf.ShowPlanFile(context.Background(), planFileName)

	if err != nil {
		log.Printf(fmt.Sprintf("Unable to show Plan (%s): %s", planFileName, err))
	}

	return plan, err
}

func generatePlan(name string, workingDir string, tfPath string, tfVarsFiles []string, tfVars []string) (*tfjson.Plan, error) {
	tmpDir, err := ioutil.TempDir("", "rover")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	tf, err := tfexec.NewTerraform(workingDir, tfPath)
	if err != nil {
		return nil, err
	}

	log.Println("Initializing Terraform...")
	// err = tf.Init(context.Background(), tfexec.Upgrade(true), tfexec.LockTimeout("60s"))
	err = tf.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		return nil, err
	}

	log.Println("Generating plan...")
	planPath := fmt.Sprintf("%s/%s-%v", tmpDir, "roverplan", time.Now().Unix())

	// Create TF Plan options
	var tfPlanOptions []tfexec.PlanOption
	tfPlanOptions = append(tfPlanOptions, tfexec.Out(planPath))

	// Add *.tfvars files
	for _, tfVarsFile := range tfVarsFiles {
		if tfVarsFile != "" {
			tfPlanOptions = append(tfPlanOptions, tfexec.VarFile(tfVarsFile))
		}
	}

	// Add Terraform variables
	for _, tfVar := range tfVars {
		if tfVar != "" {
			tfPlanOptions = append(tfPlanOptions, tfexec.Var(tfVar))
		}
	}

	_, err = tf.Plan(context.Background(), tfPlanOptions...)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Unable to run Plan: %s", err))
	}

	plan, err := tf.ShowPlanFile(context.Background(), planPath)

	return plan, err
}

func showJSON(g interface{}) {
	j, err := json.Marshal(g)
	if err != nil {
		log.Printf("Error producing JSON: %s\n", err)
		os.Exit(2)
	}
	log.Printf("%+v", string(j))
}

func showModuleJSON(module *tfconfig.Module) {
	j, err := json.MarshalIndent(module, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error producing JSON: %s\n", err)
		os.Exit(2)
	}
	os.Stdout.Write(j)
	os.Stdout.Write([]byte{'\n'})
}

func saveJSONToFile(prefix string, fileType string, path string, j interface{}) string {
	b, err := json.Marshal(j)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error producing JSON: %s\n", err)
		os.Exit(2)
	}

	newpath := filepath.Join(".", fmt.Sprintf("%s/%s", path, prefix))
	err = os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Create(fmt.Sprintf("%s/%s-%s.json", newpath, prefix, fileType))
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	_, err = f.WriteString(string(b))
	if err != nil {
		log.Fatal(err)
	}

	// log.Printf("Saved to %s", fmt.Sprintf("%s/%s-%s.json", newpath, prefix, fileType))

	return fmt.Sprintf("%s/%s-%s.json", newpath, prefix, fileType)
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}
