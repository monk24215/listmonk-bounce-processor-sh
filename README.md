# Bounce Processor for Listmonk + Mailcow

A Go service that polls Mailcow's bounce mailbox and automatically processes bounces in Listmonk via REST API.

## What it does

1. **Every 5 minutes:**
   - Connects to `bounces@defendsurviveprepare.com` inbox in Mailcow (via POP3)
   - Parses Delivery Status Notifications (DSNs) for bounce information
   - Extracts failed email addresses and bounce type (hard/soft)

2. **For each bounce:**
   - Calls Listmonk's REST API (`/api/bounces`)
   - Listmonk processes the bounce (unsubscribes, etc.)
   - Deletes the message from Mailcow (prevents reprocessing)

3. **Logs everything:**
   - Structured JSON logging to `/var/log/bounce-processor.log`
   - Systemd journal integration (viewable via `journalctl`)
   - Error tracking and alerting-ready format

## Why this approach?

**Redundancy:** Two independent bounce processing paths:
- Listmonk's native POP3 scanner (primary)
- This service (secondary fallback)

**Production-safe:** At 280K subscribers, a silent bounce processing failure damages reputation. Having two independent scanners ensures bounces are caught.

**Observable:** Structured JSON logging means you can plug in monitoring/alerting (Prometheus, DataDog, etc.) without code changes.

**Simple:** No external dependencies beyond Go stdlib. Single binary, easy to run/monitor/toggle.

## Architecture

```
Incoming mail (Cloudflare)
    ↓
Mailcow Postfix
    ↓
bounces@defendsurviveprepare.com inbox
    ↓
Bounce Processor (this service - polls every 5m)
    ↓
Listmonk REST API (/api/bounces)
    ↓
Listmonk processes bounce (unsubscribe, log, etc.)
```

## Installation

See [DEPLOY.md](DEPLOY.md) for step-by-step instructions.

**Quick start:**
```bash
./build.sh                # Build binary
scp bounce-processor root@postmaster.defendsurviveprepare.com:/tmp/
ssh root@postmaster.defendsurviveprepare.com "bash -c '
sudo useradd -r -s /bin/false bounce-processor
sudo mkdir -p /opt/bounce-processor
sudo mv /tmp/bounce-processor /opt/bounce-processor/
sudo chown bounce-processor:bounce-processor /opt/bounce-processor/bounce-processor
sudo chmod +x /opt/bounce-processor/bounce-processor
'"
# Copy and edit service file, then:
# sudo systemctl enable bounce-processor
# sudo systemctl start bounce-processor
```

## Configuration

All configuration via environment variables (set in `/etc/systemd/system/bounce-processor.service`):

| Variable | Value |
|----------|-------|
| `POP3_HOST` | `mail.defendsurviveprepare.com` |
| `POP3_PORT` | `995` |
| `POP3_USER` | `bounces@defendsurviveprepare.com` |
| `POP3_PASS` | Mailcow password for bounces@ |
| `LISTMONK_URL` | `https://postmaster.defendsurviveprepare.com` |
| `LISTMONK_USER` | `pmapi` |
| `LISTMONK_PASS` | Your Listmonk API key |

## Monitoring

### View logs in real-time
```bash
sudo journalctl -u bounce-processor -f
```

### View bounce processor log file
```bash
sudo tail -f /var/log/bounce-processor.log
```

### Parse JSON logs
```bash
sudo tail -f /var/log/bounce-processor.log | jq .
```

### Count processed bounces
```bash
sudo grep "Bounce processed" /var/log/bounce-processor.log | wc -l
```

### Find errors
```bash
sudo grep "\"level\":\"ERROR\"" /var/log/bounce-processor.log
```

## Control (from your server)

### Enable/disable the service
```bash
# Start
sudo systemctl start bounce-processor

# Stop
sudo systemctl stop bounce-processor

# Restart
sudo systemctl restart bounce-processor

# Check status
sudo systemctl status bounce-processor

# View logs
sudo journalctl -u bounce-processor -f
```

## Error handling

The service:
- **Continues on error:** If one bounce fails to process, it continues with the next
- **Retries on connection loss:** Automatic restart via systemd
- **Logs everything:** All errors to stdout and `/var/log/bounce-processor.log`

## Bounce detection

The service detects bounces by parsing:
- **Delivery Status Notifications (DSNs):** RFC 3464 format (most common)
- **Final-Recipient header:** Email address that bounced
- **Action header:** "failed" = hard bounce

## Performance

- **Resource usage:** ~5MB RAM, minimal CPU (idles between scans)
- **Scan time:** <1 second for typical 10-50 bounces
- **Scan frequency:** Every 5 minutes (configurable)
- **Throughput:** ~100 bounces/minute (Listmonk API limited)

## Troubleshooting

**Service won't start:**
```bash
sudo journalctl -u bounce-processor -n 50
```

**POP3 connection fails:**
- Verify credentials in service file
- Test: `openssl s_client -connect mail.defendsurviveprepare.com:995`

**Listmonk API returns errors:**
- Check API key in service file
- Verify Listmonk is running
- Test: `curl -u pmapi:KEY https://postmaster.defendsurviveprepare.com/api/bounces`

**Bounces not being processed:**
- Check `/var/log/bounce-processor.log`
- Verify bounces are arriving: Check Mailcow web UI
- Verify Listmonk bounce settings: Settings → Bounces → Hard bounce action = "Unsubscribe"

## License

MIT

## Support

Check logs first: `sudo journalctl -u bounce-processor -f`
