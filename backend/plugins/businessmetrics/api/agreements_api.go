/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import (
	"net/http"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/businessmetrics/models"
)

// @Summary List working agreements for a project
// @Description Get all working agreements configured for a specific project
// @Tags plugins/businessmetrics
// @Param projectName path string true "Project Name"
// @Success 200 {array} models.WorkingAgreement
// @Router /plugins/businessmetrics/agreements/{projectName} [get]
func ListAgreements(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	projectName := input.Params["projectName"]
	if projectName == "" {
		return nil, errors.BadInput.New("projectName is required")
	}

	db := basicRes.GetDal()
	var agreements []models.WorkingAgreement
	err := db.All(&agreements,
		dal.From(&models.WorkingAgreement{}),
		dal.Where("project_name = ?", projectName),
	)
	if err != nil {
		return nil, errors.Default.Wrap(err, "failed to query agreements")
	}

	// If no agreements, return defaults (without saving)
	if len(agreements) == 0 {
		agreements = models.DefaultWorkingAgreements(projectName)
	}

	return &plugin.ApiResourceOutput{
		Body:   agreements,
		Status: http.StatusOK,
	}, nil
}

// @Summary Create or update a working agreement
// @Description Create a new working agreement or update an existing one
// @Tags plugins/businessmetrics
// @Param projectName path string true "Project Name"
// @Param agreement body models.WorkingAgreement true "Working Agreement"
// @Success 200 {object} models.WorkingAgreement
// @Router /plugins/businessmetrics/agreements/{projectName} [post]
func CreateAgreement(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	projectName := input.Params["projectName"]
	if projectName == "" {
		return nil, errors.BadInput.New("projectName is required")
	}

	var agreement models.WorkingAgreement
	if err := helper.Decode(input.Body, &agreement, nil); err != nil {
		return nil, errors.BadInput.Wrap(err, "invalid request body")
	}

	agreement.ProjectName = projectName
	now := time.Now()
	if agreement.Id == "" {
		agreement.Id = projectName + ":" + string(agreement.AgreementType)
	}
	if agreement.CreatedAt.IsZero() {
		agreement.CreatedAt = now
	}
	agreement.UpdatedAt = now

	db := basicRes.GetDal()
	if err := db.CreateOrUpdate(&agreement); err != nil {
		return nil, errors.Default.Wrap(err, "failed to save agreement")
	}

	return &plugin.ApiResourceOutput{
		Body:   agreement,
		Status: http.StatusOK,
	}, nil
}

// @Summary Get a specific working agreement
// @Description Get a working agreement by project and agreement type
// @Tags plugins/businessmetrics
// @Param projectName path string true "Project Name"
// @Param agreementType path string true "Agreement Type"
// @Success 200 {object} models.WorkingAgreement
// @Router /plugins/businessmetrics/agreements/{projectName}/{agreementType} [get]
func GetAgreement(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	projectName := input.Params["projectName"]
	agreementType := input.Params["agreementType"]

	if projectName == "" || agreementType == "" {
		return nil, errors.BadInput.New("projectName and agreementType are required")
	}

	db := basicRes.GetDal()
	var agreement models.WorkingAgreement
	err := db.First(&agreement,
		dal.From(&models.WorkingAgreement{}),
		dal.Where("project_name = ? AND agreement_type = ?", projectName, agreementType),
	)
	if err != nil {
		return nil, errors.NotFound.Wrap(err, "agreement not found")
	}

	return &plugin.ApiResourceOutput{
		Body:   agreement,
		Status: http.StatusOK,
	}, nil
}

