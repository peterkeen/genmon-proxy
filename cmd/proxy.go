package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/iancoleman/strcase"
	"golang.org/x/exp/maps"
	"tailscale.com/tsnet"
)

var (
	addr     = flag.String("addr", ":80", "address to listen on")
	hostname = flag.String("hostname", "genmon-proxy", "hostname to listen on")
	upstream = flag.String("upstream", "", "upstream GenMon server")
	strip    = regexp.MustCompile(`[^a-zA-Z0-9_ ]+`)
)

type outputMap = map[string]string

func main() {
	flag.Parse()
	if *upstream == "" {
		flag.Usage()
		os.Exit(1)
	}

	s := new(tsnet.Server)
	s.Hostname = *hostname
	defer s.Close()

	ln, err := s.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}

	defer ln.Close()

	lc, err := s.LocalClient()
	if err != nil {
		log.Fatal(err)
	}

	if *addr == ":443" {
		ln = tls.NewListener(ln, &tls.Config{
			GetCertificate: lc.GetCertificate,
		})
	}

	var statusCommands = []string{
		"status_num_json",
		"maint_json",
		"outage_json",
		"monitor_json",
	}

	log.Fatal(http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errors := make(chan error, 0)
		results := make(chan outputMap, 0)

		for _, command := range statusCommands {
			go func(c string) {
				client := s.HTTPClient()
				requestAndProcess(client, c, results, errors)
			}(command)
		}

		outputStatus := make(outputMap)

		for i := 0; i < len(statusCommands); i += 1 {
			select {
			case err := <-errors:
				http.Error(w, err.Error(), 500)
				return
			case result := <-results:
				maps.Copy(outputStatus, result)
			}
		}

		output, err := json.MarshalIndent(outputStatus, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		fmt.Fprintf(w, "%s", output)
	})))
}

func requestAndProcess(client *http.Client, command string, resultChan chan map[string]string, errorChan chan error) {
	resp, err := client.Get(fmt.Sprintf("%s/cmd/%s", *upstream, command))

	if err != nil {
		errorChan <- err
		return
	}

	parsed := make(map[string]any)

	err = json.NewDecoder(resp.Body).Decode(&parsed)
	if err != nil {
		errorChan <- err
		return
	}

	var outputStatus = make(outputMap)

	for key, val := range parsed {
		err := process(key, val, &outputStatus)
		if err != nil {
			errorChan <- err
		}
	}

	resultChan <- outputStatus
}

func process(key string, value any, output *outputMap) error {
	switch val := value.(type) {
	case []any:
		for _, e := range val {
			err := process(key, e, output)
			if err != nil {
				return err
			}
		}
	case map[string]any:
		for k, v := range val {
			err := process(key+" "+k, v, output)
			if err != nil {
				return err
			}
		}
	default:
		outputKey := strip.ReplaceAllString(strcase.ToSnake(key), "")
		(*output)[outputKey] = fmt.Sprintf("%v", val)
	}

	return nil
}
