# Glass Browser Automation Setup

Glass requires Chrome with remote debugging enabled. Chrome enforces a security restriction: remote debugging only works with a non-default user data directory.

## Setup

1. **Create persistent debug profile directory:**
   ```bash
   mkdir -p ~/.chrome-debug-profile
   ```

2. **Launch Chrome with debugging:**
   ```bash
   "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
     --remote-debugging-port=9222 \
     --user-data-dir="$HOME/.chrome-debug-profile" &
   ```

3. **Verify debugging is active:**
   ```bash
   curl -s http://localhost:9222/json/version | jq '.Browser'
   ```

## Hotkey Integration (skhd)

Add to `.config/skhd/.skhdrc`:
```bash
# Launch Chrome with remote debugging (uses persistent debug profile for Glass)
default, focus < cmd + ctrl - e : pgrep -x "Google Chrome" > /dev/null && open -a /Applications/Google\ Chrome.app || "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --remote-debugging-port=9222 --user-data-dir="$HOME/.chrome-debug-profile" &
```

## Tradeoffs

- **Separate profile:** Extensions, bookmarks, and login sessions are separate from your main Chrome profile
- **Persistent:** The debug profile persists across restarts - set it up once
- **Security:** Remote debugging is only accessible from localhost

## Troubleshooting

**"DevTools remote debugging requires a non-default data directory"**
- Chrome won't enable debugging on the default profile
- Must use `--user-data-dir` pointing to a different location

**Port 9222 not responding**
- Chrome may have been running before launch with debugging
- Fully quit Chrome (`pkill -9 "Google Chrome"`), then relaunch

**Glass can't connect**
- Verify: `curl -s http://localhost:9222/json/version`
- Check: `lsof -i :9222` should show Chrome listening
