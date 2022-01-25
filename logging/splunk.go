package logging

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/jmpsec/osctrl/settings"
	"github.com/jmpsec/osctrl/types"
	"github.com/jmpsec/osctrl/utils"
	"github.com/spf13/viper"
)

// SlunkConfiguration to hold all splunk configuration values
type SlunkConfiguration struct {
	URL     string `json:"url"`
	Token   string `json:"token"`
	Host    string `json:"host"`
	Index   string `json:"index"`
	Queries string `json:"queries"`
	Status  string `json:"status"`
	Results string `json:"results"`
}

// LoggerSplunk will be used to log data using Splunk
type LoggerSplunk struct {
	Configuration SlunkConfiguration
	Headers       map[string]string
	Enabled       bool
}

// CreateLoggerSplunk to initialize the logger
func CreateLoggerSplunk(splunkFile string) (*LoggerSplunk, error) {
	config, err := LoadSplunk(splunkFile)
	if err != nil {
		return nil, err
	}
	l := &LoggerSplunk{
		Configuration: config,
		Headers: map[string]string{
			utils.Authorization:   "Splunk " + config.Token,
			utils.ContentType: utils.JSONApplicationUTF8,
		},
		Enabled: true,
	}
	return l, nil
}

// LoadSplunk - Function to load the Splunk configuration from JSON file
func LoadSplunk(file string) (SlunkConfiguration, error) {
	var _splunkCfg SlunkConfiguration
	log.Printf("Loading %s", file)
	// Load file and read config
	viper.SetConfigFile(file)
	if err := viper.ReadInConfig(); err != nil {
		return _splunkCfg, err
	}
	cfgRaw := viper.Sub(settings.LoggingSplunk)
	if err := cfgRaw.Unmarshal(&_splunkCfg); err != nil {
		return _splunkCfg, err
	}
	// No errors!
	return _splunkCfg, nil
}

const (
	// SplunkMethod Method to send requests
	SplunkMethod = "POST"
	// SplunkContentType Content Type for requests
	SplunkContentType = "application/json"
)

// SplunkMessage to handle log format to be sent to Splunk
type SplunkMessage struct {
	Time       int64       `json:"time"`
	Host       string      `json:"host"`
	Source     string      `json:"source"`
	SourceType string      `json:"sourcetype"`
	Index      string      `json:"index"`
	Event      interface{} `json:"event"`
}

// Settings - Function to prepare settings for the logger
func (logSP *LoggerSplunk) Settings(mgr *settings.Settings) {
	log.Printf("Setting Splunk logging settings\n")
	// Setting link for on-demand queries
	var _v string
	_v = logSP.Configuration.Queries
	if err := mgr.SetString(_v, settings.ServiceAdmin, settings.QueryResultLink, false); err != nil {
		log.Printf("Error setting %s with %s - %v", _v, settings.QueryResultLink, err)
	}
	_v = logSP.Configuration.Status
	// Setting link for status logs
	if err := mgr.SetString(_v, settings.ServiceAdmin, settings.StatusLogsLink, false); err != nil {
		log.Printf("Error setting %s with %s - %v", _v, settings.StatusLogsLink, err)
	}
	_v = logSP.Configuration.Results
	// Setting link for result logs
	if err := mgr.SetString(_v, settings.ServiceAdmin, settings.ResultLogsLink, false); err != nil {
		log.Printf("Error setting %s with %s - %v", _v, settings.ResultLogsLink, err)
	}
}

// Send - Function that sends JSON logs to Splunk HTTP Event Collector
func (logSP *LoggerSplunk) Send(logType string, data []byte, environment, uuid string, debug bool) {
	if debug {
		log.Printf("DebugService: Send %s via splunk", logType)
	}
	// Check if this is result/status or query
	var sourceType string
	var logs []interface{}
	if logType == types.QueryLog {
		sourceType = logType
		// For on-demand queries, just a JSON blob with results and statuses
		var result interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			log.Printf("error parsing data %s %v", string(data), err)
		}
		logs = append(logs, result)
	} else {
		sourceType = logType + ":" + environment
		// For scheduled queries, convert the array in an array of multiple events
		if err := json.Unmarshal(data, &logs); err != nil {
			log.Printf("error parsing log %s %v", string(data), err)
		}
	}
	// Prepare data according to HTTP Event Collector format
	var events []SplunkMessage
	for _, l := range logs {
		jsonEvent, err := json.Marshal(l)
		if err != nil {
			log.Printf("Error parsing data %s", err)
			continue
		}
		eventData := SplunkMessage{
			Time:       time.Now().Unix(),
			Host:       logSP.Configuration.Host,
			Source:     uuid,
			SourceType: sourceType,
			Index:      logSP.Configuration.Index,
			Event:      string(jsonEvent),
		}
		events = append(events, eventData)
	}
	// Serialize data for Splunk
	jsonEvents, err := json.Marshal(events)
	if err != nil {
		log.Printf("Error parsing data %s", err)
	}
	jsonParam := strings.NewReader(string(jsonEvents))
	if debug {
		log.Printf("DebugService: Sending %d bytes to Splunk for %s - %s", len(data), environment, uuid)
	}
	// Send log with a POST to the Splunk URL
	resp, body, err := utils.SendRequest(SplunkMethod, logSP.Configuration.URL, jsonParam, logSP.Headers)
	if err != nil {
		log.Printf("Error sending request %s", err)
	}
	if debug {
		log.Printf("DebugService: HTTP %d %s", resp, body)
	}
}
