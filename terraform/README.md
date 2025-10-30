# Terraform Infrastructure

This directory contains Terraform configurations for deploying the GitHub Migrator application to Azure App Services.

## Structure

```
terraform/
├── modules/
│   ├── app-service/       # Azure App Service module
│   └── postgresql/        # PostgreSQL Flexible Server module
└── environments/
    ├── dev/              # Development environment
    └── prod/             # Production environment
```

## Modules

### app-service

Deploys an Azure App Service with:
- Linux App Service Plan
- Docker container support
- GHCR integration
- Health check configuration
- Managed identity
- Application settings and secrets

### postgresql

Deploys Azure PostgreSQL Flexible Server with:
- Configurable SKU and storage
- Automated backups
- High availability options
- Firewall rules
- Database creation

## Environments

### Development (dev)

- **Database**: SQLite (embedded)
- **App Service**: Basic tier (B1)
- **Always On**: Disabled (to save costs)
- **Purpose**: Testing and development

### Production (prod)

- **Database**: PostgreSQL Flexible Server
- **App Service**: Standard tier (S1) or higher
- **Always On**: Enabled
- **High Availability**: Zone redundant
- **Backups**: 30-day retention with geo-redundancy
- **Purpose**: Production workloads

## Quick Start

### Prerequisites

