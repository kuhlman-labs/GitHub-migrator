package models

import "time"

// ADOProject represents an Azure DevOps project
type ADOProject struct {
	ID              int64     `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	Organization    string    `json:"organization" db:"organization" gorm:"column:organization;not null;index"`
	Name            string    `json:"name" db:"name" gorm:"column:name;not null"`
	Description     *string   `json:"description,omitempty" db:"description" gorm:"column:description;type:text"`
	RepositoryCount int       `json:"repository_count" db:"repository_count" gorm:"column:repository_count;default:0"`
	State           string    `json:"state" db:"state" gorm:"column:state"`                // wellFormed, createPending, etc.
	Visibility      string    `json:"visibility" db:"visibility" gorm:"column:visibility"` // private, public
	DiscoveredAt    time.Time `json:"discovered_at" db:"discovered_at" gorm:"column:discovered_at;not null"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at" gorm:"column:updated_at;not null;autoUpdateTime"`
}

// TableName specifies the table name for ADOProject model
func (ADOProject) TableName() string {
	return "ado_projects"
}
