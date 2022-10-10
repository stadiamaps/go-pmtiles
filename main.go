package main

import (
	"flag"
	"github.com/protomaps/go-pmtiles/pmtiles"
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
	"log"
	"net/http"
	"os"
	"runtime/pprof"
	"strconv"
	"time"
)

func main() {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	if len(os.Args) < 2 {
		logger.Println("Command is required.")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "show":
		pmtiles.Show(logger, os.Args)
	case "serve":
		serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
		port := serveCmd.String("p", "8080", "port to serve on")
		cors := serveCmd.String("cors", "", "CORS allowed origin value")
		cacheSize := serveCmd.Int("cache", 64, "Cache size in mb")
		serveCmd.Parse(os.Args[2:])
		path := serveCmd.Arg(0)
		if path == "" {
			logger.Println("USAGE: serve  [-p PORT] [-cors VALUE] LOCAL_PATH or https://BUCKET")
			os.Exit(1)
		}
		loop := pmtiles.NewLoop(path, logger, *cacheSize, "*")
		loop.Start()

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			status_code, headers, body := loop.Get(r.Context(), r.URL.Path)
			for k, v := range headers {
				w.Header().Set(k, v)
			}
			w.WriteHeader(status_code)
			w.Write(body)
			logger.Printf("served %s in %s", r.URL.Path, time.Since(start))
		})

		logger.Printf("Serving %s on HTTP port: %s with Access-Control-Allow-Origin: %s\n", path, *port, *cors)
		logger.Fatal(http.ListenAndServe(":"+*port, nil))
	case "subpyramid":
		subpyramidCmd := flag.NewFlagSet("subpyramid", flag.ExitOnError)
		cpuProfile := subpyramidCmd.Bool("profile", false, "profiling output")
		subpyramidCmd.Parse(os.Args[2:])
		path := subpyramidCmd.Arg(0)
		output := subpyramidCmd.Arg(1)

		var err error
		num_args := make([]int, 5)
		for i := 0; i < 5; i++ {
			if num_args[i], err = strconv.Atoi(subpyramidCmd.Arg(i + 2)); err != nil {
				panic(err)
			}
		}

		if *cpuProfile {
			f, err := os.Create("output.profile")
			if err != nil {
				log.Fatal(err)
			}
			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
		}
		bounds := "-180,-90,180,90" // TODO deal with antimeridian, center of tile, etc
		pmtiles.SubpyramidXY(logger, path, output, uint8(num_args[0]), uint32(num_args[1]), uint32(num_args[2]), uint32(num_args[3]), uint32(num_args[4]), bounds)
	case "convert":
		convertCmd := flag.NewFlagSet("convert", flag.ExitOnError)
		convertCmd.Parse(os.Args[2:])
		path := convertCmd.Arg(0)
		output := convertCmd.Arg(1)
		pmtiles.Convert(logger, path, output)
	case "upload":
		pmtiles.Upload(logger, os.Args[2:])
	case "validate":
		// pmtiles.Validate()
	default:
		logger.Println("unrecognized command.")
		flag.PrintDefaults()
		os.Exit(1)
	}

}
