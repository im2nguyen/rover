package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
)

// Heavily inspired by: https://github.com/chromedp/examples/blob/master/download_file/main.go
func main() {
	// ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithDebugf(log.Printf))
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// create a timeout as a safety net to prevent any infinite wait loops
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	url := "http://localhost:9000"

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
			WithDownloadPath(".").
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

	e := os.Rename(fmt.Sprintf("./%s", downloadGUID), "./rover.png")
	if e != nil {
		log.Fatal(e)
	}

	log.Println("Complete Download")
}
