package cli

import (
	"flag"
	"github.com/sirupsen/logrus"
	"os"
)

func ParseArgs() (string, string, string, string) {
	userID := flag.String("user_id", "self", "VK user ID (default is the current user).")
	logLevel := flag.String("log_level", "INFO", "Set the logging level (DEBUG, INFO, WARNING, ERROR, CRITICAL).")
	logFile := flag.String("log_file", "", "Set the log file path. If not set, logs will be printed to console.")
	query := flag.String("query", "", "Specify a predefined query to run after data collection.")

	flag.Parse()

	validQueries := map[string]bool{
		"total_users":          true,
		"total_groups":         true,
		"top_users":            true,
		"top_groups":           true,
		"mutual_followers":     true,
		"top_subscribers":      true,
		"top_cities":           true,
		"top_mutual_followers": true,
	}

	if *query != "" && !validQueries[*query] {
		logrus.Panicf("query %s not found", *query)
		flag.Usage()
		os.Exit(1)
	}

	return *userID, *logLevel, *logFile, *query
}
