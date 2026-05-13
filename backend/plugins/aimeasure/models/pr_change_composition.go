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

package models

import "time"

// BatchBucket buckets PR sizes for distribution analysis.
type BatchBucket string

const (
	BucketXS BatchBucket = "XS" // < 50 LOC
	BucketS  BatchBucket = "S"  // 50 - 200 LOC
	BucketM  BatchBucket = "M"  // 200 - 500 LOC
	BucketL  BatchBucket = "L"  // 500 - 1000 LOC
	BucketXL BatchBucket = "XL" // > 1000 LOC
)

// PRChangeComposition records the size and refactor-ratio characteristics of a merged PR.
// One row per PR; written once at merge time.
type PRChangeComposition struct {
	PRId          string      `gorm:"primaryKey;type:varchar(255)" json:"prId"`
	Additions     int         `gorm:"type:int" json:"additions"`
	Deletions     int         `gorm:"type:int" json:"deletions"`
	FileCount     int         `gorm:"type:int" json:"fileCount"`
	AdditiveLines int         `gorm:"type:int" json:"additiveLines"`
	RefactorLines int         `gorm:"type:int" json:"refactorLines"`
	RefactorRatio float64     `gorm:"type:decimal(5,4)" json:"refactorRatio"`
	BatchBucket   BatchBucket `gorm:"type:varchar(4);not null;index" json:"batchBucket"`
	ComputedAt    time.Time   `gorm:"not null" json:"computedAt"`
}

func (PRChangeComposition) TableName() string {
	return "pr_change_composition"
}
