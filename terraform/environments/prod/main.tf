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

# Reference existing resource group
data "azurerm_resource_group" "main" {
  name = var.resource_group_name
}

# Deploy PostgreSQL Database
module "postgresql" {
  source = "../../modules/postgresql"

  resource_group_name          = data.azurerm_resource_group.main.name
  location                     = data.azurerm_resource_group.main.location
  server_name                  = "${var.app_name_prefix}-db-prod"
  database_name                = var.database_name
  admin_username               = var.database_admin_username
  postgres_version             = var.postgres_version
  sku_name                     = var.database_sku
  storage_mb                   = var.database_storage_mb
  backup_retention_days        = var.database_backup_retention_days
  geo_redundant_backup_enabled = var.database_geo_redundant_backup_enabled
  high_availability_mode       = var.database_high_availability_mode
  availability_zone            = "3"
  standby_availability_zone    = "1"

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

# Deploy App Service (with PostgreSQL and Deployment Slots)
module "app_service" {
  source = "../../modules/app-service"

  depends_on = [module.postgresql]

  resource_group_name      = data.azurerm_resource_group.main.name
  location                 = data.azurerm_resource_group.main.location
  app_service_plan_name    = "${var.app_name_prefix}-plan-prod"
  app_service_name         = "${var.app_name_prefix}-prod"
  sku_name                 = var.app_service_sku
  always_on                = var.always_on
  docker_image             = "${lower(var.docker_image_repository)}:${var.docker_image_tag}"
  docker_registry_url      = var.docker_registry_url
  docker_registry_username = var.docker_registry_username
  docker_registry_password = var.docker_registry_password

  # Enable deployment slots for zero-downtime deployments
  enable_staging_slot = var.enable_staging_slot
  enable_dev_slot     = var.enable_dev_slot

  # Minimal application settings - all other configuration is done via the UI
  # after deployment. The application has sensible defaults for source, destination,
  # migration, logging, and auth settings.
  app_settings = {
    # Server Configuration
    "GHMIG_SERVER_PORT" = "8080"

    # Database Configuration (PostgreSQL for prod)
    "GHMIG_DATABASE_TYPE" = "postgres"
    "GHMIG_DATABASE_DSN"  = module.postgresql.dsn

    # Environment
    "ENVIRONMENT" = "prod"
  }

  # Staging slot uses same database (for pre-prod testing)
  staging_slot_app_settings = {
    "GHMIG_DATABASE_TYPE" = "postgres"
    "GHMIG_DATABASE_DSN"  = module.postgresql.dsn
  }

  # Dev slot uses same database (for development testing)
  dev_slot_app_settings = {
    "GHMIG_DATABASE_TYPE" = "postgres"
    "GHMIG_DATABASE_DSN"  = module.postgresql.dsn
  }

  tags = merge(
    var.tags,
    {
      Environment = "prod"
      ManagedBy   = "Terraform"
    }
  )
}

