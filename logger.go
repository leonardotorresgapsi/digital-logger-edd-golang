package eddlogger

import (
	"encoding/json"
	"strings"

	"github.com/icastillogomar/digital-logger-edd-golang/drivers"
)

type EddLogger struct {
	service string
	driver  drivers.BaseDriver
}

type LogOptions struct {
	TraceID         string
	Level           string
	Action          string
	Context         string
	Method          string
	Path            string
	RequestHeaders  map[string]string
	RequestBody     interface{}
	StatusCode      int
	ResponseHeaders map[string]string
	ResponseBody    interface{}
	MessageInfo     string
	MessageRaw      string
	DurationMs      float64
	Tags            []string
	Service         string
}

func NewLogger(service string) *EddLogger {
	if service == "" {
		service = "digital-edd"
	}
	return &EddLogger{
		service: service,
	}
}

func (l *EddLogger) getDriver() drivers.BaseDriver {
	if l.driver != nil {
		return l.driver
	}
	l.driver = l.createDriver()
	return l.driver
}

func (l *EddLogger) createDriver() drivers.BaseDriver {
	if IsProduction() {
		driver, err := drivers.NewPubSubDriver("", "")
		if err != nil {
			LogError("No se pudo inicializar PubSubDriver: " + err.Error())
			LogWarning("Usando ConsoleDriver como fallback")
			return drivers.NewConsoleDriver()
		}
		return driver
	}

	driver, err := drivers.NewPostgresDriver("")
	if err != nil {
		LogError("No se pudo inicializar PostgresDriver: " + err.Error())
		LogWarning("Usando ConsoleDriver como fallback")
		return drivers.NewConsoleDriver()
	}
	return driver
}

func (l *EddLogger) SetDriver(driver drivers.BaseDriver) {
	l.driver = driver
}

func (l *EddLogger) SendTraceLog(trace *TraceLog) (string, error) {
	data, err := json.Marshal(trace)
	if err != nil {
		return "", err
	}

	var record map[string]interface{}
	if err := json.Unmarshal(data, &record); err != nil {
		return "", err
	}

	return l.getDriver().Send(record)
}

func (l *EddLogger) Log(opts *LogOptions) (string, error) {
	if opts == nil {
		opts = &LogOptions{}
	}

	level := opts.Level
	if level == "" {
		level = string(INFO)
	}

	var request *RequestInfo
	if opts.Method != "" && opts.Path != "" {
		h := sanitizeHeaders(opts.RequestHeaders)
		request = &RequestInfo{
			Method:  opts.Method,
			Path:    opts.Path,
			Headers: h,
			Body:    opts.RequestBody,
		}
	}

	var response *ResponseInfo
	if opts.StatusCode != 0 {
		h := sanitizeHeaders(opts.ResponseHeaders)
		response = &ResponseInfo{
			StatusCode: opts.StatusCode,
			Headers:    h,
			Body:       opts.ResponseBody,
		}
	}

	service := opts.Service
	if service == "" {
		service = l.service
	}

	trace := &TraceLog{
		TraceID:     opts.TraceID,
		Timestamp:   GetMexicoTimeAsUTC(),
		Service:     service,
		Level:       LogLevel(level),
		Action:      opts.Action,
		Context:     opts.Context,
		Request:     request,
		Response:    response,
		MessageInfo: opts.MessageInfo,
		MessageRaw:  opts.MessageRaw,
		DurationMs:  opts.DurationMs,
		Tags:        opts.Tags,
	}

	return l.SendTraceLog(trace)
}

func (l *EddLogger) Close() error {
	if l.driver != nil {
		return l.driver.Close()
	}
	return nil
}

func sanitizeHeaders(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		out[k] = v // allow empty value if you want; BQ requires value, but "" is still a value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
