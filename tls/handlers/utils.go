package handlers

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jmpsec/osctrl/environments"
	"github.com/jmpsec/osctrl/nodes"
	"github.com/jmpsec/osctrl/types"
	"github.com/segmentio/ksuid"
)

// Helper to generate a random enough node key
func generateNodeKey(uuid string, ts time.Time) string {
	timestamp := strconv.FormatInt(ts.UTC().UnixNano(), 10)
	hasher := sha1.New()
	_, _ = hasher.Write([]byte(uuid + timestamp))
	bs := hasher.Sum(nil)
	return fmt.Sprintf("%x", bs)
}

// Helper to generate a carve session_id using KSUID
// See https://github.com/segmentio/ksuid for more info about KSUIDs
func generateCarveSessionID() string {
	id := ksuid.New()
	return id.String()
}

// Helper to check if the provided secret is valid for this environment
func (h *HandlersTLS) checkValidSecret(enrollSecret string, env environments.TLSEnvironment) bool {
	return (strings.TrimSpace(enrollSecret) == env.Secret)
}

// Helper to check if the provided SecretPath is valid for enrolling in a environment
func (h *HandlersTLS) checkValidEnrollSecretPath(env environments.TLSEnvironment, secretpath string) bool {
	return h.checkValidRemovePath(secretpath, env.EnrollSecretPath)
}

// Helper to check if the provided SecretPath is expired for enrolling in a environment
func (h *HandlersTLS) checkExpiredEnrollSecretPath(env environments.TLSEnvironment) bool {
	return h.checkExpiredPath(env.EnrollExpire)
}

// Helper to check if the provided SecretPath is valid for removing in a environment
func (h *HandlersTLS) checkValidRemoveSecretPath(env environments.TLSEnvironment, secretpath string) bool {
	return h.checkValidRemovePath(secretpath, env.RemoveSecretPath)
}

// Helper to check if the provided SecretPath is expired for removing in a environment
func (h *HandlersTLS) checkExpiredRemoveSecretPath(env environments.TLSEnvironment) bool {
	return h.checkExpiredPath(env.RemoveExpire)
}

// Helper to check if the provided generic SecretPath is valid
func (h *HandlersTLS) checkValidRemovePath(secretpath, envSecret string) bool {
	return (strings.TrimSpace(secretpath) == envSecret)
}

// Helper to check if a provided generic SecretPath is expired
func (h *HandlersTLS) checkExpiredPath(maybeExpired time.Time) bool {
	return (!environments.IsItExpired(maybeExpired))
}

// Helper to convert an enrollment request into a osquery node
func nodeFromEnroll(req types.EnrollRequest, environment, ipaddress, nodekey string, recBytes int) nodes.OsqueryNode {
	// Prepare the enrollment request to be stored as raw JSON
	enrollRaw, err := json.Marshal(req)
	if err != nil {
		log.Printf("error serializing enrollment: %v", err)
		enrollRaw = []byte("")
	}
	// Avoid the error "unsupported Unicode escape sequence" due to \u0000
	enrollRaw = bytes.Replace(enrollRaw, []byte("\\u0000"), []byte(""), -1)
	return nodes.OsqueryNode{
		NodeKey:         nodekey,
		UUID:            strings.ToUpper(req.HostIdentifier),
		Platform:        req.HostDetails.EnrollOSVersion.Platform,
		PlatformVersion: req.HostDetails.EnrollOSVersion.Version,
		OsqueryVersion:  req.HostDetails.EnrollOsqueryInfo.Version,
		Hostname:        req.HostDetails.EnrollSystemInfo.Hostname,
		Localname:       req.HostDetails.EnrollSystemInfo.LocalHostname,
		IPAddress:       ipaddress,
		Username:        "unknown",
		OsqueryUser:     "unknown",
		Environment:     environment,
		CPU:             strings.TrimRight(req.HostDetails.EnrollSystemInfo.CPUBrand, "\x00"),
		Memory:          req.HostDetails.EnrollSystemInfo.PhysicalMemory,
		HardwareSerial:  req.HostDetails.EnrollSystemInfo.HardwareSerial,
		ConfigHash:      req.HostDetails.EnrollOsqueryInfo.ConfigHash,
		BytesReceived:   recBytes,
		RawEnrollment:   enrollRaw,
		LastStatus:      time.Time{},
		LastResult:      time.Time{},
		LastConfig:      time.Time{},
		LastQueryRead:   time.Time{},
		LastQueryWrite:  time.Time{},
	}
}

// Helper to remove duplicates from array of strings
func uniq(duplicated []string) []string {
	keys := make(map[string]bool)
	result := []string{}
	for _, entry := range duplicated {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			result = append(result, entry)
		}
	}
	return result
}
