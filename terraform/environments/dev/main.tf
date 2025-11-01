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
    resource_group_name  = "terraform-state-rg"
    storage_account_name = "tfstateghmig854248"
    container_name       = "tfstate"
    key                  = "github-migrator-dev.tfstate"
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
      Environment = "dev"
      ManagedBy   = "Terraform"
    }
  )
}

# Storage account for persistent database storage
resource "azurerm_storage_account" "dev_db" {
  name                     = "${replace(var.app_name_prefix, "-", "")}devdb"
  resource_group_name      = azurerm_resource_group.main.name
  location                 = azurerm_resource_group.main.location
  account_tier             = "Standard"
  account_replication_type = "LRS"

  tags = merge(
    var.tags,
    {
      Environment = "dev"
      ManagedBy   = "Terraform"
      Purpose     = "database-persistence"
    }
  )
}

# File share for SQLite database
resource "azurerm_storage_share" "dev_db" {
  name                 = "database"
  storage_account_name = azurerm_storage_account.dev_db.name
  quota                = 1

  depends_on = [azurerm_storage_account.dev_db]
}

# Deploy App Service (with SQLite - no database dependency)
module "app_service" {
  source = "../../modules/app-service"

  depends_on = [azurerm_resource_group.main]

  resource_group_name      = azurerm_resource_group.main.name
  location                 = azurerm_resource_group.main.location
  app_service_plan_name    = "${var.app_name_prefix}-plan-dev"
  app_service_name         = "${var.app_name_prefix}-dev"
  sku_name                 = var.app_service_sku
  always_on                = var.always_on
  docker_image             = "${var.docker_registry_url}/${lower(var.docker_image_repository)}:${var.docker_image_tag}"
  docker_registry_url      = var.docker_registry_url
  docker_registry_username = var.docker_registry_username
  docker_registry_password = var.docker_registry_password

  app_settings = {
    # Server Configuration
    "GHMIG_SERVER_PORT" = "8080"

    # Database Configuration (SQLite for dev)
    "GHMIG_DATABASE_TYPE" = "sqlite"
    "GHMIG_DATABASE_DSN"  = "/app/data/migrator.db"

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
    "ENVIRONMENT" = "dev"
  }

  cors_allowed_origins = var.cors_allowed_origins

  # Mount Azure File Share for database persistence
  storage_mounts = [
    {
      name         = "database-mount"
      type         = "AzureFiles"
      account_name = azurerm_storage_account.dev_db.name
      share_name   = azurerm_storage_share.dev_db.name
      access_key   = azurerm_storage_account.dev_db.primary_access_key
      mount_path   = "/app/data"
    }
  ]

  tags = merge(
    var.tags,
    {
      Environment = "dev"
      ManagedBy   = "Terraform"
    }
  )
}

