package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

// Heavily inspired by: https://github.com/chromedp/examples/blob/master/download_file/main.go
func screenshot(s *http.Server) {
	// ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithDebugf(log.Printf))
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// create a timeout as a safety net to prevent any infinite wait loops
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	url := fmt.Sprintf("http://%s", s.Addr)

	// this will be used to capture the file name later
	downloadComplete := make(chan bool)

	var buf []byte
	if err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(url),
		// wait for graph to be visible
		chromedp.WaitVisible(`#cytoscape-div`),
		// find and click "Save Graph" button
		chromedp.Click(`#saveGraph`, chromedp.NodeVisible),
		chromedp.Screenshot("#cytoscape-div", &buf, chromedp.NodeVisible),
	}); err != nil {
		log.Fatal(err)
	}
	if err := ioutil.WriteFile("rover.svg", buf, 0o644); err != nil {
		log.Fatal(err)
	}

	<-downloadComplete

	log.Println("Image generation complete.")

	// Shutdown http server
	s.Shutdown(context.Background())
}

// This function resolves the "invalid cross-device link" error for moving files
// between volumes for Docker.
// https://gist.github.com/var23rav/23ae5d0d4d830aff886c3c970b8f6c6b
func moveFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("Couldn't open source file: %s", err)
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return fmt.Errorf("Couldn't open dest file: %s", err)
	}
	defer outputFile.Close()
	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		return fmt.Errorf("Writing to output file failed: %s", err)
	}
	// The copy was successful, so now delete the original file
	err = os.Remove(sourcePath)
	if err != nil {
		return fmt.Errorf("Failed removing original file: %s", err)
	}
	return nil
}
