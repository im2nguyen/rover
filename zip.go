package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"
)

func (r *rover) generateZip(fe fs.FS, filename string) error {
	newZipFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	// Add frontend to zip file
	feItems, err := fs.ReadDir(fe, ".")
	if err != nil {
		log.Fatalln(err)
	}

	for _, feItem := range feItems {
		if !feItem.IsDir() {
			if err = AddEmbeddedToZip(fe, zipWriter, feItem.Name()); err != nil {
				return err
			}
			continue
		}

		// Iterate through subdirectories (ui/dist/*)
		feSubItems, err := fs.ReadDir(fe, feItem.Name())
		if err != nil {
			return err
		}
		for _, feSubItem := range feSubItems {
			if err = AddEmbeddedToZip(fe, zipWriter, fmt.Sprintf("%s/%s", feItem.Name(), feSubItem.Name())); err != nil {
				return err
			}
		}
	}

	// Add plan, rso, map, graph to zip file
	if err = AddFileToZip(zipWriter, "plan", r.Plan); err != nil {
		return err
	}
	if err = AddFileToZip(zipWriter, "rso", r.RSO); err != nil {
		return err
	}
	if err = AddFileToZip(zipWriter, "map", r.Map); err != nil {
		return err
	}
	if err = AddFileToZip(zipWriter, "graph", r.Graph); err != nil {
		return err
	}

	return nil
}

func AddEmbeddedToZip(fe fs.FS, zipWriter *zip.Writer, filename string) error {
	writer, err := zipWriter.Create(filename)
	if err != nil {
		return err
	}

	var fileToZip fs.File

	// Rename standalone to index.html references from absolute to relative
	if filename == "index.html" {
		curContent, err := fs.ReadFile(fe, filename)
		if err != nil {
			return err
		}

		contents := strings.Split(string(curContent), "</head>")
		// Add js files, workaround since CORS error if you try to do getJSON
		content := fmt.Sprintf("%s%s%s", contents[0], `<script type="text/javascript" language="javascript" src="./map.js"></script>
		<script type="text/javascript" language="javascript" src="./rso.js"></script>
		<script type="text/javascript" language="javascript" src="./graph.js"></script>`, contents[1])
		content = strings.ReplaceAll(content, "=\"/", "=\"./")

		tempFileName, tempFile, err := createTempFile("temp-index.html", []byte(content))
		if err != nil {
			return err
		}
		defer os.Remove(tempFile.Name()) // clean up
		defer tempFile.Close()

		fileToZip, err = os.Open(tempFileName)
		if err != nil {
			return err
		}
		defer fileToZip.Close()
	} else if strings.HasSuffix(filename, ".js") {
		curContent, err := fs.ReadFile(fe, filename)
		if err != nil {
			return err
		}

		rawContent := bytes.ReplaceAll(curContent, []byte("r.p+\""), []byte("\"./"))

		tempFileName, tempFile, err := createTempFile("temp-index.html", rawContent)
		if err != nil {
			return err
		}
		defer os.Remove(tempFile.Name()) // clean up
		defer tempFile.Close()

		fileToZip, err = os.Open(tempFileName)
		if err != nil {
			return err
		}
		defer fileToZip.Close()

	} else {
		fileToZip, err = fe.Open(filename)
		if err != nil {
			return err
		}
		defer fileToZip.Close()
	}

	_, err = io.Copy(writer, fileToZip)
	return err
}

func AddFileToZip(zipWriter *zip.Writer, fileType string, j interface{}) error {
	filename := fmt.Sprintf("%s.js", fileType)

	writer, err := zipWriter.Create(filename)
	if err != nil {
		return err
	}

	b, err := json.Marshal(j)
	if err != nil {
		return fmt.Errorf("error producing JSON: %s", err)
	}

	// add syntax to make json file a js object
	content := fmt.Sprintf("const %s = %s", fileType, string(b))

	tempFileName, tempFile, err := createTempFile(filename, []byte(content))
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name()) // clean up
	defer tempFile.Close()

	fileToZip, err := os.Open(tempFileName)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	_, err = io.Copy(writer, fileToZip)
	return err
}

func createTempFile(filename string, b []byte) (string, *os.File, error) {
	tempFile, err := os.CreateTemp("", filename)
	if err != nil {
		log.Fatal(err)
	}

	_, err = tempFile.Write(b)
	if err != nil {
		return "", tempFile, err
	}

	return tempFile.Name(), tempFile, nil
}
