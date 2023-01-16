package logging

import (
	"github.com/TakeoffTech/go-telemetry/opencensusx"
	"github.com/TakeoffTech/site-info-svc/common"
	"os"
)

// init function will initialise the opencensus telemetry
func init() {
	projectID := os.Getenv(common.EnvOpencensusxProjectID)
	if projectID != "" {
		opencensusx.InitTelemetryWithServiceName(logger, common.ServiceName)
	}
}
