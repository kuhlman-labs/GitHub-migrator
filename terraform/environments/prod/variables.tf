# Azure Infrastructure Variables
variable "azure_subscription_id" {
  description = "Azure subscription ID"
  type        = string
}

variable "resource_group_name" {
  description = "Name of the Azure resource group"
  type        = string
}

variable "location" {
  description = "Azure region for resources"
  type        = string
  default     = "eastus"
}

# App Service Configuration
variable "app_name_prefix" {
  description = "Prefix for application resources"
  type        = string
  default     = "github-migrator"
}

variable "app_service_sku" {
  description = "SKU for the App Service Plan"
  type        = string
  default     = "S1"
}

variable "always_on" {
  description = "Enable Always On for App Service"
  type        = bool
  default     = true
}

# Container Registry Configuration
variable "acr_name" {
  description = "Name of the Azure Container Registry (must be globally unique, alphanumeric only)"
  type        = string
}

variable "acr_image_name" {
  description = "Image name within ACR (e.g., github-migrator)"
  type        = string
  default     = "github-migrator"
}

variable "docker_image_tag" {
  description = "Docker image tag"
  type        = string
  default     = "prod"
}

# PostgreSQL Database Configuration (Azure resource provisioning)
variable "database_name" {
  description = "Name of the database"
  type        = string
  default     = "migrator"
}

variable "database_admin_username" {
  description = "Database admin username"
  type        = string
  default     = "psqladmin"
}

variable "postgres_version" {
  description = "PostgreSQL version"
  type        = string
  default     = "15"
}

variable "database_sku" {
  description = "Database SKU"
  type        = string
  default     = "GP_Standard_D2s_v3"
}

variable "database_storage_mb" {
  description = "Database storage in MB"
  type        = number
  default     = 32768
}

variable "database_backup_retention_days" {
  description = "Database backup retention days"
  type        = number
  default     = 30
}

variable "database_geo_redundant_backup_enabled" {
  description = "Enable geo-redundant backups"
  type        = bool
  default     = true
}

variable "database_high_availability_mode" {
  description = "High availability mode"
  type        = string
  default     = "ZoneRedundant"
}

variable "database_additional_firewall_rules" {
  description = "Additional firewall rules for database"
  type = map(object({
    start_ip = string
    end_ip   = string
  }))
  default = {}
}

variable "database_server_configurations" {
  description = "Database server configurations"
  type        = map(string)
  default     = {}
}

# Tags
variable "tags" {
  description = "Tags to apply to resources"
  type        = map(string)
  default     = {}
}

# Deployment Slots Configuration
variable "enable_staging_slot" {
  description = "Enable the staging deployment slot for zero-downtime deployments"
  type        = bool
  default     = true
}

variable "enable_dev_slot" {
  description = "Enable the dev deployment slot for development testing"
  type        = bool
  default     = true
}

# Note: Application configuration (source, destination, migration, logging, auth)
# is done via the Settings UI after deployment. The application has sensible defaults
# and stores configuration in the database.
