# Railway Deployment Guide

This guide will help you deploy the Briefly application to [Railway](https://railway.app), a modern platform-as-a-service (PaaS) that simplifies application deployment.

## Prerequisites

1. **Railway Account**: Sign up at [railway.app](https://railway.app)
2. **GitHub Repository**: Your code should be pushed to GitHub
3. **Gemini API Key**: Required for article summarization
   - Get your key at: [Google AI Studio](https://makersuite.google.com/app/apikey)

## Architecture Overview

The deployment consists of two Railway services:

1. **PostgreSQL Database** - Managed database for storing articles, digests, and metadata
2. **Web Application** - The Briefly Go server

## Quick Start Deployment

### Step 1: Create a New Railway Project

1. Go to [railway.app/new](https://railway.app/new)
2. Click "Deploy from GitHub repo"
3. Select your `briefly` repository
4. Railway will create a new project

### Step 2: Add PostgreSQL Database

1. In your Railway project dashboard, click "+ New"
2. Select "Database" → "Add PostgreSQL"
3. Railway will provision a PostgreSQL instance
4. The `DATABASE_URL` environment variable will be automatically created and linked

### Step 3: Configure Environment Variables

In your web service settings, add the following environment variables:

#### Required Variables

```bash
# Gemini API Key (required for summarization)
GEMINI_API_KEY=your-gemini-api-key-here
```

#### Auto-Configured Variables (Railway provides these automatically)

- `PORT` - Automatically set by Railway (typically 3000-8000)
- `DATABASE_URL` - Automatically set when PostgreSQL is added
  - Format: `postgresql://user:password@host:5432/railway?sslmode=disable`

#### Optional Variables (for observability)

```bash
# LangFuse (LLM tracing)
LANGFUSE_PUBLIC_KEY=pk-lf-...
LANGFUSE_SECRET_KEY=sk-lf-...
LANGFUSE_HOST=https://cloud.langfuse.com

# PostHog (analytics)
POSTHOG_API_KEY=phc_...
POSTHOG_HOST=https://app.posthog.com

# Application settings (optional overrides)
LOG_LEVEL=info
DEBUG=false
```

### Step 4: Deploy

Railway will automatically:

1. Build your Docker image using the `Dockerfile`
2. Run migrations via the release command (`./briefly migrate up`)
3. Start your application server
4. Expose it on a public URL (e.g., `briefly-production.up.railway.app`)

## Deployment Configuration Files

The following files configure your Railway deployment:

### `Dockerfile`

- **Multi-stage build**: Builds Go binary and creates minimal runtime image
- **Security**: Runs as non-root user
- **Health check**: Built-in health endpoint at `/health`
- **Size**: ~50MB final image (Alpine-based)

### `railway.json`

Configures Railway-specific deployment settings:

```json
{
  "build": {
    "builder": "DOCKERFILE",
    "dockerfilePath": "Dockerfile"
  },
  "deploy": {
    "releaseCommand": "./briefly migrate up",
    "healthcheckPath": "/health",
    "restartPolicyType": "ON_FAILURE"
  }
}
```

### `.dockerignore`

Excludes unnecessary files from Docker image to reduce build time and image size.

## Database Migrations

**Automatic Migration:** The `releaseCommand` in `railway.json` runs migrations before each deployment:

```bash
./briefly migrate up
```

This ensures your database schema is always up-to-date with the latest code.

**Manual Migration (if needed):**

If you need to run migrations manually:

1. Open Railway project dashboard
2. Click on your web service
3. Go to "Settings" → "Deploy"
4. Use the "Run Command" feature:
   ```bash
   ./briefly migrate up
   ```

## Monitoring & Logs

### View Logs

Railway provides real-time logs:

1. Go to your service in Railway dashboard
2. Click "Logs" tab
3. Filter by log level (info, warn, error)

### Health Check

The application exposes several health endpoints:

- **Health Check**: `GET /health` - Returns 200 if healthy
- **Status**: `GET /api/status` - Returns application status

### Metrics

Railway provides built-in metrics:

- CPU usage
- Memory usage
- Network traffic
- Request count

## Post-Deployment Verification

### 1. Check Deployment Status

Verify your deployment succeeded:

```bash
curl https://your-app.up.railway.app/health
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2025-11-07T10:00:00Z"
}
```

### 2. Verify Database Connection

Check if the database is connected:

```bash
curl https://your-app.up.railway.app/api/status
```

Expected response should include database status.

### 3. Test Key Endpoints

- **Homepage**: `GET /`
- **API Status**: `GET /api/status`
- **Digests**: `GET /api/digests`
- **Articles**: `GET /api/articles/recent`
- **Themes**: `GET /api/themes`

## Running Commands in Production

Railway allows you to run one-off commands in your deployed environment:

1. Go to your service in Railway dashboard
2. Navigate to "Settings" → "Deploy"
3. Use "Run Command" to execute:

### Add RSS Feeds

```bash
./briefly feed add https://hnrss.org/newest
./briefly feed add https://blog.golang.org/feed.atom
```

### Aggregate Articles

```bash
./briefly aggregate --since 24
```

### Generate Digest

```bash
./briefly digest generate --since 7
```

### Check Database Status

```bash
./briefly migrate status
```

## Automated Scheduling with Railway Cron

Railway supports native cron jobs for automated tasks. Create separate services for scheduled tasks.

### How Railway Cron Works

- **Separate services**: Each cron job is a dedicated service that executes and exits
- **Minimum frequency**: 5 minutes (cannot run more often)
- **Timezone**: All schedules use UTC
- **Cost efficient**: Only pay for execution time (seconds to minutes)

### Recommended Cron Services

#### 1. Daily Aggregation Service

Fetches articles from feeds daily.

**Setup in Railway Dashboard:**
1. Click "+ New" → "Empty Service"
2. Connect to your GitHub repository
3. Configure:
   - **Service Name**: `daily-aggregation`
   - **Build Command**: `go build -o briefly ./cmd/briefly`
   - **Start Command**: `./briefly aggregate --since 24 --themes`
   - **Cron Schedule**: `0 2 * * *` (2 AM UTC daily)
4. Link environment variables: `DATABASE_URL`, `GEMINI_API_KEY`

#### 2. Weekly Digest Service

Generates weekly digest every Monday.

**Setup:**
1. Click "+ New" → "Empty Service"
2. Connect to same repository
3. Configure:
   - **Service Name**: `weekly-digest`
   - **Build Command**: `go build -o briefly ./cmd/briefly`
   - **Start Command**: `./briefly digest generate --since 7`
   - **Cron Schedule**: `0 10 * * 1` (10 AM UTC every Monday)
4. Link environment variables: `DATABASE_URL`, `GEMINI_API_KEY`

#### 3. Frequent Updates (Optional)

For more up-to-date content throughout the day.

**Setup:**
- **Service Name**: `frequent-aggregation`
- **Start Command**: `./briefly aggregate --since 4 --themes`
- **Cron Schedule**: `0 */4 * * *` (every 4 hours)

### Cron Schedule Examples

```bash
# Every 5 minutes (minimum)
*/5 * * * *

# Every hour
0 * * * *

# Every 4 hours
0 */4 * * *

# Every day at 2 AM UTC
0 2 * * *

# Weekdays at 9 AM UTC
0 9 * * 1-5

# Every Monday at 10 AM UTC
0 10 * * 1

# First day of month at midnight
0 0 1 * *
```

**Tool**: Use [crontab.guru](https://crontab.guru/) to test expressions

### Timezone Conversion

Railway uses UTC. Convert your local time:

| Your Time | UTC Equivalent |
|-----------|----------------|
| 6 AM PST | 2 PM UTC |
| 9 AM PST | 5 PM UTC |
| 6 PM PST | 2 AM UTC (next day) |
| 9 AM EST | 2 PM UTC |
| 6 PM EST | 11 PM UTC |

### Monitoring Cron Jobs

**View Execution Logs:**
```bash
# Via Railway CLI
railway logs -s daily-aggregation
railway logs -s weekly-digest --follow

# Via Dashboard
Go to service → Observability → Logs
```

**Manual Trigger (Testing):**
1. Go to cron service in dashboard
2. Click "Deploy" → "Trigger Deploy"
3. Watch logs to verify execution

### Cost Optimization

Cron services are extremely cost-efficient:

- **Daily 2-minute task**: ~$0.01-0.05/month
- **Weekly 5-minute task**: ~$0.001/month
- **Comparison**: 24/7 service = $10-20/month

Most cron jobs fall within Railway's free credits ($5 Hobby, $20 Pro).

### Important Notes

✅ **Your code is already ready** - Commands exit properly after completion

⚠️ **Limitations:**
- Minimum frequency: 5 minutes
- Timing may vary by a few minutes
- Always UTC timezone
- No concurrent execution (skips if previous run still active)

## Scaling

Railway supports horizontal and vertical scaling:

### Vertical Scaling (Increase Resources)

1. Go to "Settings" → "Resources"
2. Adjust CPU and Memory limits
3. Redeploy

### Horizontal Scaling (Multiple Replicas)

Edit `railway.json`:

```json
{
  "deploy": {
    "numReplicas": 2
  }
}
```

**Note:** Ensure your application is stateless for horizontal scaling.

## Troubleshooting

### Issue: Deployment Fails to Start

**Check:**
1. Logs for error messages
2. Environment variables are set correctly
3. Database connection is working

**Solution:**
```bash
# Verify DATABASE_URL is set
echo $DATABASE_URL

# Test database connection manually
./briefly migrate status
```

### Issue: Migration Fails

**Error:** `migration failed: connection refused`

**Solution:**
1. Verify PostgreSQL service is running
2. Check `DATABASE_URL` environment variable
3. Ensure network connectivity between services

**Manual migration:**
```bash
./briefly migrate up --verbose
```

### Issue: Health Check Failing

**Error:** `GET /health returns 502 or 503`

**Possible causes:**
1. Server not listening on Railway's `PORT`
2. Application crashed during startup
3. Health check path incorrect

**Solution:**
Check logs for startup errors:
```bash
railway logs
```

### Issue: Out of Memory

**Error:** `OOMKilled` in logs

**Solution:**
1. Increase memory allocation in Railway settings
2. Check for memory leaks in application logs
3. Optimize application memory usage

### Issue: API Keys Not Working

**Error:** `Gemini API key is required`

**Solution:**
1. Verify `GEMINI_API_KEY` is set in Railway environment variables
2. Check API key is valid (no spaces, correct format)
3. Ensure key has not expired or reached quota limits

## Environment Configuration Priority

The application follows this configuration priority (highest to lowest):

1. **Command-line flags** - `--port`, `--host`, etc.
2. **Environment variables** - `PORT`, `DATABASE_URL`, `GEMINI_API_KEY`
3. **Config file** - `.briefly.yaml` (if present in repo)
4. **Default values** - Port 8080, host 0.0.0.0

Railway automatically provides `PORT` and `DATABASE_URL`, so no manual configuration needed.

## Security Best Practices

1. **Never commit secrets** to your repository
   - Use Railway's environment variables for all sensitive data
   - Add `.env` files to `.gitignore` (already configured)

2. **Use HTTPS** - Railway provides automatic HTTPS for all deployments

3. **Database Security**
   - Railway PostgreSQL uses SSL by default
   - Connection strings include authentication
   - Database is not publicly accessible (only accessible to your services)

4. **API Keys**
   - Store in Railway environment variables
   - Rotate keys regularly
   - Monitor usage for anomalies

## Cost Optimization

Railway offers:

- **Starter Plan**: $5/month + usage
- **Pro Plan**: $20/month + usage
- **Free Trial**: Limited resources

### Cost-Saving Tips

1. **Use vertical scaling** instead of horizontal (fewer instances)
2. **Set sleep policies** for non-production environments
3. **Monitor usage** in Railway dashboard
4. **Optimize Docker image** (already optimized at ~50MB)

## CI/CD Integration

Railway automatically deploys on every push to your main branch:

1. **Push to GitHub** → Railway detects change
2. **Build Docker image** → Uses Dockerfile
3. **Run migrations** → Executes release command
4. **Deploy** → Starts new container
5. **Health check** → Verifies deployment

### Customize Deployment Branch

By default, Railway deploys from `main`. To change:

1. Go to "Settings" → "Service"
2. Update "Branch" setting
3. Save changes

## Rollback

If a deployment fails, Railway keeps previous versions:

1. Go to "Deployments" tab
2. Find the last successful deployment
3. Click "..." → "Redeploy"

## Custom Domain

To use a custom domain (e.g., `briefly.yourdomain.com`):

1. Go to "Settings" → "Networking"
2. Click "Add Custom Domain"
3. Enter your domain
4. Update DNS records as shown by Railway
5. Railway will provision SSL certificate automatically

## Additional Resources

- **Railway Documentation**: [docs.railway.app](https://docs.railway.app)
- **Briefly GitHub**: Your repository URL
- **Support**: Railway Discord server

## Summary Checklist

- [ ] Railway account created
- [ ] PostgreSQL database added
- [ ] `GEMINI_API_KEY` environment variable set
- [ ] Application deployed successfully
- [ ] Health check passing (`/health` returns 200)
- [ ] Database migrations completed
- [ ] RSS feeds added (optional)
- [ ] Custom domain configured (optional)

## Next Steps

After deployment:

1. **Add RSS Feeds**: Use Railway's command runner to add your preferred feeds
2. **Schedule Aggregation**: Set up cron job or Railway scheduled task to run `./briefly aggregate --since 24` daily
3. **Generate Digests**: Run weekly digest generation via scheduled task
4. **Monitor Logs**: Check logs regularly for errors or issues
5. **Set Up Alerts**: Configure Railway alerts for downtime or errors

Your Briefly application is now live and ready to aggregate news! 🎉
