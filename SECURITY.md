# Security Guide - Guardian Tracker

## Credential Management

### CRITICAL: Rotate All Credentials

If you have previously committed credentials to this repository, you MUST:

1. **Rotate ALL credentials immediately:**
   - Bungie API Key: Generate new key at https://www.bungie.net/en/Application
   - Bungie Client Secret: Regenerate in your Bungie application settings
   - JWT Secret: Generate new 32+ character random string
   - Database passwords: Change in your database and update configs

2. **Remove credentials from git history:**
   ```bash
   # Option 1: BFG Repo Cleaner (recommended)
   bfg --delete-files .env
   bfg --replace-text passwords.txt  # File containing secrets to remove

   # Option 2: git filter-branch (slower)
   git filter-branch --force --index-filter \
     "git rm --cached --ignore-unmatch .env" \
     --prune-empty --tag-name-filter cat -- --all
   ```

3. **Force push and notify collaborators:**
   ```bash
   git push origin --force --all
   git push origin --force --tags
   ```

## Environment Variables

### Required Secrets (Never Commit!)

| Variable | Description | Where to Get |
|----------|-------------|--------------|
| `BUNGIE_API_KEY` | Bungie API access key | [Bungie Applications](https://www.bungie.net/en/Application) |
| `BUNGIE_CLIENT_ID` | OAuth client ID | Bungie application settings |
| `BUNGIE_CLIENT_SECRET` | OAuth client secret | Bungie application settings |
| `JWT_SECRET` | Token signing key (32+ chars) | Generate with `openssl rand -base64 32` |

### Setting Up Secrets

#### Local Development
```bash
# Copy example file
cp .env.example .env

# Edit with your actual values
# NEVER use production credentials in development
```

#### Kubernetes Production
```bash
# Create secret from literal values (don't put in scripts!)
kubectl create secret generic auth-service-secrets \
  --from-literal=BUNGIE_API_KEY=your_key \
  --from-literal=BUNGIE_CLIENT_ID=your_id \
  --from-literal=BUNGIE_CLIENT_SECRET=your_secret \
  --from-literal=JWT_SECRET=your_jwt_secret

# Or from a secure file (then delete the file!)
kubectl create secret generic auth-service-secrets --from-env-file=.env.secrets
rm .env.secrets
```

## Security Features Implemented

### Authentication
- JWT-based authentication with access/refresh token rotation
- CSRF protection on OAuth flow (state parameter)
- Token expiration (24h access, 30d refresh)
- Server-side state validation

### Input Validation
- Zod schema validation on all GraphQL inputs
- Maximum length limits on all string inputs
- Type validation for membership IDs and types

### Rate Limiting
- 100 requests per 15 minutes per IP (production)
- Protects against brute force and DoS

### Error Handling
- Stack traces hidden in production
- Generic error messages for unknown errors
- Detailed logging server-side only

### CORS
- Strict origin validation in production
- Credentials require explicit origin match

## Security Checklist for Production

- [ ] All secrets rotated and removed from git history
- [ ] JWT_SECRET is 32+ characters, randomly generated
- [ ] CORS_ALLOWED_ORIGINS set to your domain only
- [ ] NODE_ENV=production on all services
- [ ] GO_ENV=production on Go services
- [ ] Rate limiting enabled
- [ ] TLS/HTTPS configured
- [ ] Database connections use SSL
- [ ] Health endpoints don't expose sensitive info
- [ ] Logging doesn't include tokens or secrets

## Reporting Security Issues

If you discover a security vulnerability, please email [security@your-domain.com] instead of opening a public issue.
