package main

import (
	"flag"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/grapeshot/aws_tags_exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// promLogger implements promhttp.Logger
type promLogger struct{}

func (pl promLogger) Println(v ...interface{}) {
	glog.Error(v)
}

type registryCollection struct {
	Registry   *prometheus.Registry
	Collectors map[string]struct{}
	Region     *string
}

func metricsServer(registry prometheus.Gatherer, host string, port int) {
	// Address to listen on for web interface and telemetry
	listenAddress := net.JoinHostPort(host, strconv.Itoa(port))
	glog.Infof("Starting metrics server: %s", listenAddress)

	mux := http.NewServeMux()

	// Add metricsPath
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{ErrorLog: promLogger{}}))

	// Add index
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>AWS Tags Exporter Server</title></head>
             <body>
             <h1>AWS Tags Exporter Metrics</h1>
			 <ul>
             <li><a href='` + "/metrics" + `'>metrics</a></li>
			 </ul>
             </body>
             </html>`))
	})
	glog.Fatal(http.ListenAndServe(listenAddress, mux))
}

// registerCollectors creates and starts informers and initializes and
// registers metrics for collection.
func registerCollectors(r registryCollection) []string {
	activeCollectors := []string{}
	for c := range r.Collectors {
		if f, ok := collector.AvailableCollectors[c]; ok {
			if err := f(r.Registry, r.Region); err != nil {
				glog.Errorf("%s", err)
				continue
			}
			activeCollectors = append(activeCollectors, c)
		} else {
			glog.Warningf("No requested collector: %s", c)
		}
	}

	return activeCollectors
}

func main() {
	// Parse the args (expecting -aws.region)
	Port := flag.Int("web.listen-address", 60020, "Port number to listen on, default is 60020")
	Host := flag.String("web.host", "0.0.0.0", "Port number to listen on, default is 0.0.0.0")
	Region := flag.String("aws.region", "", "AWS region to query.")
	flag.Parse()

	if *Region == "" {
		glog.Fatal("Please supply a region")
	}

	collector := registryCollection{
		Registry:   prometheus.NewRegistry(),
		Collectors: map[string]struct{}{"elb": {}, "rds": {}},
		Region:     Region}

	activeCollectors := registerCollectors(collector)
	glog.Infof("Active collectors: %s", strings.Join(activeCollectors, ","))
	metricsServer(collector.Registry, *Host, *Port)

}
