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

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

const VERSION = "0.3.3"

var TRUE = true

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
	Name             string
	WorkingDir       string
	TfPath           string
	TfVarsFiles      []string
	TfVars           []string
	TfBackendConfigs []string
	PlanPath         string
	PlanJSONPath     string
	WorkspaceName    string
	TFCOrgName       string
	TFCWorkspaceName string
	ShowSensitive    bool
	GenImage         bool
	TFCNewRun        bool
	Plan             *tfjson.Plan
	RSO              *ResourcesOverview
	Map              *Map
	Graph            Graph
}

func main() {
	var tfPath, workingDir, name, zipFileName, ipPort, planPath, planJSONPath, workspaceName, tfcOrgName, tfcWorkspaceName string
	var standalone, genImage, showSensitive, getVersion, tfcNewRun bool
	var tfVarsFiles, tfVars, tfBackendConfigs arrayFlags
	flag.StringVar(&tfPath, "tfPath", "/opt/homebrew/bin/terraform", "Path to Terraform binary")
	flag.StringVar(&workingDir, "workingDir", ".", "Path to Terraform configuration")
	flag.StringVar(&name, "name", "rover", "Configuration name")
	flag.StringVar(&zipFileName, "zipFileName", "rover", "Standalone zip file name")
	flag.StringVar(&ipPort, "ipPort", "0.0.0.0:9000", "IP and port for Rover server")
	flag.StringVar(&planPath, "planPath", "", "Plan file path")
	flag.StringVar(&planJSONPath, "planJSONPath", "", "Plan JSON file path")
	flag.StringVar(&workspaceName, "workspaceName", "", "Workspace name")
	flag.StringVar(&tfcOrgName, "tfcOrg", "", "Terraform Cloud Organization name")
	flag.StringVar(&tfcWorkspaceName, "tfcWorkspace", "", "Terraform Cloud Workspace name")
	flag.BoolVar(&standalone, "standalone", false, "Generate standalone HTML files")
	flag.BoolVar(&showSensitive, "showSensitive", false, "Display sensitive values")
	flag.BoolVar(&tfcNewRun, "tfcNewRun", false, "Create new Terraform Cloud run")
	flag.BoolVar(&getVersion, "version", false, "Get current version")
	flag.BoolVar(&genImage, "genImage", false, "Generate graph image")
	flag.Var(&tfVarsFiles, "tfVarsFile", "Path to *.tfvars files")
	flag.Var(&tfVars, "tfVar", "Terraform variable (key=value)")
	flag.Var(&tfBackendConfigs, "tfBackendConfig", "Path to *.tfbackend files")
	flag.Parse()

	if getVersion {
		fmt.Printf("Rover v%s\n", VERSION)
		return
	}

	log.Println("Starting Rover...")

	parsedTfVarsFiles := strings.Split(tfVarsFiles.String(), ",")
	parsedTfVars := strings.Split(tfVars.String(), ",")
	parsedTfBackendConfigs := strings.Split(tfBackendConfigs.String(), ",")

	path, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.New("Unable to get current working directory"))
	}

	if planPath != "" {
		if !strings.HasPrefix(planPath, "/") {
			planPath = filepath.Join(path, planPath)
		}
	}

	if planJSONPath != "" {
		if !strings.HasPrefix(planJSONPath, "/") {
			planJSONPath = filepath.Join(path, planJSONPath)
		}
	}

	r := rover{
		Name:             name,
		WorkingDir:       workingDir,
		TfPath:           tfPath,
		PlanPath:         planPath,
		PlanJSONPath:     planJSONPath,
		ShowSensitive:    showSensitive,
		GenImage:         genImage,
		TfVarsFiles:      parsedTfVarsFiles,
		TfVars:           parsedTfVars,
		TfBackendConfigs: parsedTfBackendConfigs,
		WorkspaceName:    workspaceName,
		TFCOrgName:       tfcOrgName,
		TFCWorkspaceName: tfcWorkspaceName,
		TFCNewRun:        tfcNewRun,
	}

	// Generate assets
	err = r.generateAssets()
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
		// http.Serve() returns error on shutdown
		if genImage {
			log.Println("Server shut down.")
		} else {
			log.Fatalf("Could not start server: %s\n", err.Error())
		}
	}

}

