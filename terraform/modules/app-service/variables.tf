variable "resource_group_name" {
  description = "Name of the Azure resource group"
  type        = string
}

variable "location" {
  description = "Azure region for resources"
  type        = string
  default     = "eastus"
}

variable "app_service_plan_name" {
  description = "Name of the App Service Plan"
  type        = string
}

variable "app_service_name" {
  description = "Name of the App Service"
  type        = string
}

variable "sku_name" {
  description = "SKU name for the App Service Plan (e.g., B1, S1, P1v2)"
  type        = string
  default     = "B1"
}

variable "always_on" {
  description = "Enable Always On for the App Service"
  type        = bool
  default     = true
}

variable "docker_image" {
  description = "Docker image name with tag (e.g., github-migrator:latest)"
  type        = string
}

variable "docker_registry_url" {
  description = "Docker registry URL (e.g., ghcr.io)"
  type        = string
  default     = "ghcr.io"
}

variable "docker_registry_username" {
  description = "Docker registry username"
  type        = string
  sensitive   = true
}

variable "docker_registry_password" {
  description = "Docker registry password or token"
  type        = string
  sensitive   = true
}

variable "app_settings" {
  description = "Application settings for the App Service"
  type        = map(string)
  default     = {}
}

variable "cors_allowed_origins" {
  description = "List of allowed origins for CORS"
  type        = list(string)
  default     = ["*"]
}

variable "tags" {
  description = "Tags to apply to resources"
  type        = map(string)
  default     = {}
}

variable "storage_mounts" {
  description = "List of storage mounts for the App Service"
  type = list(object({
    name         = string
    type         = string
    account_name = string
    share_name   = string
    access_key   = string
    mount_path   = string
  }))
  default = []
}

