package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"kabackend/database"
	"kabackend/middleware"
	"kabackend/routes"
	"kabackend/security"
	"kabackend/services"
)

func mux() *http.ServeMux {
	m := http.NewServeMux()

	// Auth
	m.HandleFunc("POST /register", routes.RegisterHandler)
	m.HandleFunc("POST /login", routes.LoginHandler)
	m.HandleFunc("GET /me", routes.MeHandler)

	// Weather
	m.HandleFunc("GET /weather", routes.WeatherHandler)
	m.HandleFunc("GET /weather/kerala-map", routes.KeralaMapHandler)

	// Version
	m.HandleFunc("GET /app/version", routes.VersionHandler)

	// Safety contacts
	m.HandleFunc("POST /safety-contacts", routes.AddSafetyContactHandler)
	m.HandleFunc("GET /safety-contacts", routes.ListSafetyContactsHandler)
	m.HandleFunc("PUT /safety-contacts/{id}", routes.UpdateSafetyContactHandler)
	m.HandleFunc("DELETE /safety-contacts/{id}", routes.DeleteSafetyContactHandler)

	// SOS
	m.HandleFunc("POST /sos", routes.CreateSosHandler)
	m.HandleFunc("GET /sos/active/incoming", routes.ListIncomingActiveSosHandler)
	m.HandleFunc("GET /sos/mine/active", routes.GetMyActiveSosHandler)
	m.HandleFunc("GET /sos/{id}", routes.GetSosHandler)
	m.HandleFunc("PATCH /sos/{id}/location", routes.UpdateSosLocationHandler)
	m.HandleFunc("POST /sos/{id}/resolve", routes.ResolveSosHandler)

	// SOS live location (WebSocket) - see services/ws_hub.go
	m.HandleFunc("GET /ws/sos/{id}", routes.SosLocationWebSocketHandler)

	// Device token
	m.HandleFunc("POST /device-token", routes.RegisterDeviceTokenHandler)

	// Notifications
	m.HandleFunc("POST /notifications", routes.CreateNotificationHandler)
	m.HandleFunc("GET /notifications", routes.ListNotificationsHandler)
	m.HandleFunc("GET /notifications/{id}", routes.GetNotificationHandler)
	m.HandleFunc("PUT /notifications/{id}", routes.UpdateNotificationHandler)
	m.HandleFunc("DELETE /notifications/{id}", routes.DeleteNotificationHandler)

	// President dashboard
	m.HandleFunc("GET /president/dashboard", routes.PresidentDashboardHandler)

	// Home
	m.HandleFunc("GET /{$}", routes.HomeHandler)

	return m
}

func main() {
	database.Connect()
	database.RunMigrations()

	if err := security.InitCrypto(); err != nil {
		log.Fatalf("failed to initialize AES key: %v", err)
	}

	// Keepalive-ping every open SOS WebSocket so dead connections on flaky
	// 2G/3G links get pruned promptly instead of hanging half-open.
	services.GlobalSosHub.StartKeepalive(20 * time.Second)

	handler := middleware.RecoveryMiddleware(
		middleware.CORSMiddleware(
			middleware.VersionCheckMiddleware(
				middleware.EncryptionMiddleware(mux()),
			),
		),
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("Kerala Disaster Management App By Websinaro Is Running on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}
