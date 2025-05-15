//go:build e2e
// +build e2e

package e2e

import (
	"testing"

	"github.com/apache/incubator-devlake/core/tests/e2ehelper"
)

func TestSquadcastIncidentIngestion(t *testing.T) {
	e2ehelper.RunPluginE2ETest(t, &e2ehelper.PluginE2ETestConfig{
		PluginName:    "squadcast",
		Subtasks:      []string{"Collect Squadcast Incidents", "Extract Squadcast Incidents"},
		RawTableNames: []string{"_raw_squadcast_incidents"},
		// Add more config as needed, e.g., snapshot tables, mock data, etc.
	})
}
