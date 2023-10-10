package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"

	"github.com/iancoleman/strcase"
	"golang.org/x/exp/maps"
	"tailscale.com/tsnet"
)

var (
	addr     = flag.String("addr", ":80", "address to listen on")
	hostname = flag.String("hostname", "genmon-proxy", "hostname to listen on")
	upstream = flag.String("upstream", "", "upstream GenMon server")
)

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

	var statusCommands = map[string]string{
		"status_json":  "Status",
		"maint_json":   "Maintenance",
		"outage_json":  "Outage",
		"monitor_json": "Monitor",
	}

	log.Fatal(http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errors := make(chan error, 0)
		results := make(chan map[string]string, 0)

		for command, topLevelKey := range statusCommands {
			go func(c string, k string) {
				client := s.HTTPClient()
				requestAndProcess(client, c, k, results, errors)
			}(command, topLevelKey)
		}

		outputStatus := make(map[string]string)

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

func requestAndProcess(client *http.Client, command string, topLevelKey string, resultChan chan map[string]string, errorChan chan error) {
	resp, err := client.Get(fmt.Sprintf("%s/cmd/%s", *upstream, command))

	if err != nil {
		errorChan <- err
		return
	}

	strip := regexp.MustCompile(`[^a-zA-Z0-9_ ]+`)
	var parsed map[string]any

	err = json.NewDecoder(resp.Body).Decode(&parsed)
	if err != nil {
		errorChan <- err
		return
	}

	var outputStatus = make(map[string]string)

	entries := parsed[topLevelKey].([]any)

	for _, entry := range entries { // array of one-key maps
		for entryKey, entryVal := range entry.(map[string]any) { // map of key => Any([]map[string]any, map[string]any)
			rt := reflect.TypeOf(entryVal)
			fmt.Printf("%s %s\n", entryKey, entryVal)

			if rt.Kind() == reflect.Slice {
				for _, val := range entryVal.([]any) {
					rt = reflect.TypeOf(val)
					if rt.Kind() == reflect.String {
						outputKey := strcase.ToSnake(topLevelKey + " " + entryKey)
						outputStatus[outputKey] = val.(string)
					} else {
						for vk, vv := range val.(map[string]any) {
							outputKey := strip.ReplaceAllString(strcase.ToSnake(topLevelKey+" "+entryKey+" "+vk), "")
							outputStatus[outputKey] = fmt.Sprintf("%v", vv)
						}
					}
				}
			} else if rt.Kind() == reflect.String {
				outputKey := strcase.ToSnake(topLevelKey + " " + entryKey)
				outputStatus[outputKey] = entryVal.(string)
			} else {
				m := entryVal.(map[string]any)
				for valKey, valVal := range m {
					outputKey := strcase.ToSnake(topLevelKey + " " + entryKey + " " + valKey)
					outputStatus[outputKey] = fmt.Sprintf("%v", valVal)
				}
			}
		}
	}

	resultChan <- outputStatus
}
