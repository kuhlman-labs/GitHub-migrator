variable "acr_name" {
  description = "Name of the Azure Container Registry (must be globally unique, alphanumeric only)"
  type        = string
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
