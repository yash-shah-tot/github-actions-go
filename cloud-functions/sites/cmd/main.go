package main

import (
	// Blank-import the function package so the init() runs
	//"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	_ "github.com/TakeoffTech/site-info-svc/cloud-functions/sites"
	"log"
	"os"
)

func main() {
	os.Setenv("FUNCTION_TARGET", "FUNCTION_TARGET")
	os.Setenv("PROJECT_ID", "PROJECT_ID")
	os.Setenv("OPENCENSUSX_PROJECT_ID", "PROJECT_ID")
	os.Setenv("AUDIT_LOG_TOPIC", "AUDIT_LOG_TOPIC")
	os.Setenv("SITE_MESSAGE_TOPIC", "SITE_MESSAGE_TOPIC")
	// Only for POST AND PATCH Site
	os.Setenv("GOOGLE_MAPS_API_KEY", "GOOGLE_MAPS_API_KEY")

	// Use PORT environment variable, or default to 8080.
	port := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}
	if err := funcframework.Start(port); err != nil {
		log.Fatalf("funcframework.Start: %v", err)
	}
}
