package api

import (
	"github.com/apache/incubator-devlake/core/errors"
	coreModels "github.com/apache/incubator-devlake/core/models"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/argocd/models"
)

func MakePipelinePlanV200(
	subtaskMetas []plugin.SubTaskMeta,
	connectionId uint64,
	bpScopes []*coreModels.BlueprintScope,
) (coreModels.PipelinePlan, []plugin.Scope, errors.Error) {
	var plan coreModels.PipelinePlan
	var scopes []plugin.Scope
	
	// Simplified - skip connection validation for now
	
	for _, bpScope := range bpScopes {
		var stage []*coreModels.PipelineTask
		var scope plugin.Scope
		
		// Extract subtask names from metadata
		var subtaskNames []string
		for _, meta := range subtaskMetas {
			subtaskNames = append(subtaskNames, meta.Name)
		}
		
		// Create ArgoCD tasks for each scope
		task := &coreModels.PipelineTask{
			Plugin:   "argocd",
			Subtasks: subtaskNames,
			Options: map[string]interface{}{
				"connectionId": connectionId,
			},
		}
		
		// If scope has specific configuration, add it
		if bpScope.ScopeId != "" {
			task.Options["appId"] = bpScope.ScopeId
		}
		
		stage = append(stage, task)
		plan = append(plan, stage)
		
		// Create scope reference
		app := &models.ArgoCDApplication{}
		// Note: In a real implementation, you might want to load this from the database
		// or fetch from ArgoCD API based on bpScope.ScopeId
		scope = app
		scopes = append(scopes, scope)
	}
	
	return plan, scopes, nil
}