package server

import (
	"math/rand"
	"strconv"

	"github.com/google/uuid"
)

func GenerateproxyString(poolGroup string, countryCode string, isSticky bool, city string, state string, sessionDuration *int) string {
	config := ""
	if countryCode != "" {
		config += "-country-" + countryCode
	}
	if state != "" {
		config += "-state-" + state
	}
	if city != "" {
		config += "-city-" + city
	}
	switch poolGroup {
	case "netnut":
		if isSticky {
			min := 10000000
			max := 100000000
			randInt := rand.Intn(max-min+1) + min
			config += "-session-" + strconv.Itoa(randInt)
		}

	case "geonode":
		if isSticky {
			config += "-session-" + uuid.New().String()[:8]
		}
	case "iproyal":
		if isSticky {
			config += "-session-" + uuid.New().String()[:8]
		}
	}
	if sessionDuration != nil && *sessionDuration != 0 {
		config += "-lifetime-" + strconv.Itoa(*sessionDuration)
	} else {
		config += "-lifetime-" + "60"
	}
	return config
}
