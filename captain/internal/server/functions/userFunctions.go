package server

import (
	"math/rand"
	"strconv"

	"github.com/google/uuid"
)

func GenerateproxyString(poolGroup string, countryCode string, isSticky bool) string {

	config := ""
	switch poolGroup {
	case "netnut":
		if countryCode != "" {
			config += "-res-" + countryCode
		}
		if isSticky {
			min := 10000000
			max := 100000000
			randInt := rand.Intn(max-min+1) + min
			config += "-sid-" + strconv.Itoa(randInt)
		}
	case "geonode":
		if countryCode != "" {
			config += "-country-" + countryCode
		}
		if isSticky {
			config += "-session-" + uuid.New().String()[:8] + "-lifetime-60"
		}
	case "iproyal":
		if countryCode != "" {
			config += "-country-" + countryCode
		}
		if isSticky {
			config += "_session-" + uuid.New().String()[:8] + "_lifetime-1h"
		}
	}

	return config
}