// @Summary Update a working agreement
// @Description Update an existing working agreement
// @Tags plugins/businessmetrics
// @Param projectName path string true "Project Name"
// @Param agreementType path string true "Agreement Type"
// @Param agreement body models.WorkingAgreement true "Working Agreement"
// @Success 200 {object} models.WorkingAgreement
// @Router /plugins/businessmetrics/agreements/{projectName}/{agreementType} [put]
func UpdateAgreement(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	projectName := input.Params["projectName"]
	agreementType := input.Params["agreementType"]

	if projectName == "" || agreementType == "" {
		return nil, errors.BadInput.New("projectName and agreementType are required")
	}

	var agreement models.WorkingAgreement
	if err := helper.Decode(input.Body, &agreement, nil); err != nil {
		return nil, errors.BadInput.Wrap(err, "invalid request body")
	}

	agreement.Id = projectName + ":" + agreementType
	agreement.ProjectName = projectName
	agreement.AgreementType = models.AgreementType(agreementType)
	agreement.UpdatedAt = time.Now()

	db := basicRes.GetDal()
	if err := db.Update(&agreement); err != nil {
		return nil, errors.Default.Wrap(err, "failed to update agreement")
	}

	return &plugin.ApiResourceOutput{
		Body:   agreement,
		Status: http.StatusOK,
	}, nil
}

// @Summary Delete a working agreement
// @Description Delete a working agreement
// @Tags plugins/businessmetrics
// @Param projectName path string true "Project Name"
// @Param agreementType path string true "Agreement Type"
// @Success 200 {object} map[string]string
// @Router /plugins/businessmetrics/agreements/{projectName}/{agreementType} [delete]
func DeleteAgreement(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	projectName := input.Params["projectName"]
	agreementType := input.Params["agreementType"]

	if projectName == "" || agreementType == "" {
		return nil, errors.BadInput.New("projectName and agreementType are required")
	}

	db := basicRes.GetDal()
	err := db.Delete(&models.WorkingAgreement{}, dal.Where("project_name = ? AND agreement_type = ?", projectName, agreementType))
	if err != nil {
		return nil, errors.Default.Wrap(err, "failed to delete agreement")
	}

	return &plugin.ApiResourceOutput{
		Body:   map[string]string{"status": "deleted"},
		Status: http.StatusOK,
	}, nil
}

// @Summary List agreement violations for a project
// @Description Get all active agreement violations for a project
// @Tags plugins/businessmetrics
// @Param projectName path string true "Project Name"
// @Param includeResolved query bool false "Include resolved violations"
// @Success 200 {array} models.AgreementViolation
// @Router /plugins/businessmetrics/violations/{projectName} [get]
func ListViolations(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	projectName := input.Params["projectName"]
	if projectName == "" {
		return nil, errors.BadInput.New("projectName is required")
	}

	includeResolved := input.Query.Get("includeResolved") == "true"

	db := basicRes.GetDal()
	var violations []models.AgreementViolation

	clauses := []dal.Clause{
		dal.From(&models.AgreementViolation{}),
		dal.Where("project_name = ?", projectName),
		dal.Orderby("violated_at DESC"),
	}
	if !includeResolved {
		clauses = append(clauses, dal.Where("is_resolved = false"))
	}

	err := db.All(&violations, clauses...)
	if err != nil {
		return nil, errors.Default.Wrap(err, "failed to query violations")
	}

	return &plugin.ApiResourceOutput{
		Body:   violations,
		Status: http.StatusOK,
	}, nil
}

// @Summary Get compliance summary for a project
// @Description Get agreement compliance summaries for a project
// @Tags plugins/businessmetrics
// @Param projectName path string true "Project Name"
// @Success 200 {array} models.AgreementComplianceSummary
// @Router /plugins/businessmetrics/compliance/{projectName} [get]
func GetComplianceSummary(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	projectName := input.Params["projectName"]
	if projectName == "" {
		return nil, errors.BadInput.New("projectName is required")
	}

	db := basicRes.GetDal()
	var summaries []models.AgreementComplianceSummary
	err := db.All(&summaries,
		dal.From(&models.AgreementComplianceSummary{}),
		dal.Where("project_name = ?", projectName),
		dal.Orderby("period_start DESC"),
	)
	if err != nil {
		return nil, errors.Default.Wrap(err, "failed to query compliance summaries")
	}

	return &plugin.ApiResourceOutput{
		Body:   summaries,
		Status: http.StatusOK,
	}, nil
}