func (r *rover) generateAssets() error {
	// Get Plan
	err := r.getPlan()
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to parse Plan: %s", err))
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

	// If user provided path to plan JSON file
	if r.PlanJSONPath != "" {
		log.Println("Using provided JSON plan...")

		planJsonFile, err := os.Open(r.PlanJSONPath)
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to read Plan (%s): %s", r.PlanJSONPath, err))
		}
		defer planJsonFile.Close()

		planJson, err := ioutil.ReadAll(planJsonFile)
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to read Plan (%s): %s", r.PlanJSONPath, err))
		}

		if err := json.Unmarshal(planJson, &r.Plan); err != nil {
			return errors.New(fmt.Sprintf("Unable to read Plan (%s): %s", r.PlanJSONPath, err))
		}

		return nil
	}

	// If user specified TFC workspace
	if r.TFCWorkspaceName != "" {
		tfcToken := os.Getenv("TFC_TOKEN")

		if tfcToken == "" {
			return errors.New("TFC_TOKEN environment variable not set")
		}

		if r.TFCOrgName == "" {
			return errors.New("Must specify Terraform Cloud organization to retrieve plan from Terraform Cloud")
		}

		config := &tfe.Config{
			Token: tfcToken,
		}

		client, err := tfe.NewClient(config)
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to connect to Terraform Cloud. %s", err))
		}

		// Get TFC Workspace
		ws, err := client.Workspaces.Read(context.Background(), r.TFCOrgName, r.TFCWorkspaceName)
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to list workspace %s in %s organization. %s", r.TFCWorkspaceName, r.TFCOrgName, err))
		}

		// Retrieve all runs from specified TFC workspace
		runs, err := client.Runs.List(context.Background(), ws.ID, &tfe.RunListOptions{})
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to retrieve plan from %s in %s organization. %s", r.TFCWorkspaceName, r.TFCOrgName, err))
		}

		run := runs.Items[0]

		// Get most recent plan item
		planID := runs.Items[0].Plan.ID

		// Run hasn't been applied or discarded, therefore is still "actionable" by user
		runIsActionable := run.StatusTimestamps.AppliedAt.IsZero() && run.StatusTimestamps.DiscardedAt.IsZero()

		if runIsActionable && r.TFCNewRun {
			return errors.New(fmt.Sprintf("Did not create new run. %s in %s in %s is still active", run.ID, r.TFCWorkspaceName, r.TFCOrgName))
		}

		// If latest run is not actionable, rover will create new run
		if r.TFCNewRun {
			// Create new run in specified TFC workspace
			newRun, err := client.Runs.Create(context.Background(), tfe.RunCreateOptions{
				Refresh:   &TRUE,
				Workspace: ws,
			})
			if err != nil {
				return errors.New(fmt.Sprintf("Unable to generate new run from %s in %s organization. %s", r.TFCWorkspaceName, r.TFCOrgName, err))
			}

			run = newRun

			log.Printf("Starting new Terraform Cloud run in %s workspace...", r.TFCWorkspaceName)

			// Wait maximum of 5 mins
			for i := 0; i < 30; i++ {
				run, err := client.Runs.Read(context.Background(), newRun.ID)
				if err != nil {
					return errors.New(fmt.Sprintf("Unable to retrieve run from %s in %s organization. %s", r.TFCWorkspaceName, r.TFCOrgName, err))
				}

				if run.Plan != nil {
					planID = run.Plan.ID
					// Add 20 second timeout so plan JSON becomes available
					time.Sleep(20 * time.Second)
					log.Printf("Run %s to completed!", newRun.ID)
					break
				}

				time.Sleep(10 * time.Second)
				log.Printf("Waiting for run %s to complete (%ds)...", newRun.ID, 10*(i+1))
			}

			if planID == "" {
				return errors.New(fmt.Sprintf("Timeout waiting for plan to complete in %s in %s organization. %s", r.TFCWorkspaceName, r.TFCOrgName, err))
			}
		}

		// Get most recent plan file
		plan, err := client.Plans.Read(context.Background(), planID)
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to retrieve plan from %s in %s organization. %s", r.TFCWorkspaceName, r.TFCOrgName, err))
		}
		planBytes, err := json.Marshal(plan)
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to marshal plan from %s in %s organization. %s", r.TFCWorkspaceName, r.TFCOrgName, err))
		}
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to retrieve plan from %s in %s organization. %s", r.TFCWorkspaceName, r.TFCOrgName, err))
		}
		// If empty plan file
		if string(planBytes) == "" {
			return errors.New(fmt.Sprintf("Empty plan. Check run %s in %s in %s is not pending", run.ID, r.TFCWorkspaceName, r.TFCOrgName))
		}

		if err := json.Unmarshal(planBytes, &r.Plan); err != nil {
			return errors.New(fmt.Sprintf("Unable to parse plan (ID: %s) from %s in %s organization.: %s", planID, r.TFCWorkspaceName, r.TFCOrgName, err))
		}

		return nil
	}

	log.Println("Initializing Terraform...")

	// Create TF Init options
	var tfInitOptions []tfexec.InitOption
	tfInitOptions = append(tfInitOptions, tfexec.Upgrade(true))

	// Add *.tfbackend files
	for _, tfBackendConfig := range r.TfBackendConfigs {
		if tfBackendConfig != "" {
			tfInitOptions = append(tfInitOptions, tfexec.BackendConfig(tfBackendConfig))
		}
	}

	// tfInitOptions = append(tfInitOptions, tfexec.LockTimeout("60s"))

	err = tf.Init(context.Background(), tfInitOptions...)
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
