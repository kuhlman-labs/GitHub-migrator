# Documentation

Complete documentation for GitHub Migrator deployment and operations.

## 📚 Quick Links

### Getting Started
- **[Terraform Deployment Quick Start](./TERRAFORM_DEPLOYMENT_QUICKSTART.md)** ⭐ START HERE
  - 30-minute guide to deploy everything
  - Step-by-step with commands
  - Checklist format

### Deployment
- **[GitHub Environments Setup](./GITHUB_ENVIRONMENTS_SETUP.md)** ⭐ RECOMMENDED
  - Use GitHub Environments for better organization
  - Separate dev and production configurations
  - Protection rules and security

- **[GitHub Secrets Setup](./GITHUB_SECRETS_SETUP.md)** (Alternative)
  - Repository-level secrets approach
  - Complete list of required secrets
  - Security best practices

- **[Environments vs Secrets Comparison](./ENVIRONMENTS_VS_SECRETS.md)**
  - Compare both approaches
  - Decision matrix
  - Migration guide

- **[GitHub App Setup](./GITHUB_APP_SETUP.md)** 📱 OPTIONAL
  - Enhanced discovery and profiling
  - Higher rate limits
  - Multi-organization support
  - Only needed for enterprise-scale

- **[Azure Deployment Guide](./AZURE_DEPLOYMENT.md)**
  - Comprehensive deployment documentation
  - Architecture details
  - Monitoring and operations
  - Troubleshooting guide

### Infrastructure
- **[Terraform README](../terraform/README.md)**
  - Terraform module documentation
  - Environment configurations
  - State management
  - Best practices

### CI/CD
- **[GitHub Actions Workflows](../.github/workflows/README.md)**
  - Workflow documentation
  - How to use each workflow
  - Branch protection setup
  - Troubleshooting

### Application
- **[Operations Guide](./OPERATIONS.md)**
  - Day-to-day operations
  - Monitoring
  - Backup and recovery
  - Common tasks

- **[API Documentation](./API.md)**
  - API endpoints
  - Request/response formats
  - Authentication

## 🚀 Deployment Flow

```mermaid
graph LR
    A[1. Setup GitHub Secrets] --> B[2. Run Terraform]
    B --> C[3. Build Container]
    C --> D[4. Deploy App]
    D --> E[✅ Live!]
```

1. **Setup GitHub Secrets** (10 min)
   - Follow [GITHUB_SECRETS_SETUP.md](./GITHUB_SECRETS_SETUP.md)

2. **Run Terraform** (5 min)
   - Follow [TERRAFORM_DEPLOYMENT_QUICKSTART.md](./TERRAFORM_DEPLOYMENT_QUICKSTART.md)

3. **Build Container** (auto)
   - Triggers automatically on push

4. **Deploy App** (auto)
   - Dev deploys automatically
   - Prod requires manual trigger

## 📖 Documentation Structure

```
docs/
├── README.md (this file)
├── TERRAFORM_DEPLOYMENT_QUICKSTART.md   ⭐ Start here
├── GITHUB_ENVIRONMENTS_SETUP.md         GitHub Environments guide
├── GITHUB_SECRETS_SETUP.md              Required secrets reference
├── GITHUB_APP_SETUP.md                  📱 GitHub App setup (optional)
├── ENVIRONMENTS_VS_SECRETS.md           Compare approaches
├── AZURE_DEPLOYMENT.md                  Comprehensive deployment guide
├── OPERATIONS.md                        Day-to-day operations
├── API.md                               API documentation
├── CONTRIBUTING.md                      How to contribute
├── DEPLOYMENT.md                        Deployment strategies
└── IMPLEMENTATION_GUIDE.md              Implementation details

terraform/
└── README.md                            Terraform documentation

.github/workflows/
└── README.md                            CI/CD workflow docs
```

## 🎯 Common Tasks

### First-Time Deployment
1. Read [TERRAFORM_DEPLOYMENT_QUICKSTART.md](./TERRAFORM_DEPLOYMENT_QUICKSTART.md)
2. Follow step-by-step
3. Verify deployment

### Update Configuration
1. Update GitHub Secrets
2. Run Terraform workflow
3. Restart app service

### Update Application Code
1. Commit and push changes
2. Build workflow runs automatically
3. Deploy workflow runs (dev auto, prod manual)

### Troubleshooting
1. Check [AZURE_DEPLOYMENT.md](./AZURE_DEPLOYMENT.md#troubleshooting)
2. Review workflow logs
3. Check Azure App Service logs

## 🔐 Security

All sensitive values are stored as **GitHub Secrets**, never in code:
- Azure credentials
- GitHub tokens
- OAuth secrets
- Session secrets
- Database credentials

See [GITHUB_SECRETS_SETUP.md](./GITHUB_SECRETS_SETUP.md) for complete security guide.

## 🆘 Support

- **Deployment Issues**: [AZURE_DEPLOYMENT.md](./AZURE_DEPLOYMENT.md#troubleshooting)
- **Workflow Issues**: [GitHub Actions Workflows](../.github/workflows/README.md#troubleshooting)
- **Application Issues**: [OPERATIONS.md](./OPERATIONS.md)

## 🎉 Quick Win

Follow these files in order for fastest deployment:

1. **[GITHUB_SECRETS_SETUP.md](./GITHUB_SECRETS_SETUP.md)** - 10 minutes
2. **[TERRAFORM_DEPLOYMENT_QUICKSTART.md](./TERRAFORM_DEPLOYMENT_QUICKSTART.md)** - 20 minutes
3. ✅ **Done!** Your app is live

Total time: ~30 minutes from zero to deployed application! 🚀

