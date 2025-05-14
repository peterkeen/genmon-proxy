package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"net"

	"github.com/iancoleman/strcase"
	"golang.org/x/exp/maps"
)

var (
	addr     = flag.String("addr", ":80", "address to listen on")
	upstream = flag.String("upstream", "", "upstream GenMon server")
	strip    = regexp.MustCompile(`[^a-zA-Z0-9_ ]+`)
)

type outputMap = map[string]string

type result struct {
	res outputMap
	err error
}

func main() {
	flag.Parse()
	if *upstream == "" {
		flag.Usage()
		os.Exit(1)
	}

	ln, err := net.Listen("tcp4", *addr)
	if err != nil {
		log.Fatal(err)
	}

	defer ln.Close()

	var statusCommands = []string{
		"status_num_json",
		"maint_json",
		"outage_json",
		"monitor_json",
	}

	log.Fatal(http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		results := make(chan result, 0)

		for _, command := range statusCommands {
			go func(c string) {
				client := &http.Client{}
				requestAndProcess(client, c, results)
			}(command)
		}

		outputStatus := make(outputMap)

		for i := 0; i < len(statusCommands); i += 1 {
			result := <- results
			if result.err != nil {
				http.Error(w, result.err.Error(), 500)
				return
			} else {
				maps.Copy(outputStatus, result.res)
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

func requestAndProcess(client *http.Client, command string, resultChan chan result) {
	resp, err := client.Get(fmt.Sprintf("%s/cmd/%s", *upstream, command))

	res := result{make(outputMap), nil}

	if err != nil {
		res.err = err
		resultChan <- res
		return
	}

	parsed := make(map[string]any)

	err = json.NewDecoder(resp.Body).Decode(&parsed)
	if err != nil {
		res.err = err
		resultChan <- res
		return
	}

	for key, val := range parsed {
		process(key, val, &res.res)
	}

	resultChan <- res
}

func process(key string, value any, output *outputMap) {
	switch val := value.(type) {
	case []any:
		for _, e := range val {
			process(key, e, output)
		}
	case map[string]any:
		for k, v := range val {
			process(key+" "+k, v, output)
		}
	default:
		outputKey := strip.ReplaceAllString(strcase.ToSnake(key), "")
		(*output)[outputKey] = fmt.Sprintf("%v", val)
	}
}
