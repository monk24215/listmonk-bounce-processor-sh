# Bounce Processor Deployment on Railway

## Setup

### 1. Install Railway CLI

```bash
npm install -g @railway/cli
```

### 2. Authenticate with Railway

```bash
railway login
```

### 3. Link to your Railway project

```bash
cd bounce-processor
railway link
```

Select your project where Listmonk is running.

### 4. Add environment variables to Railway

Go to your Railway project dashboard → Bounce Processor service (or create it if needed) → Variables tab.

Add these variables:

| Variable | Value |
|----------|-------|
| `POP3_HOST` | `mail.defendsurviveprepare.com` |
| `POP3_PORT` | `995` |
| `POP3_USER` | `bounces@defendsurviveprepare.com` |
| `POP3_PASS` | **[Password from Mailcow]** |
| `LISTMONK_URL` | `https://postmaster.defendsurviveprepare.com` |
| `LISTMONK_USER` | `pmapi` |
| `LISTMONK_PASS` | **[API key from Listmonk]** |

### 5. Deploy to Railway

```bash
railway up
```

This will:
- Build the Docker image
- Push to Railway
- Start the service
- Begin polling bounces every 5 hours

### 6. View logs

```bash
railway logs -f
```

Or in Railway dashboard → Logs tab.

## How it works on Railway

1. **Service starts** → scans bounces immediately
2. **Processes bounces** → calls Listmonk API
3. **Waits 5 hours** → sleeps
4. **Repeats** → scans again, processes, sleeps

Logs appear in Railway dashboard (Live Logs tab).

## Monitoring

### View logs live
```bash
railway logs -f
```

### View specific service logs
```bash
railway logs --service bounce-processor -f
```

### Check service status
```bash
railway status
```

## Operations

### Stop the service (pauses bounces)
```bash
railway deploy --detach --no-build
```

Then stop via Railway dashboard: Services → Bounce Processor → Stop

### Restart the service
Railway dashboard → Services → Bounce Processor → Restart

### Update environment variables
Railway dashboard → Variables tab → Edit and save

Service will auto-restart with new variables.

### Redeploy (rebuild and push)
```bash
railway up --force
```

## Cost implications

Railway pricing:
- **$5/month** base per-service
- **Compute:** ~$0.01/hour per vCPU when running
- **5-hour scan:** ~0.005 hours per month = negligible

**Estimated cost:** ~$5-6/month for the bounce processor service alone.

To reduce cost further:
- Increase scan interval (currently 5 hours)
- Use Railway Crons (when available): Schedule one scan per day instead

## Troubleshooting

### Service won't start
```bash
railway logs -f
```

Check for error messages. Common issues:
- Wrong POP3 credentials
- Wrong Listmonk API key
- Listmonk service unreachable

### No bounces being processed
- Check if bounces exist in Mailcow: Log into Mailcow web UI
- Check Listmonk bounce settings: Settings → Bounces → Hard bounce action = "Unsubscribe"
- Wait for next 5-hour scan cycle

### Logs not appearing
```bash
railway logs --tail 100 -f
```

Or check Railway dashboard → Logs tab (may take 30s to appear).

## Updating the code

If you update main.go:

```bash
# Rebuild and redeploy
railway up
```

Railway will detect changes, rebuild the image, and deploy.

## Removing the service

```bash
# Stop via Railway dashboard, then:
railway down
```

This will delete the service from Railway.

## Multiple environments

If you have staging/production on Railway:

```bash
# List projects
railway list

# Switch projects
railway link

# Deploy to specific project
railway up
```

## Performance notes

- Scan takes **<1 second** for typical 10-50 bounces
- Memory: **~10MB** per run
- CPU: **Minimal** (exits after scan)

5-hour scans minimize cost while keeping bounces fresh.
