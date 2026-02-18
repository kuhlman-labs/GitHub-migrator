variable "acr_name" {
  description = "Name of the Azure Container Registry (must be globally unique, alphanumeric only)"
  type        = string

  validation {
    condition     = can(regex("^[a-zA-Z0-9]{5,50}$", var.acr_name))
    error_message = "ACR name must be 5-50 alphanumeric characters only."
  }
}

variable "resource_group_name" {
  description = "Name of the Azure resource group"
  type        = string
}

variable "location" {
  description = "Azure region for the registry"
  type        = string
}

variable "sku" {
  description = "SKU tier for the container registry"
  type        = string
  default     = "Basic"
}

variable "tags" {
  description = "Tags to apply to the registry"
  type        = map(string)
  default     = {}
}
