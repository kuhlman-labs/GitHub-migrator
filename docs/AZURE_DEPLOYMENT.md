# Azure Deployment Guide

This guide provides complete instructions for deploying the GitHub Migrator application to Azure App Services using Terraform and GitHub Actions.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Architecture](#architecture)
- [Azure Setup](#azure-setup)
- [Terraform Deployment](#terraform-deployment)
- [GitHub Actions Setup](#github-actions-setup)
- [Post-Deployment Configuration](#post-deployment-configuration)
- [Monitoring and Operations](#monitoring-and-operations)
- [Troubleshooting](#troubleshooting)

## Overview

This deployment creates:
- **Dev Environment**: Azure App Service with SQLite database
- **Production Environment**: Azure App Service with PostgreSQL Flexible Server
- **CI/CD Pipeline**: GitHub Actions for automated builds and deployments
- **Container Registry**: GitHub Container Registry (GHCR) for Docker images

## Prerequisites

### Required Tools
- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli) >= 2.0
- [Git](https://git-scm.com/downloads)
- GitHub account with repository admin access

### Required Access
- Azure subscription with contributor access
- Existing Azure Resource Group
- GitHub repository with admin permissions

## Architecture

### Dev Environment
```
GitHub Actions → GHCR → Azure App Service
                         └─ SQLite (local storage)
```

### Production Environment
```
GitHub Actions → GHCR → Azure App Service
                         └─ PostgreSQL Flexible Server
```

## Azure Setup

### 1. Create Azure Service Principal

Create a service principal for GitHub Actions to authenticate with Azure:

```bash
# Login to Azure
az login

# Get your subscription ID
az account show --query id -o tsv

# Create service principal
az ad sp create-for-rbac \
  --name "github-migrator-deploy" \
  --role contributor \
  --scopes /subscriptions/{SUBSCRIPTION_ID}/resourceGroups/{RESOURCE_GROUP_NAME} \
  --sdk-auth
```

Save the JSON output - you'll need it for GitHub secrets.

### 2. Verify Resource Group

Ensure your resource group exists:

```bash
az group show --name YOUR_RESOURCE_GROUP_NAME
```

If it doesn't exist, create it:

```bash
az group create --name YOUR_RESOURCE_GROUP_NAME --location eastus
```

## Terraform Deployment

### Dev Environment Deployment

#### 1. Navigate to Dev Environment

```bash
cd terraform/environments/dev
```

#### 2. Create terraform.tfvars

Copy the example file and fill in your values:

```bash
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars` with your actual values:

```hcl
# Azure Configuration
azure_subscription_id = "your-subscription-id"
resource_group_name   = "your-resource-group"

# Docker Configuration
docker_image_repository  = "your-github-username/github-migrator"
docker_registry_username = "your-github-username"
docker_registry_password = "your-github-token"

# Application Configuration
source_token      = "your-source-github-token"
destination_token = "your-destination-github-token"

# Add other required values...
```

#### 3. Initialize and Apply Terraform

```bash
# Initialize Terraform
terraform init

# Review the plan
terraform plan

# Apply the configuration
terraform apply
```

#### 4. Save Outputs

```bash
# Save important outputs
terraform output app_service_name
terraform output app_service_url
```

### Production Environment Deployment

Follow the same steps for production:

```bash
cd terraform/environments/prod

# Copy and edit terraform.tfvars
cp terraform.tfvars.example terraform.tfvars
vim terraform.tfvars

# Initialize and apply
terraform init
terraform plan
terraform apply

# Save outputs (including database credentials)
terraform output -json > outputs.json
```

**Important**: Store database credentials securely:

```bash
# Get database password (sensitive)
terraform output database_admin_password

# Store in Azure Key Vault or secure password manager
```

## GitHub Actions Setup

### 1. Configure Repository Secrets

Navigate to your GitHub repository: Settings → Secrets and variables → Actions

Add the following secrets:

#### Required Secrets

| Secret Name | Description | Example/Notes |
|------------|-------------|---------------|
| `AZURE_CREDENTIALS` | Service principal JSON from Azure setup | Full JSON output from `az ad sp create-for-rbac` |
| `AZURE_SUBSCRIPTION_ID` | Your Azure subscription ID | UUID format |
| `AZURE_RESOURCE_GROUP` | Resource group name | e.g., `github-migrator-rg` |
| `AZURE_APP_SERVICE_NAME_DEV` | Dev App Service name | Output from Terraform dev |
| `AZURE_APP_SERVICE_NAME_PROD` | Production App Service name | Output from Terraform prod |

#### Optional Secrets (if using GitHub App authentication)

| Secret Name | Description |
|------------|-------------|
| `SOURCE_APP_ID` | Source GitHub App ID |
| `SOURCE_APP_PRIVATE_KEY` | Source GitHub App private key |
| `DESTINATION_APP_ID` | Destination GitHub App ID |
| `DESTINATION_APP_PRIVATE_KEY` | Destination GitHub App private key |

### 2. Configure GitHub Environments

Create environments for deployment protection:

#### Dev Environment
1. Go to Settings → Environments → New environment
2. Name: `dev`
3. No protection rules needed for dev

#### Production Environment
1. Go to Settings → Environments → New environment
2. Name: `production`
3. Enable required reviewers (recommended)
4. Add yourself as a required reviewer
5. Enable wait timer if desired (e.g., 5 minutes)

### 3. Enable GitHub Container Registry

Ensure your repository has packages enabled:

1. Go to repository Settings → Actions → General
2. Scroll to "Workflow permissions"
3. Select "Read and write permissions"
4. Check "Allow GitHub Actions to create and approve pull requests"

### 4. Test the Workflows

#### Build Workflow

The build workflow runs automatically on:
- Push to `main` branch
- Pull requests
- Manual trigger

To manually trigger:
1. Go to Actions → Build and Push Container Image
2. Click "Run workflow"
3. Select branch and click "Run workflow"

#### Deploy to Dev

Automatically runs after successful build on `main` branch, or manually:
1. Go to Actions → Deploy to Dev Environment
2. Click "Run workflow"

#### Deploy to Production

Manual trigger or on release:
1. Go to Actions → Deploy to Production Environment
2. Click "Run workflow"
3. Enter image tag (e.g., `prod`, `v1.0.0`)
4. Click "Run workflow"
5. Approve the deployment if required reviewers are configured

## Post-Deployment Configuration

### 1. Verify Deployment

#### Dev Environment

```bash
# Test health endpoint
curl https://github-migrator-dev.azurewebsites.net/health

# Expected response: {"status":"ok"}
```

#### Production Environment

```bash
# Test health endpoint
curl https://github-migrator-prod.azurewebsites.net/health
```

### 2. Configure GitHub OAuth (if using authentication)

If you enabled authentication, configure OAuth:

1. Create GitHub OAuth App:
   - Go to GitHub Settings → Developer settings → OAuth Apps → New OAuth App
   - Application name: `GitHub Migrator (Production)`
   - Homepage URL: `https://your-app-name-prod.azurewebsites.net`
   - Callback URL: `https://your-app-name-prod.azurewebsites.net/api/v1/auth/callback`

2. Update Terraform variables:
   ```hcl
   auth_enabled                    = true
   auth_github_oauth_client_id     = "your-oauth-client-id"
   auth_github_oauth_client_secret = "your-oauth-client-secret"
   auth_callback_url               = "https://your-app-url/api/v1/auth/callback"
   auth_frontend_url               = "https://your-app-url"
   ```

3. Reapply Terraform:
   ```bash
   cd terraform/environments/prod
   terraform apply
   ```

### 3. Configure Custom Domain (Optional)

To use a custom domain:

1. In Azure Portal:
   - Navigate to your App Service
   - Select "Custom domains"
   - Click "Add custom domain"
   - Follow the wizard

2. Update CORS and OAuth URLs to use custom domain

### 4. Configure SSL/TLS

Azure App Services automatically provides SSL for `*.azurewebsites.net` domains.

For custom domains:
1. Azure Portal → App Service → TLS/SSL settings
2. Upload certificate or use App Service Managed Certificate (free)

### 5. Database Migrations

Production database migrations run automatically on app startup. To verify:

```bash
# Check App Service logs
az webapp log tail --name github-migrator-prod --resource-group YOUR_RG
```

Look for migration success messages.

## Monitoring and Operations

### Application Logs

#### View logs via Azure CLI

```bash
# Stream logs
az webapp log tail --name YOUR_APP_NAME --resource-group YOUR_RG

# Download logs
az webapp log download --name YOUR_APP_NAME --resource-group YOUR_RG
```

#### View logs in Azure Portal

1. Navigate to App Service
2. Select "Log stream" from left menu
3. View real-time logs

### Metrics and Alerts

Configure monitoring in Azure Portal:

1. Navigate to App Service → Monitoring → Metrics
2. Common metrics to monitor:
   - CPU Percentage
   - Memory Percentage
   - HTTP 5xx errors
   - Response Time
   - Data In/Out

3. Set up alerts:
   - Navigate to Alerts → New alert rule
   - Define conditions (e.g., CPU > 80%)
   - Configure action groups for notifications

### Database Monitoring (Production)

1. Navigate to PostgreSQL Flexible Server
2. Monitoring → Metrics
3. Monitor:
   - CPU Percentage
   - Memory Percentage
   - Storage Used
   - Active Connections
   - Replication Lag (if HA enabled)

### Application Insights (Optional)

For advanced monitoring:

```bash
# Create Application Insights
az monitor app-insights component create \
  --app github-migrator-prod \
  --location eastus \
  --resource-group YOUR_RG
  
# Get instrumentation key
az monitor app-insights component show \
  --app github-migrator-prod \
  --resource-group YOUR_RG \
  --query instrumentationKey -o tsv
```

Add to App Service environment variables:
- `APPLICATIONINSIGHTS_CONNECTION_STRING`

## Troubleshooting

### Container Pull Failures

**Problem**: App Service can't pull container image from GHCR

**Solution**:
1. Verify GHCR token has `read:packages` permission
2. Check container image exists:
   ```bash
   docker pull ghcr.io/your-username/github-migrator:dev
   ```
3. Verify App Service settings have correct registry credentials

### Database Connection Issues (Production)

**Problem**: App can't connect to PostgreSQL

**Solution**:
1. Verify firewall rules allow Azure services:
   ```bash
   az postgres flexible-server firewall-rule list \
     --resource-group YOUR_RG \
     --name YOUR_DB_SERVER
   ```
2. Check connection string in App Service configuration
3. Verify database exists and credentials are correct
4. Check database server is running

### App Won't Start

**Problem**: App Service shows "Application Error"

**Solution**:
1. Check logs:
   ```bash
   az webapp log tail --name YOUR_APP_NAME --resource-group YOUR_RG
   ```
2. Verify all environment variables are set correctly
3. Ensure container image is valid
4. Check health endpoint path is `/health`

### Migrations Failing

**Problem**: Database migrations fail on startup

**Solution**:
1. Check database connectivity
2. Verify database user has CREATE permissions
3. Check migration files are included in container image
4. Review migration logs in application logs

### High Memory Usage

**Problem**: App Service hitting memory limits

**Solution**:
1. Scale up App Service Plan:
   ```bash
   az appservice plan update \
     --name YOUR_PLAN_NAME \
     --resource-group YOUR_RG \
     --sku S2
   ```
2. Review application logs for memory leaks
3. Consider adjusting migration workers count

## Scaling

### Vertical Scaling (Scale Up)

Increase App Service Plan tier:

```bash
az appservice plan update \
  --name github-migrator-plan-prod \
  --resource-group YOUR_RG \
  --sku P1v2
```

### Horizontal Scaling (Scale Out)

Add more instances:

```bash
az appservice plan update \
  --name github-migrator-plan-prod \
  --resource-group YOUR_RG \
  --number-of-workers 2
```

Or configure auto-scaling in Azure Portal:
1. App Service Plan → Scale out
2. Enable auto-scale
3. Set rules based on metrics (CPU, Memory, etc.)

## Backup and Disaster Recovery

### Database Backups (Production)

PostgreSQL Flexible Server automatically creates backups based on retention period.

To restore:

```bash
# List backup times
az postgres flexible-server backup list \
  --resource-group YOUR_RG \
  --name YOUR_DB_SERVER

# Restore to point in time
az postgres flexible-server restore \
  --resource-group YOUR_RG \
  --name YOUR_DB_SERVER \
  --restore-time "2024-01-01T00:00:00Z" \
  --source-server /subscriptions/{sub-id}/resourceGroups/{rg}/providers/Microsoft.DBforPostgreSQL/flexibleServers/{server}
```

### Application Backup

Container images are versioned in GHCR, allowing easy rollback:

1. Find previous working version in GHCR
2. Update Terraform `docker_image_tag` variable
3. Run `terraform apply`

Or use GitHub Actions deploy workflow with specific tag.

## Cost Optimization

### Dev Environment
- Use B1 tier (Basic) - ~$13/month
- SQLite storage (no database costs)
- Can be stopped when not in use

### Production Environment
- S1 tier - ~$70/month
- PostgreSQL Flexible Server B_Standard_B1ms - ~$30/month
- Consider reserved instances for long-term savings

### Tips
1. Use deployment slots for staging instead of separate environments
2. Enable auto-scaling to scale down during low usage
3. Monitor and right-size resources based on actual usage
4. Consider Azure Dev/Test pricing if eligible

## Security Best Practices

1. **Secrets Management**
   - Never commit secrets to Git
   - Use Azure Key Vault for production secrets
   - Rotate credentials regularly

2. **Network Security**
   - Configure CORS allowed origins in production (don't use *)
   - Enable HTTPS only
   - Consider Private Endpoints for database

3. **Authentication**
   - Enable GitHub OAuth for production
   - Configure authorization rules
   - Use strong session secrets (32+ characters)

4. **Database Security**
   - Use strong admin password
   - Enable SSL for connections
   - Configure firewall rules (restrict to Azure services)
   - Enable threat protection

5. **Monitoring**
   - Enable Azure Security Center
   - Configure alerts for suspicious activity
   - Review access logs regularly

## Support and Maintenance

### Regular Maintenance Tasks

1. **Weekly**
   - Review application logs for errors
   - Check application metrics
   - Verify backups are running

2. **Monthly**
   - Review and optimize costs
   - Update dependencies
   - Review security advisories

3. **Quarterly**
   - Test disaster recovery procedures
   - Review and update documentation
   - Audit access permissions

### Getting Help

- **Application Issues**: Check application logs and [OPERATIONS.md](./OPERATIONS.md)
- **Azure Issues**: Azure Support Portal
- **Terraform Issues**: Review Terraform state and plan output
- **GitHub Actions Issues**: Check workflow logs in GitHub Actions tab

## Additional Resources

- [Azure App Services Documentation](https://docs.microsoft.com/en-us/azure/app-service/)
- [PostgreSQL Flexible Server Documentation](https://docs.microsoft.com/en-us/azure/postgresql/flexible-server/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Terraform Azure Provider Documentation](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs)
- [GitHub Container Registry Documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)