1. Install [Terraform](https://www.terraform.io/downloads.html) >= 1.0
2. Install [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)
3. Login to Azure: `az login`

### Deploy Development Environment

```bash
# Navigate to dev environment
cd environments/dev

# Copy and configure variables
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values

# Initialize Terraform
terraform init

# Preview changes
terraform plan

# Deploy
terraform apply
```

### Deploy Production Environment

```bash
# Navigate to prod environment
cd environments/prod

# Copy and configure variables
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values

# Initialize Terraform
terraform init

# Preview changes
terraform plan

# Deploy
terraform apply

# Save outputs securely
terraform output -json > outputs.json
chmod 600 outputs.json
```

## Required Variables

### Common Variables (both environments)

| Variable | Description | Required |
|----------|-------------|----------|
| `azure_subscription_id` | Azure subscription ID | Yes |
| `resource_group_name` | Existing resource group name | Yes |
| `docker_image_repository` | GHCR image repository | Yes |
| `docker_registry_username` | GitHub username | Yes |
| `docker_registry_password` | GitHub token with packages read | Yes |
| `source_token` | Source GitHub PAT | Yes |
| `destination_token` | Destination GitHub PAT | Yes |

### Production-Specific Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `database_sku` | PostgreSQL SKU | `GP_Standard_D2s_v3` |
| `database_storage_mb` | Storage size in MB | `32768` |
| `database_backup_retention_days` | Backup retention | `30` |
| `auth_enabled` | Enable authentication | `true` |

## Outputs

### Development Environment

- `app_service_url` - URL of the deployed application
- `app_service_name` - Name of the App Service
- `app_service_identity_principal_id` - Managed identity principal ID

### Production Environment

All dev outputs plus:
- `database_server_fqdn` - PostgreSQL server FQDN
- `database_name` - Database name
- `database_admin_username` - Database admin username (sensitive)
- `database_admin_password` - Database admin password (sensitive)

## State Management

### Local State (Default)

By default, Terraform state is stored locally in `terraform.tfstate`.

**Warning**: Local state should not be used for production. Multiple users or CI/CD pipelines will conflict.

### Remote State (Recommended for Production)

Configure Azure Storage backend for state:

1. Create storage account for Terraform state:

```bash
# Create resource group for Terraform state
az group create --name terraform-state-rg --location eastus

# Create storage account
az storage account create \
  --name tfstate$(uuidgen | tr -d '-' | tr '[:upper:]' '[:lower:]' | cut -c1-17) \
  --resource-group terraform-state-rg \
  --location eastus \
  --sku Standard_LRS

# Create container
az storage container create \
  --name tfstate \
  --account-name YOUR_STORAGE_ACCOUNT_NAME
```

2. Uncomment and configure backend in `main.tf`:

```hcl
backend "azurerm" {
  resource_group_name  = "terraform-state-rg"
  storage_account_name = "YOUR_STORAGE_ACCOUNT_NAME"
  container_name       = "tfstate"
  key                  = "github-migrator-prod.tfstate"
}
```

3. Initialize with backend:

```bash
terraform init -migrate-state
```

## Managing Secrets

### Sensitive Variables

Never commit sensitive values to Git. Use one of these methods:

#### 1. terraform.tfvars (Local Development)

```hcl
# terraform.tfvars (add to .gitignore)
source_token      = "ghp_xxxxxxxxxx"
destination_token = "ghp_xxxxxxxxxx"
```

#### 2. Environment Variables

```bash
export TF_VAR_source_token="ghp_xxxxxxxxxx"
export TF_VAR_destination_token="ghp_xxxxxxxxxx"
terraform apply
```

#### 3. Azure Key Vault (Recommended for Production)

Store secrets in Azure Key Vault and reference them:

```bash
# Store secret in Key Vault
az keyvault secret set \
  --vault-name YOUR_VAULT \
  --name source-token \
  --value "ghp_xxxxxxxxxx"

# Retrieve in Terraform (requires additional configuration)
```

## Updating the Infrastructure

### Update Application Settings

1. Modify `app_settings` in `main.tf`
2. Run `terraform plan` to review changes
3. Run `terraform apply` to update

### Update Docker Image Tag

```bash
# Update variable
terraform apply -var="docker_image_tag=v1.2.0"
```

Or update in `terraform.tfvars`:

```hcl
docker_image_tag = "v1.2.0"
```

Then apply:

```bash
terraform apply
```

### Scale Up/Down

Update the SKU in `terraform.tfvars`:

```hcl
# Scale up to Standard tier
app_service_sku = "S2"
```

Then apply:

```bash
terraform apply
```

## Destroying Infrastructure

**Warning**: This will permanently delete all resources and data.

### Development Environment

```bash
cd environments/dev
terraform destroy
```

### Production Environment

**Backup database first!**

```bash
cd environments/prod

# Backup database
az postgres flexible-server backup create \
  --resource-group YOUR_RG \
  --name YOUR_DB_SERVER

# Then destroy
terraform destroy
```

## Troubleshooting

### Error: Resource Group Not Found

Ensure the resource group exists:

```bash
az group show --name YOUR_RESOURCE_GROUP
```

Create if needed:

```bash
az group create --name YOUR_RESOURCE_GROUP --location eastus
```

### Error: Provider Configuration

Ensure Azure CLI is logged in:

```bash
az login
az account show
```

Set correct subscription:

```bash
az account set --subscription YOUR_SUBSCRIPTION_ID
```

### Error: State Lock

If Terraform state is locked:

```bash
# Force unlock (use with caution)
terraform force-unlock LOCK_ID
```

### Container Pull Errors

Verify GHCR credentials:

```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u YOUR_USERNAME --password-stdin
docker pull ghcr.io/YOUR_USERNAME/github-migrator:dev
```

## Best Practices

1. **Always run `terraform plan` before `apply`**
2. **Use remote state for production**
3. **Store sensitive outputs securely**
4. **Tag all resources appropriately**
5. **Use workspaces or separate directories for environments**
6. **Keep modules version-pinned in production**
7. **Document all customizations**
8. **Regular state backups**
9. **Use `.tfvars` files, never hardcode secrets**
10. **Review and audit changes in production**

## Additional Resources

- [Terraform Azure Provider](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs)
- [Azure App Service Terraform](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/linux_web_app)
- [Azure PostgreSQL Terraform](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/postgresql_flexible_server)
- [Terraform Best Practices](https://www.terraform.io/docs/cloud/guides/recommended-practices/index.html)

## Support

For deployment issues, see [AZURE_DEPLOYMENT.md](../docs/AZURE_DEPLOYMENT.md) for detailed troubleshooting.

