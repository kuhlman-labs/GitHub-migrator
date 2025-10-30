# GitHub Environments vs Repository Secrets

Quick comparison to help you choose the best approach for your deployment.

## 🎯 TL;DR - Which Should I Use?

**✅ Use GitHub Environments if:**
- You have multiple environments (dev, staging, prod)
- You want protection rules (approvals for production)
- You need different configurations per environment
- You prefer better organization and visibility

**✅ Use Repository Secrets if:**
- You only have one environment
- You want simpler, faster setup
- You don't need protection rules
- Your configurations are identical across environments

## 📊 Detailed Comparison

| Feature | GitHub Environments | Repository Secrets |
|---------|-------------------|-------------------|
| **Organization** | Excellent - grouped by environment | Basic - all in one list |
| **Setup Complexity** | Moderate - more steps | Simple - fewer steps |
| **Protection Rules** | ✅ Yes (approvals, wait timers) | ❌ No |
| **Visibility** | Secrets hidden, variables visible | All secrets hidden |
| **Configuration per Env** | ✅ Easy - separate configs | ⚠️ Need naming suffix (DEV/PROD) |
| **Deployment History** | ✅ Per environment | ❌ No tracking |
| **Best For** | Multi-environment setups | Single environment or simple setups |

## 🏗️ Architecture Comparison

### GitHub Environments Approach

```
Repository
├── Shared Secrets (3)
│   ├── AZURE_CREDENTIALS
│   ├── AZURE_SUBSCRIPTION_ID
│   └── AZURE_RESOURCE_GROUP
│
├── Environment: dev
│   ├── Variables (20+) ← Non-sensitive config
│   │   ├── APP_SERVICE_SKU = "B1"
│   │   ├── MIGRATION_WORKERS = "3"
│   │   └── ... 
│   └── Secrets (5) ← Sensitive data
│       ├── SOURCE_GITHUB_TOKEN
│       ├── DEST_GITHUB_TOKEN
│       ├── AUTH_GITHUB_OAUTH_CLIENT_SECRET
│       └── AUTH_SESSION_SECRET
│
└── Environment: production
    ├── Variables (25+) ← Different from dev
    │   ├── APP_SERVICE_SKU = "S1"
    │   ├── MIGRATION_WORKERS = "5"
    │   ├── DATABASE_SKU = "GP_Standard_D2s_v3"
    │   └── ...
    └── Secrets (5) ← Separate from dev
        ├── SOURCE_GITHUB_TOKEN
        ├── DEST_GITHUB_TOKEN
        ├── AUTH_GITHUB_OAUTH_CLIENT_SECRET
        └── AUTH_SESSION_SECRET
```

**Total Items:**
- Shared: 3 secrets
- Dev: 20 variables + 5 secrets
- Prod: 25 variables + 5 secrets
- **Total: 58 items**

### Repository Secrets Approach

```
Repository
└── Secrets (20+)
    ├── AZURE_CREDENTIALS
    ├── AZURE_SUBSCRIPTION_ID
    ├── AZURE_RESOURCE_GROUP
    ├── AZURE_APP_SERVICE_NAME_DEV
    ├── AZURE_APP_SERVICE_NAME_PROD
    ├── SOURCE_GITHUB_TOKEN
    ├── DEST_GITHUB_TOKEN
    ├── AUTH_ENABLED_DEV
    ├── AUTH_ENABLED_PROD
    ├── AUTH_GITHUB_OAUTH_CLIENT_ID_DEV
    ├── AUTH_GITHUB_OAUTH_CLIENT_ID_PROD
    ├── AUTH_GITHUB_OAUTH_CLIENT_SECRET_DEV
    ├── AUTH_GITHUB_OAUTH_CLIENT_SECRET_PROD
    ├── AUTH_SESSION_SECRET_DEV
    ├── AUTH_SESSION_SECRET_PROD
    └── ...
```

**Total Items:**
- All: ~20-25 secrets (with _DEV/_PROD suffixes)

