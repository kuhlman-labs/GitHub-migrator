terraform {
  required_version = ">= 1.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.51"
    }
  }

  # Configure backend for state storage
  backend "azurerm" {
    resource_group_name  = "mcoe-opps"
    storage_account_name = "tfstateghmig854247"
    container_name       = "tfstate"
    key                  = "github-migrator-prod.tfstate"
  }
}

provider "azurerm" {
  features {}
  subscription_id = var.azure_subscription_id
}

# Create resource group
resource "azurerm_resource_group" "main" {
  name     = var.resource_group_name
  location = var.location

  tags = merge(
    var.tags,
    {
      Environment = "production"
      ManagedBy   = "Terraform"
    }
  )
}

# Deploy PostgreSQL Database
module "postgresql" {
  source = "../../modules/postgresql"

  resource_group_name          = azurerm_resource_group.main.name
  location                     = azurerm_resource_group.main.location
  server_name                  = "${var.app_name_prefix}-db-prod"
  database_name                = var.database_name
  admin_username               = var.database_admin_username
  postgres_version             = var.postgres_version
  sku_name                     = var.database_sku
  storage_mb                   = var.database_storage_mb
  backup_retention_days        = var.database_backup_retention_days
  geo_redundant_backup_enabled = var.database_geo_redundant_backup_enabled
  high_availability_mode       = var.database_high_availability_mode

  additional_firewall_rules = var.database_additional_firewall_rules
  server_configurations     = var.database_server_configurations

  tags = merge(
    var.tags,
    {
      Environment = "prod"
      ManagedBy   = "Terraform"
    }
  )
}

# Deploy App Service (with PostgreSQL)
module "app_service" {
  source = "../../modules/app-service"

  depends_on = [azurerm_resource_group.main, module.postgresql]

  resource_group_name      = azurerm_resource_group.main.name
  location                 = azurerm_resource_group.main.location
  app_service_plan_name    = "${var.app_name_prefix}-plan-prod"
  app_service_name         = "${var.app_name_prefix}-prod"
  sku_name                 = var.app_service_sku
  always_on                = var.always_on
  docker_image             = "${var.docker_registry_url}/${lower(var.docker_image_repository)}:${var.docker_image_tag}"
  docker_registry_url      = var.docker_registry_url
  docker_registry_username = var.docker_registry_username
  docker_registry_password = var.docker_registry_password

  app_settings = {
    # Server Configuration
    "GHMIG_SERVER_PORT" = "8080"

    # Database Configuration (PostgreSQL for prod)
    "GHMIG_DATABASE_TYPE" = "postgres"
    "GHMIG_DATABASE_DSN"  = module.postgresql.dsn

    # Source Configuration
    "GHMIG_SOURCE_TYPE"     = var.source_type
    "GHMIG_SOURCE_BASE_URL" = var.source_base_url
    "GHMIG_SOURCE_TOKEN"    = var.source_token

    # Source GitHub App Configuration (optional)
    "GHMIG_SOURCE_APP_ID"              = tostring(var.source_app_id)
    "GHMIG_SOURCE_APP_PRIVATE_KEY"     = var.source_app_private_key
    "GHMIG_SOURCE_APP_INSTALLATION_ID" = tostring(var.source_app_installation_id)

    # Destination Configuration
    "GHMIG_DESTINATION_TYPE"     = var.destination_type
    "GHMIG_DESTINATION_BASE_URL" = var.destination_base_url
    "GHMIG_DESTINATION_TOKEN"    = var.destination_token

    # Destination GitHub App Configuration (optional)
    "GHMIG_DESTINATION_APP_ID"              = tostring(var.dest_app_id)
    "GHMIG_DESTINATION_APP_PRIVATE_KEY"     = var.dest_app_private_key
    "GHMIG_DESTINATION_APP_INSTALLATION_ID" = tostring(var.dest_app_installation_id)

    # Migration Configuration
    "GHMIG_MIGRATION_WORKERS"                            = tostring(var.migration_workers)
    "GHMIG_MIGRATION_POLL_INTERVAL_SECONDS"              = tostring(var.migration_poll_interval_seconds)
    "GHMIG_MIGRATION_POST_MIGRATION_MODE"                = var.migration_post_migration_mode
    "GHMIG_MIGRATION_DEST_REPO_EXISTS_ACTION"            = var.migration_dest_repo_exists_action
    "GHMIG_MIGRATION_VISIBILITY_HANDLING_PUBLIC_REPOS"   = var.migration_visibility_public_repos
    "GHMIG_MIGRATION_VISIBILITY_HANDLING_INTERNAL_REPOS" = var.migration_visibility_internal_repos

    # Logging Configuration
    "GHMIG_LOGGING_LEVEL"  = var.logging_level
    "GHMIG_LOGGING_FORMAT" = var.logging_format

    # Auth Configuration (optional)
    "GHMIG_AUTH_ENABLED"                    = tostring(var.auth_enabled)
    "GHMIG_AUTH_GITHUB_OAUTH_CLIENT_ID"     = var.auth_github_oauth_client_id
    "GHMIG_AUTH_GITHUB_OAUTH_CLIENT_SECRET" = var.auth_github_oauth_client_secret
    "GHMIG_AUTH_CALLBACK_URL"               = var.auth_callback_url
    "GHMIG_AUTH_FRONTEND_URL"               = var.auth_frontend_url
    "GHMIG_AUTH_SESSION_SECRET"             = var.auth_session_secret
    "GHMIG_AUTH_SESSION_DURATION_HOURS"     = tostring(var.auth_session_duration_hours)

    # Auth Authorization Rules
    "GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_ORG_MEMBERSHIP"   = jsonencode(var.auth_require_org_membership)
    "GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_TEAM_MEMBERSHIP"  = jsonencode(var.auth_require_team_membership)
    "GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_ENTERPRISE_ADMIN" = tostring(var.auth_require_enterprise_admin)
    "GHMIG_AUTH_AUTHORIZATION_RULES_REQUIRE_ENTERPRISE_SLUG"  = var.auth_require_enterprise_slug

    # Environment
    "ENVIRONMENT" = "prod"
  }

  cors_allowed_origins = var.cors_allowed_origins

  tags = merge(
    var.tags,
    {
      Environment = "prod"
      ManagedBy   = "Terraform"
    }
  )
}

