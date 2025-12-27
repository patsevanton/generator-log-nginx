package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/caarlos0/env/v6"
)

type config struct {
	Rate             float32 `env:"RATE" envDefault:"1"`
	IPv4Percent      int     `env:"IPV4_PERCENT" envDefault:"100"`
	StatusOkPercent  int     `env:"STATUS_OK_PERCENT" envDefault:"80"`
	PathMinLength    int     `env:"PATH_MIN" envDefault:"1"`
	PathMaxLength    int     `env:"PATH_MAX" envDefault:"5"`
	PercentageGet    int     `env:"GET_PERCENT" envDefault:"60"`
	PercentagePost   int     `env:"POST_PERCENT" envDefault:"30"`
	PercentagePut    int     `env:"PUT_PERCENT" envDefault:"0"`
	PercentagePatch  int     `env:"PATCH_PERCENT" envDefault:"0"`
	PercentageDelete int     `env:"DELETE_PERCENT" envDefault:"0"`
}

type logEntry struct {
	Timestamp time.Time `json:"ts"`
	HTTP      HTTPInfo  `json:"http"`
	Nginx     NginxInfo `json:"nginx"`
}

type HTTPInfo struct {
	RequestID        string  `json:"request_id"`
	Method           string  `json:"method"`
	StatusCode       int     `json:"status_code"`
	URL              string  `json:"url"`
	Host             string  `json:"host"`
	URI              string  `json:"uri"`
	RequestTime      float64 `json:"request_time"`
	UserAgent        string  `json:"user_agent"`
	Protocol         string  `json:"protocol"`
	TraceSessionID   string  `json:"trace_session_id"`
	ServerProtocol   string  `json:"server_protocol"`
	ContentType      string  `json:"content_type"`
	BytesSent        string  `json:"bytes_sent"`
}

type NginxInfo struct {
	XForwardFor  string `json:"x-forward-for"`
	RemoteAddr   string `json:"remote_addr"`
	HTTPReferrer string `json:"http_referrer"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		panic(err)
	}
	checkMinMax(&cfg.PathMinLength, &cfg.PathMaxLength)

	ticker := time.NewTicker(time.Second / time.Duration(cfg.Rate))

	gofakeit.Seed(time.Now().UnixNano())

	for range ticker.C {
		timeLocal := time.Now()

		ip := weightedIPVersion(cfg.IPv4Percent)
		httpMethod := weightedHTTPMethod(cfg.PercentageGet, cfg.PercentagePost, cfg.PercentagePut, cfg.PercentagePatch, cfg.PercentageDelete)
		path := randomPath(cfg.PathMinLength, cfg.PathMaxLength)
		statusCode := weightedStatusCode(cfg.StatusOkPercent)
		bodyBytesSent := realisticBytesSent(statusCode)
		userAgent := gofakeit.UserAgent()

		// Generate a fake request ID
		requestID := strings.ToLower(gofakeit.UUID())
		
		// Generate a fake host
		host := gofakeit.DomainName()
		
		// Generate a fake referrer
		httpReferrer := fmt.Sprintf("http://%s%s", gofakeit.DomainName(), randomPath(cfg.PathMinLength, cfg.PathMaxLength))

		logEntry := logEntry{
			Timestamp: timeLocal,
			HTTP: HTTPInfo{
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
			Nginx: NginxInfo{
				XForwardFor:  ip,
				RemoteAddr:   "",
				HTTPReferrer: httpReferrer,
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

func realisticBytesSent(statusCode int) int {
	if statusCode != 200 {
		return gofakeit.Number(30, 120)
	}

	return gofakeit.Number(800, 3100)
}

func weightedStatusCode(percentageOk int) int {
	roll := gofakeit.Number(0, 100)
	if roll <= percentageOk {
		return 200
	}

	return gofakeit.HTTPStatusCodeSimple()
}

func weightedHTTPMethod(percentageGet, percentagePost, percentagePut, percentagePatch, percentageDelete int) string {
	total := percentageGet + percentagePost + percentagePut + percentagePatch + percentageDelete
	if total > 100 {
		panic("HTTP method percentages add up to more than 100%")
	}

	roll := gofakeit.Number(0, 100)
	if roll <= percentageGet {
		return "GET"
	} else if roll <= percentageGet+percentagePost {
		return "POST"
	} else if roll <= percentageGet+percentagePost+percentagePut {
		return "PUT"
	} else if roll <= percentageGet+percentagePost+percentagePut+percentagePatch {
		return "PATCH"
	} else if roll <= percentageGet+percentagePost+percentagePut+percentagePatch+percentageDelete {
		return "DELETE"
	}

	return gofakeit.HTTPMethod()
}

func weightedIPVersion(percentageIPv4 int) string {
	roll := gofakeit.Number(0, 100)
	if roll <= percentageIPv4 {
		return gofakeit.IPv4Address()
	} else {
		return gofakeit.IPv6Address()
	}
}

func randomPath(min, max int) string {
	var path strings.Builder
	length := gofakeit.Number(min, max)

	path.WriteString("/")

	for i := 0; i < length; i++ {
		if i > 0 {
			path.WriteString(gofakeit.RandomString([]string{"-", "-", "_", "%20", "/", "/", "/"}))
		}
		path.WriteString(gofakeit.BuzzWord())
	}

	path.WriteString(gofakeit.RandomString([]string{".html", ".php", ".htm", ".jpg", ".png", ".gif", ".svg", ".css", ".js"}))

	result := path.String()
	return strings.Replace(result, " ", "%20", -1)
}

func checkMinMax(min, max *int) {
	if *min < 1 {
		*min = 1
	}
	if *max < 1 {
		*max = 1
	}
	if *min > *max {
		*min, *max = *max, *min
	}
}
