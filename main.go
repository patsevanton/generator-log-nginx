package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/caarlos0/env/v6"
)

type config struct {
	Rate float32 `env:"RATE" envDefault:"1"`

	// Environment variables for specifying exact values
	IPAddresses string `env:"IP_ADDRESSES" envDefault:""`
	HTTPMethods string `env:"HTTP_METHODS" envDefault:""`
	Paths       string `env:"PATHS" envDefault:""`
	StatusCodes string `env:"STATUS_CODES" envDefault:""`
	Hosts       string `env:"HOSTS" envDefault:""`
}

type logEntry struct {
	Timestamp time.Time `json:"ts"`
	HTTP      httpInfo  `json:"http"`
	Nginx     nginxInfo `json:"nginx"`
}

type httpInfo struct {
	RequestID      string  `json:"request_id"`
	Method         string  `json:"method"`
	StatusCode     int     `json:"status_code"`
	URL            string  `json:"url"`
	Host           string  `json:"host"`
	URI            string  `json:"uri"`
	RequestTime    float64 `json:"request_time"`
	UserAgent      string  `json:"user_agent"`
	Protocol       string  `json:"protocol"`
	TraceSessionID string  `json:"trace_session_id"`
	ServerProtocol string  `json:"server_protocol"`
	ContentType    string  `json:"content_type"`
	BytesSent      string  `json:"bytes_sent"`
}

type nginxInfo struct {
	XForwardFor  string `json:"x-forward-for"`
	RemoteAddr   string `json:"remote_addr"`
	HTTPReferrer string `json:"http_referrer"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		panic(err)
	}

	ticker := time.NewTicker(time.Second / time.Duration(cfg.Rate))

	gofakeit.Seed(time.Now().UnixNano())

	// Parse environment variables for specific values
	ipList := parseEnvList(cfg.IPAddresses)
	methodList := parseEnvList(cfg.HTTPMethods)
	pathList := parseEnvList(cfg.Paths)
	statusCodeList := parseEnvIntList(cfg.StatusCodes)
	hostList := parseEnvList(cfg.Hosts)

	// Validate that required environment variables are set
	if len(ipList) == 0 {
		panic("IP_ADDRESSES environment variable must be set with at least one IP address")
	}
	if len(methodList) == 0 {
		panic("HTTP_METHODS environment variable must be set with at least one HTTP method")
	}
	if len(pathList) == 0 {
		panic("PATHS environment variable must be set with at least one path")
	}
	if len(statusCodeList) == 0 {
		panic("STATUS_CODES environment variable must be set with at least one status code")
	}
	if len(hostList) == 0 {
		panic("HOSTS environment variable must be set with at least one host")
	}

	for range ticker.C {
		timeLocal := time.Now()

		// Use only values from environment variables
		ip := ipList[rand.Intn(len(ipList))]
		httpMethod := methodList[rand.Intn(len(methodList))]
		path := pathList[rand.Intn(len(pathList))]
		statusCode := statusCodeList[rand.Intn(len(statusCodeList))]
		host := hostList[rand.Intn(len(hostList))]

		bodyBytesSent := realisticBytesSent(statusCode)
		userAgent := gofakeit.UserAgent()

		// Generate a fake request ID
		requestID := strings.ToLower(gofakeit.UUID())

		logEntry := logEntry{
			Timestamp: timeLocal,
			HTTP: httpInfo{
				RequestID:      requestID,
				Method:         httpMethod,
				StatusCode:     statusCode,
				URL:            fmt.Sprintf("%s/%s", host, strings.TrimPrefix(path, "/")),
				Host:           host,
				URI:            path,
				RequestTime:    gofakeit.Float64Range(0.001, 2.000),
				UserAgent:      userAgent,
				Protocol:       "HTTP/1.1",
				TraceSessionID: "",
				ServerProtocol: "HTTP/1.1",
				ContentType:    "application/json",
				BytesSent:      fmt.Sprintf("%d", bodyBytesSent),
			},
			Nginx: nginxInfo{
				XForwardFor:  ip,
				RemoteAddr:   ip,
				HTTPReferrer: "",
			},
		}

		jsonData, err := json.Marshal(logEntry)
		if err != nil {
			panic(err)
		}

		// Print with the desired prefix format
		fmt.Printf("ingress-nginx-controller controller %s\n", string(jsonData))
	}
}

func parseEnvList(envVar string) []string {
	if envVar == "" {
		return []string{}
	}
	return strings.Split(strings.TrimSpace(envVar), ",")
}

func parseEnvIntList(envVar string) []int {
	if envVar == "" {
		return []int{}
	}

	parts := strings.Split(strings.TrimSpace(envVar), ",")
	var result []int
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if val, err := strconv.Atoi(part); err == nil {
			result = append(result, val)
		}
	}
	return result
}

func realisticBytesSent(statusCode int) int {
	if statusCode >= 400 {
		return rand.Intn(120-30) + 30
	}

	return rand.Intn(3100-800) + 800
}
