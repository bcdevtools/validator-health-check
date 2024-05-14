package utils

import (
	"regexp"
	"strings"
)

//goland:noinspection HttpUrlsUsage
func ReplaceAnySchemeWithHttp(endpoint string) string {
	if !strings.Contains(endpoint, "://") { // not contains scheme
		if strings.HasPrefix(endpoint, "//") {
			endpoint = "http:" + endpoint
		} else {
			endpoint = "http://" + endpoint
		}
	} else if strings.HasPrefix(endpoint, "tcp://") {
		// replace with http scheme
		endpoint = "http" + endpoint[3:]
	} else if strings.HasPrefix(endpoint, "http://") {
		// keep
	} else if strings.HasPrefix(endpoint, "https://") {
		// keep
	} else {
		// replace with http scheme
		endpoint = "http" + endpoint[strings.Index(endpoint, "://"):]
	}
	return endpoint
}

var endsWithPort = regexp.MustCompile(`:\d+$`)

func NormalizeRpcEndpoint(endpoint string) string {
	endpoint = strings.TrimSuffix(endpoint, "/")

	var protocolPart, trimmedProtocolPart string
	spl1 := strings.SplitN(endpoint, "://", 2)
	if len(spl1) == 1 {
		protocolPart = ""
		trimmedProtocolPart = endpoint
	} else {
		protocolPart = spl1[0]
		trimmedProtocolPart = spl1[1]
	}

	var hostPart, subPath string
	spl2 := strings.SplitN(trimmedProtocolPart, "/", 2)
	if len(spl2) == 1 {
		hostPart = trimmedProtocolPart
		subPath = ""
	} else {
		hostPart = spl2[0]
		subPath = spl2[1]
	}

	if !endsWithPort.MatchString(hostPart) {
		if //goland:noinspection HttpUrlsUsage
		protocolPart == "http" || protocolPart == "ws" || protocolPart == "" {
			hostPart += ":80"
		} else if protocolPart == "https" || protocolPart == "wss" {
			hostPart += ":443"
		}
	}

	var newEndpoint string
	if protocolPart != "" {
		newEndpoint += protocolPart + "://"
	}
	newEndpoint += hostPart
	if subPath != "" {
		newEndpoint += "/" + subPath
	}

	return newEndpoint
}
