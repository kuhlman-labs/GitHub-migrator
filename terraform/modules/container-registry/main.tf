terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 5.1"
    }
  }
}

resource "azurerm_container_registry" "main" {
  name                = var.acr_name
  resource_group_name = var.resource_group_name
  location            = var.location
  sku                 = var.sku

  # Admin credentials are used by CI to push images.
  # App Service pulls use managed identity instead.
  admin_enabled = true

  tags = var.tags
}