## 💡 Key Differences

### 1. Configuration Visibility

**Environments:**
```yaml
# Visible in UI as VARIABLE
APP_SERVICE_SKU = "B1"
MIGRATION_WORKERS = "3"

# Hidden as SECRET
SOURCE_GITHUB_TOKEN = "ghp_xxxxx"
```

**Repository Secrets:**
```yaml
# ALL hidden as secrets
APP_SERVICE_SKU_DEV = "B1"
MIGRATION_WORKERS_DEV = "3"
SOURCE_GITHUB_TOKEN = "ghp_xxxxx"
```

### 2. Protection Rules

**Environments:**
```yaml
production:
  protection_rules:
    - required_reviewers: [@you, @team]
    - wait_timer: 5 minutes
    - allowed_branches: [main]
```

**Repository Secrets:**
```
No protection rules available
Must rely on branch protection only
```

### 3. Workflow Usage

**Environments:**
```yaml
jobs:
  deploy-dev:
    environment: dev  # ← One line to specify
    steps:
      - run: echo "${{ vars.APP_SERVICE_SKU }}"  # No suffix needed
      - run: echo "${{ secrets.SOURCE_GITHUB_TOKEN }}"
```

**Repository Secrets:**
```yaml
jobs:
  deploy-dev:
    steps:
      - run: echo "${{ secrets.APP_SERVICE_SKU_DEV }}"  # Need suffix
      - run: echo "${{ secrets.SOURCE_GITHUB_TOKEN }}"
```

## 🎓 Recommendations

### Start Simple, Grow Complex

1. **Just Learning?** 
   - Start with Repository Secrets
   - Fewer concepts to learn
   - Faster to set up

2. **Going to Production?**
   - Switch to Environments
   - Better organization for long term
   - Protection rules prevent accidents

3. **Already Using Repository Secrets?**
   - It works fine! No urgent need to switch
   - Consider migrating when you add staging/more environments

### Migration Path

If you want to migrate from Repository Secrets to Environments:

1. Create environments (dev, production)
2. Copy secrets to appropriate environments
3. Convert non-sensitive values to variables
4. Update workflows to use `environment: dev`
5. Test thoroughly
6. Delete old repository secrets (with _DEV/_PROD suffixes)

## 📚 Documentation Links

- **[GitHub Environments Setup Guide](./GITHUB_ENVIRONMENTS_SETUP.md)** - Complete guide
- **[GitHub Secrets Setup Guide](./GITHUB_SECRETS_SETUP.md)** - Repository secrets guide
- **[Terraform Deployment Quickstart](./TERRAFORM_DEPLOYMENT_QUICKSTART.md)** - Works with both approaches

## ✅ Our Recommendation

**For this project, we recommend GitHub Environments because:**

1. ✅ You have two distinct environments (dev + production)
2. ✅ Production needs protection rules (avoid accidental deployments)
3. ✅ Configurations differ significantly (SQLite vs PostgreSQL, SKUs, workers)
4. ✅ Better visibility of non-sensitive configs as variables
5. ✅ Cleaner workflow files (no _DEV/_PROD suffixes)
6. ✅ Deployment history per environment

**Setup time difference:** ~10 minutes extra for environments setup, but worth it for the benefits!

## 🎯 Quick Decision Matrix

| Your Situation | Recommendation |
|----------------|----------------|
| New project with dev + prod | **Environments** ⭐ |
| Only one environment | **Repository Secrets** |
| Need production approvals | **Environments** (only option) |
| Want to see config values | **Environments** (use variables) |
| Quick prototype/demo | **Repository Secrets** |
| Team collaboration | **Environments** (clearer) |
| Enterprise deployment | **Environments** (more control) |

---

**Ready to set up?**
- 👉 [GitHub Environments Setup](./GITHUB_ENVIRONMENTS_SETUP.md)
- 👉 [Repository Secrets Setup](./GITHUB_SECRETS_SETUP.md)

