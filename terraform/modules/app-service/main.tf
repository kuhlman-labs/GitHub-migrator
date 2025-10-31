terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.51"
    }
  }
}

# App Service Plan
resource "azurerm_service_plan" "main" {
  name                = var.app_service_plan_name
  location            = var.location
  resource_group_name = var.resource_group_name
  os_type             = "Linux"
  sku_name            = var.sku_name

  tags = var.tags
}

# App Service
resource "azurerm_linux_web_app" "main" {
  name                = var.app_service_name
  location            = var.location
  resource_group_name = var.resource_group_name
  service_plan_id     = azurerm_service_plan.main.id

  https_only = true

  site_config {
    always_on         = var.always_on
    health_check_path = "/health"

    application_stack {
      docker_image_name        = var.docker_image
      docker_registry_url      = "https://${var.docker_registry_url}"
      docker_registry_username = var.docker_registry_username
      docker_registry_password = var.docker_registry_password
    }

    # CORS configuration
    cors {
      allowed_origins     = var.cors_allowed_origins
      support_credentials = true
    }
  }

  app_settings = merge(
    var.app_settings,
    {
      "WEBSITES_ENABLE_APP_SERVICE_STORAGE" = "false"
      "DOCKER_REGISTRY_SERVER_URL"          = "https://${var.docker_registry_url}"
      "DOCKER_REGISTRY_SERVER_USERNAME"     = var.docker_registry_username
      "DOCKER_REGISTRY_SERVER_PASSWORD"     = var.docker_registry_password
      "WEBSITES_PORT"                       = "8080"
    }
  )

  identity {
    type = "SystemAssigned"
  }

  logs {
    application_logs {
      file_system_level = "Information"
    }

    http_logs {
      file_system {
        retention_in_days = 7
        retention_in_mb   = 35
      }
    }
  }

  tags = var.tags
}

