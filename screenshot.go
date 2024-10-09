package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/browser"
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
	var downloadGUID string

	downloadComplete := make(chan bool)
	chromedp.ListenTarget(ctx, func(v interface{}) {
		if ev, ok := v.(*browser.EventDownloadProgress); ok {
			if ev.State == browser.DownloadProgressStateCompleted {
				downloadGUID = ev.GUID
				close(downloadComplete)
			}
		}
	})

	if err := chromedp.Run(ctx, chromedp.Tasks{
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath(os.TempDir()).
			WithEventsEnabled(true),

		chromedp.Navigate(url),
		// wait for graph to be visible
		chromedp.WaitVisible(`#cytoscape-div`),
		// find and click "Save Graph" button
		chromedp.Click(`#saveGraph`, chromedp.NodeVisible),
	}); err != nil && !strings.Contains(err.Error(), "net::ERR_ABORTED") {
		// Note: Ignoring the net::ERR_ABORTED page error is essential here since downloads
		// will cause this error to be emitted, although the download will still succeed.
		log.Fatal(err)
	}
	<-downloadComplete

	e := moveFile(fmt.Sprintf("%v/%v", os.TempDir(), downloadGUID), "./rover.svg")
	if e != nil {
		log.Fatal(e)
	}

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
		return fmt.Errorf("couldn't open source file: %s", err)
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return fmt.Errorf("couldn't open dest file: %s", err)
	}
	defer outputFile.Close()
	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		return fmt.Errorf("writing to output file failed: %s", err)
	}
	// The copy was successful, so now delete the original file
	err = os.Remove(sourcePath)
	if err != nil {
		return fmt.Errorf("failed removing original file: %s", err)
	}
	return nil
}
