package fingerprinting

import (
	"os"
	"runtime"
	"strings"
)

// EnvironmentOS return the OS name of the current environment
func EnvironmentOS() string {
	envOS := runtime.GOOS
	switch envOS {
	case "windows", "darwin", "linux":
		return envOS
	default:
		return "unknown"
	}
}

// Environment return the name of the current environment
func Environment() string {
	var env = map[string]string{
		"NETLIFY_IMAGES_CDN_DOMAIN":                 "Netlify",
		"VERCEL":                                    "Vercel",
		"AWS_LAMBDA_FUNCTION_VERSION":               "AWS Lambda",
		"GOOGLE_CLOUD_PROJECT":                      "GCP Compute Instances",
		"WEBSITE_FUNCTIONS_AZUREMONITOR_CATEGORIES": "Azure Cloud Functions",
	}
	for k := range env {
		if _, ok := os.LookupEnv(k); ok {
			return env[k]
		}

		if _, ok := os.LookupEnv("PATH"); ok && strings.Contains(os.Getenv("PATH"), ".heroku") {
			return "Heroku"
		}

		if _, ok := os.LookupEnv("_"); ok && strings.Contains(os.Getenv("_"), "google") {
			return "GCP Cloud Functions"
		}

		if _, ok := os.LookupEnv("WEBSITE_INSTANCE_ID"); ok {
			if _, ok = os.LookupEnv("ORYX_ENV_TYPE"); ok &&
				strings.Contains(os.Getenv("ORYX_ENV_TYPE"), "AppService") {

				return "Azure Compute"
			}
		}
	}

	return "Unknown"
}

// Version return the current version of the Go runtime
func Version() string {
	return runtime.Version()
}
