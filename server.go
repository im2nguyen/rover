package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	// tfjson "github.com/hashicorp/terraform-json"
)

func (ro *rover) startServer(ipPort string, frontendFS http.Handler) error {

	m := http.NewServeMux()
	s := http.Server{Addr: ipPort, Handler: m}

	m.Handle("/", frontendFS)
	m.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// simple healthcheck
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"alive": true}`)
	})
	m.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		fileType := strings.Replace(r.URL.Path, "/api/", "", 1)

		var j []byte
		var err error

		enableCors(&w)

		switch fileType {
		case "plan":
			j, err = json.Marshal(ro.Plan)
			if err != nil {
				io.WriteString(w, fmt.Sprintf("Error producing plan JSON: %s\n", err))
			}
		case "rso":
			j, err = json.Marshal(ro.RSO)
			if err != nil {
				io.WriteString(w, fmt.Sprintf("Error producing rso JSON: %s\n", err))
			}
		case "map":
			j, err = json.Marshal(ro.Map)
			if err != nil {
				io.WriteString(w, fmt.Sprintf("Error producing map JSON: %s\n", err))
			}
		case "graph":
			j, err = json.Marshal(ro.Graph)
			if err != nil {
				io.WriteString(w, fmt.Sprintf("Error producing graph JSON: %s\n", err))
			}
		default:
			io.WriteString(w, "Please enter a valid file type: plan, rso, map, graph\n")
		}

		w.Header().Set("Content-Type", "application/json")
		io.Copy(w, bytes.NewReader(j))
	})

	log.Printf("Rover is running on %s", ipPort)

	l, err := net.Listen("tcp", ipPort)
	if err != nil {
		log.Fatal(err)
	}

	// The browser can connect now because the listening socket is open.
	if ro.GenImage {
		go screenshot(&s)
	}

	// Start the blocking server loop.
	return s.Serve(l)

}
