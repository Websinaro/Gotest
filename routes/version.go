package routes

import (
	"net/http"

	"kabackend/config"
)

func VersionHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"latest_version":        config.LatestVersion,
		"min_supported_version": config.MinSupportedVersion,
		"force_update_message":  config.ForceUpdateMessage,
	})
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Kerala Disaster Management App By Websinaro Is Running",
	})
}
