variable "azure_subscription_id" {
  description = "Azure subscription ID"
  type        = string
}

variable "resource_group_name" {
  description = "Name of the existing Azure resource group"
  type        = string
}

variable "app_name_prefix" {
  description = "Prefix for application resources"
  type        = string
  default     = "github-migrator"
}

variable "app_service_sku" {
  description = "SKU for the App Service Plan"
  type        = string
  default     = "B1"
}

variable "always_on" {
  description = "Enable Always On for App Service"
  type        = bool
  default     = false
}

# Docker Configuration
variable "docker_registry_url" {
  description = "Docker registry URL (e.g., ghcr.io)"
  type        = string
  default     = "ghcr.io"
}

variable "docker_image_repository" {
  description = "Docker image repository (e.g., username/github-migrator)"
  type        = string
}

variable "docker_image_tag" {
  description = "Docker image tag"
  type        = string
  default     = "dev"
}

variable "docker_registry_username" {
  description = "Docker registry username"
  type        = string
  sensitive   = true
}

variable "docker_registry_password" {
  description = "Docker registry password/token"
  type        = string
  sensitive   = true
}

# Application Configuration
variable "source_type" {
  description = "Source repository type"
  type        = string
  default     = "github"
}

variable "source_base_url" {
  description = "Source API base URL"
  type        = string
  default     = "https://api.github.com"
}

variable "source_token" {
  description = "Source authentication token"
  type        = string
  sensitive   = true
}

variable "destination_type" {
  description = "Destination repository type"
  type        = string
  default     = "github"
}

variable "destination_base_url" {
  description = "Destination API base URL"
  type        = string
  default     = "https://api.github.com"
}

variable "destination_token" {
  description = "Destination authentication token"
  type        = string
  sensitive   = true
}

# Source GitHub App Configuration (optional)
variable "source_app_id" {
  description = "Source GitHub App ID (0 = not using GitHub App)"
  type        = number
  default     = 0
}

variable "source_app_private_key" {
  description = "Source GitHub App private key (PEM format)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "source_app_installation_id" {
  description = "Source GitHub App installation ID (0 = auto-discover)"
  type        = number
  default     = 0
}

# Destination GitHub App Configuration (optional)
variable "dest_app_id" {
  description = "Destination GitHub App ID (0 = not using GitHub App)"
  type        = number
  default     = 0
}

variable "dest_app_private_key" {
  description = "Destination GitHub App private key (PEM format)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "dest_app_installation_id" {
  description = "Destination GitHub App installation ID (0 = auto-discover)"
  type        = number
  default     = 0
}

# Migration Configuration
variable "migration_workers" {
  description = "Number of migration workers"
  type        = number
  default     = 3
}

variable "migration_poll_interval_seconds" {
  description = "Migration poll interval in seconds"
  type        = number
  default     = 30
}

variable "migration_post_migration_mode" {
  description = "Post migration mode"
  type        = string
  default     = "production_only"
}

variable "migration_dest_repo_exists_action" {
  description = "Action when destination repo exists"
  type        = string
  default     = "fail"
}

variable "migration_visibility_public_repos" {
  description = "Visibility for public repos"
  type        = string
  default     = "private"
}

variable "migration_visibility_internal_repos" {
  description = "Visibility for internal repos"
  type        = string
  default     = "private"
}

# Logging Configuration
variable "logging_level" {
  description = "Logging level"
  type        = string
  default     = "info"
}

variable "logging_format" {
  description = "Logging format"
  type        = string
  default     = "json"
}

# Auth Configuration
variable "auth_enabled" {
  description = "Enable authentication"
  type        = bool
  default     = false
}

variable "auth_github_oauth_client_id" {
  description = "GitHub OAuth client ID"
  type        = string
  default     = ""
}

variable "auth_github_oauth_client_secret" {
  description = "GitHub OAuth client secret"
  type        = string
  sensitive   = true
  default     = ""
}

variable "auth_callback_url" {
  description = "OAuth callback URL"
  type        = string
  default     = ""
}

variable "auth_frontend_url" {
  description = "Frontend URL"
  type        = string
  default     = ""
}

variable "auth_session_secret" {
  description = "Session secret for JWT signing"
  type        = string
  sensitive   = true
  default     = ""
}

variable "auth_session_duration_hours" {
  description = "Session duration in hours"
  type        = number
  default     = 24
}

variable "auth_require_org_membership" {
  description = "List of GitHub organizations users must be members of (empty = no requirement)"
  type        = list(string)
  default     = []
}

variable "auth_require_team_membership" {
  description = "List of GitHub teams users must be members of in format 'org/team-slug' (empty = no requirement)"
  type        = list(string)
  default     = []
}

variable "auth_require_enterprise_admin" {
  description = "Require user to be enterprise admin"
  type        = bool
  default     = false
}

variable "auth_require_enterprise_slug" {
  description = "Enterprise slug to check admin status (required if require_enterprise_admin is true)"
  type        = string
  default     = ""
}

# CORS Configuration
variable "cors_allowed_origins" {
  description = "Allowed origins for CORS"
  type        = list(string)
  default     = ["*"]
}

variable "tags" {
  description = "Tags to apply to resources"
  type        = map(string)
  default     = {}
}

