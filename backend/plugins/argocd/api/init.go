package api

import (
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

// Placeholder types for plugins without scope/scope config
// Implements plugin.ToolLayerScope
// (All methods return zero values)
type NoScope struct{}

func (NoScope) TableName() string          { return "" }
func (NoScope) ScopeId() string            { return "" }
func (NoScope) ScopeName() string          { return "" }
func (NoScope) ScopeFullName() string      { return "" }
func (NoScope) ScopeParams() interface{}   { return nil }
func (NoScope) ScopeConnectionId() uint64  { return 0 }
func (NoScope) ScopeScopeConfigId() uint64 { return 0 }

// Implements plugin.ToolLayerScopeConfig
type NoScopeConfig struct{}

func (NoScopeConfig) TableName() string               { return "" }
func (NoScopeConfig) ScopeConfigId() uint64           { return 0 }
func (NoScopeConfig) ScopeConfigConnectionId() uint64 { return 0 }

// dsHelper provides helpers for connection and scope APIs
var dsHelper *api.DsHelper[ArgoCDConnection, NoScope, NoScopeConfig]

func init() {
	dsHelper = api.NewDataSourceHelper[
		ArgoCDConnection,
		NoScope,
		NoScopeConfig,
	](nil, "", nil, nil, nil, nil)
}
