# Bounce Processor on Railway — Quick Start

## TL;DR

```bash
# 1. Install Railway CLI
npm install -g @railway/cli

# 2. Authenticate
railway login

# 3. Link to your Railway project
cd bounce-processor
railway link

# 4. Add environment variables (via Railway dashboard)
#    Settings → Variables
#    POP3_HOST: mail.defendsurviveprepare.com
#    POP3_PORT: 995
#    POP3_USER: bounces@defendsurviveprepare.com
#    POP3_PASS: [from Mailcow]
#    LISTMONK_URL: https://postmaster.defendsurviveprepare.com
#    LISTMONK_USER: pmapi
#    LISTMONK_PASS: [from Listmonk API tokens]

# 5. Deploy
railway up

# 6. Monitor
railway logs -f
```

## What happens

- **Deploy starts** → builds Docker image → uploads to Railway
- **Service runs** → scans bounces immediately, then every 5 hours
- **Logs appear** → Railway dashboard Logs tab or `railway logs -f`

## Get credentials

### Mailcow (bounces@ password)
1. Go to Mailcow Admin → Mailboxes
2. Find `bounces@defendsurviveprepare.com`
3. Click "Login" button to test/verify the password
4. Copy password

### Listmonk (API key)
1. Go to Listmonk → Settings → API Tokens
2. Under "pmapi" user, copy the API key

## Environment variables (Railway dashboard)

Go to your Railway project → Bounce Processor service → Variables tab

Paste these:

```
POP3_HOST=mail.defendsurviveprepare.com
POP3_PORT=995
POP3_USER=bounces@defendsurviveprepare.com
POP3_PASS=[your mailcow password]
LISTMONK_URL=https://postmaster.defendsurviveprepare.com
LISTMONK_USER=pmapi
LISTMONK_PASS=[your listmonk api key]
```

Save. Railway will auto-restart the service with new variables.

## Check if it's working

```bash
railway logs -f
```

You should see:
```
{"timestamp":"2026-08-22T23:45:00Z","level":"INFO","message":"Bounce processor started",...}
{"timestamp":"2026-08-22T23:45:05Z","level":"INFO","message":"Scanning bounces","message_count":5}
{"timestamp":"2026-08-22T23:45:06Z","level":"INFO","message":"Bounce processed","email":"user@example.com"}
```

## Cost

**~$5-6/month** on Railway (base service + minimal compute for 5-hour scans).

## Control

- **Restart:** Railway dashboard → Services → Bounce Processor → Restart
- **Stop:** Railway dashboard → Stop (pauses bounce processing)
- **View logs:** `railway logs -f` or Railway dashboard Logs tab
- **Update config:** Railway dashboard Variables tab → Save (auto-restarts)

## That's it

The service will scan bounces every 5 hours automatically. Logs appear in Railway dashboard.

See [RAILWAY-DEPLOY.md](RAILWAY-DEPLOY.md) for full docs.
