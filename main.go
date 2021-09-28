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

type rover struct {
	Name           string
	WorkingDir     string
	TfPath         string
	TfVarsFiles    []string
	TfVars         []string
	PlanPath       string
	WorkspaceName  string
	TFConfigExists bool
	ShowSensitive  bool
	Config         *tfconfig.Module
	Plan           *tfjson.Plan
	RSO            *ResourcesOverview
	Map            *Map
	Graph          Graph
}

func main() {
	log.Println("Starting Rover...")

	var tfPath, workingDir, name, zipFileName, ipPort, planPath, workspaceName string
	var standalone, tfConfigExists, showSensitive bool
	var tfVarsFiles, tfVars arrayFlags
	flag.StringVar(&tfPath, "tfPath", "/usr/local/bin/terraform", "Path to Terraform binary")
	flag.StringVar(&workingDir, "workingDir", ".", "Path to Terraform configuration")
	flag.StringVar(&name, "name", "rover", "Configuration name")
	flag.StringVar(&zipFileName, "zipFileName", "rover", "Standalone zip file name")
	flag.StringVar(&ipPort, "ipPort", "0.0.0.0:9000", "IP and port for Rover server")
	flag.StringVar(&planPath, "planPath", "", "Plan file path")
	flag.StringVar(&workspaceName, "workspaceName", "", "Workspace name")
	flag.BoolVar(&standalone, "standalone", false, "Generate standalone HTML files")
	flag.BoolVar(&tfConfigExists, "tfConfigExists", true, "Terraform configuration exist - set to false if Terraform configuration unavailable (Terraform Cloud, Terragrunt, auto-generated HCL, CDKTF)")
	flag.BoolVar(&showSensitive, "showSensitive", false, "Display sensitive values")
	flag.Var(&tfVarsFiles, "tfVarsFile", "Path to *.tfvars files")
	flag.Var(&tfVars, "tfVar", "Terraform variable (key=value)")
	flag.Parse()

	parsedTfVarsFiles := strings.Split(tfVarsFiles.String(), ",")
	parsedTfVars := strings.Split(tfVars.String(), ",")

	if planPath != "" {
		path, err := os.Getwd()
		if err != nil {
			log.Fatal(errors.New("Unable to get current working directory"))
		}

		if !strings.HasPrefix(planPath, "/") {
			planPath = filepath.Join(path, planPath)
		}
	}

	r := rover{
		Name:           name,
		WorkingDir:     workingDir,
		TfPath:         tfPath,
		PlanPath:       planPath,
		TFConfigExists: tfConfigExists,
		ShowSensitive:  showSensitive,
		TfVarsFiles:    parsedTfVarsFiles,
		TfVars:         parsedTfVars,
		WorkspaceName:  workspaceName,
	}

	// Generate assets
	err := r.generateAssets()
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Println("Done generating assets.")

	// Save to file (debug)
	// saveJSONToFile(name, "plan", "output", r.Plan)
	// saveJSONToFile(name, "rso", "output", r.Plan)
	// saveJSONToFile(name, "map", "output", r.Map)
	// saveJSONToFile(name, "graph", "output", r.Graph)

	// Embed frontend
	fe, err := fs.Sub(frontend, "ui/dist")
	if err != nil {
		log.Fatalln(err)
	}
	frontendFS := http.FileServer(http.FS(fe))

	if standalone {
		err = r.generateZip(fe, fmt.Sprintf("%s.zip", zipFileName))
		if err != nil {
			log.Fatalln(err)
		}

		log.Printf("Generated zip file: %s.zip\n", zipFileName)
		return
	}

	err = r.startServer(ipPort, frontendFS)
	if err != nil {
		log.Fatalf("Could not start server: %s\n", err.Error())
	}
}

func (r *rover) generateAssets() error {
	// Get Plan
	err := r.getPlan()
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to parse Plan: %s", err))
	}

	if r.TFConfigExists {
		// Parse Configuration
		log.Println("Parsing configuration...")
		// Get current directory file
		r.Config, _ = tfconfig.LoadModule(r.WorkingDir)
		if r.Config.Diagnostics.HasErrors() {
			return errors.New(fmt.Sprintf("Unable to parse configuration: %s", r.Config.Diagnostics.Error()))
		}
	}

	// Generate RSO, Map, Graph
	err = r.GenerateResourceOverview()
	if err != nil {
		return err
	}

	err = r.GenerateMap()
	if err != nil {
		return err
	}

	// Generate Graph
	log.Println("Generating resource graph...")
	err = r.GenerateGraph()
	if err != nil {
		return err
	}

	return nil
}

func (r *rover) getPlan() error {
	tmpDir, err := ioutil.TempDir("", "rover")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	tf, err := tfexec.NewTerraform(r.WorkingDir, r.TfPath)
	if err != nil {
		return err
	}

	// If user provided path to plan file
	if r.PlanPath != "" {
		log.Println("Using provided plan...")
		r.Plan, err = tf.ShowPlanFile(context.Background(), r.PlanPath)
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to read Plan (%s): %s", r.PlanPath, err))
		}

		return nil
	}

	log.Println("Initializing Terraform...")
	// err = tf.Init(context.Background(), tfexec.Upgrade(true), tfexec.LockTimeout("60s"))
	err = tf.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to initialize Terraform Plan: %s", err))
	}

	if r.WorkspaceName != "" {
		log.Printf("Running in %s workspace...", r.WorkspaceName)
		err = tf.WorkspaceSelect(context.Background(), r.WorkspaceName)
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to select workspace (%s): %s", r.WorkspaceName, err))
		}
	}

	log.Println("Generating plan...")
	planPath := fmt.Sprintf("%s/%s-%v", tmpDir, "roverplan", time.Now().Unix())

	// Create TF Plan options
	var tfPlanOptions []tfexec.PlanOption
	tfPlanOptions = append(tfPlanOptions, tfexec.Out(planPath))

	// Add *.tfvars files
	for _, tfVarsFile := range r.TfVarsFiles {
		if tfVarsFile != "" {
			tfPlanOptions = append(tfPlanOptions, tfexec.VarFile(tfVarsFile))
		}
	}

	// Add Terraform variables
	for _, tfVar := range r.TfVars {
		if tfVar != "" {
			tfPlanOptions = append(tfPlanOptions, tfexec.Var(tfVar))
		}
	}

	_, err = tf.Plan(context.Background(), tfPlanOptions...)
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to run Plan: %s", err))
	}

	r.Plan, err = tf.ShowPlanFile(context.Background(), planPath)
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to read Plan: %s", err))
	}

	return nil
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

	return fmt.Sprintf("%s/%s-%s.json", newpath, prefix, fileType)
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}
