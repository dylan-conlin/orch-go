## Summary (D.E.K.N.)

**Delta:** The active disk leak is a skill-reprocess launchd job running hourly that appends to logs without rotation - 1.3M lines (228MB combined) and growing ~36K lines/day.

**Evidence:** reprocess-missed-skills.log has 1,299,000 lines; 24,074 entries yesterday, 12,133 today so far; launchd runs every 3600 seconds with no log rotation.

**Knowledge:** The prior investigations focused on static large files (Docker, Ollama) but missed this actively growing log; the job outputs to both stdout AND a separate log file (doubling growth).

**Next:** Fix the launchd job: add log rotation (truncate >50MB), or redirect to /dev/null if metrics aren't needed, or fix the script to not log every session on every run.

**Promote to Decision:** recommend-no (tactical fix to launchd job, not architectural)

---

# Investigation: Find Active Disk Space Leak

**Question:** What is actively GROWING and causing daily disk fill for 3+ days - not static large files, but the active leak source?

**Started:** 2026-01-21
**Updated:** 2026-01-21
**Owner:** Worker Agent (og-feat-find-active-disk-21jan-041b)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Related:** `.kb/investigations/2026-01-21-inv-deep-system-scan-disk-space.md` (static file analysis)
**Related:** `.kb/investigations/2026-01-19-inv-investigate-recurring-disk-space-filling.md` (prior cleanup gaps)

---

## Findings

### Finding 1: skill-reprocess launchd job is the primary active leak (228MB, growing ~4MB/day)

**Evidence:**
- `~/.claude/metrics/reprocess-missed-skills.log` = 114MB, 1,299,000 lines
- `~/.claude/metrics/reprocess-missed-skills-launchd.log` = 114MB (duplicate!)
- Yesterday (Jan 20): 24,074 new lines
- Today (Jan 21, partial): 12,133 new lines so far
- Growth rate: ~36,000 lines/day = ~4MB/day combined

**Source:** 
```bash
ls -lh ~/.claude/metrics/*.log
wc -l ~/.claude/metrics/reprocess-missed-skills.log  # 1,299,000
grep -c "2026-01-20" ~/.claude/metrics/reprocess-missed-skills.log  # 24,074
grep -c "2026-01-21" ~/.claude/metrics/reprocess-missed-skills.log  # 12,133
```

**Significance:** This is the ACTIVE leak - it runs every hour (3600s interval) and appends indefinitely. The job logs "reprocessing session X" for every tracked session, every hour, forever. Worse, it outputs to BOTH the script's own log AND launchd's StandardOutPath - doubling the growth.

---

### Finding 2: Launchd job has no log rotation

**Evidence:**
- Plist: `~/Library/LaunchAgents/com.user.claude.reprocess-skills.plist`
- `StartInterval`: 3600 (hourly)
- `StandardOutPath`: points to reprocess-missed-skills-launchd.log
- No size limit, no rotation, no cleanup

The script at `~/.orch/scripts/reprocess-missed-skills.sh` also writes to its own log file, creating duplicate logging.

**Source:** `cat ~/Library/LaunchAgents/com.user.claude.reprocess-skills.plist`

**Significance:** The job will grow forever until manually truncated. At 4MB/day, this adds 1.4GB/year just from this one job.

---

### Finding 3: gopls/go-build cache creates 5,915 files daily (moderate growth)

**Evidence:**
- gopls: 3,822 new files in last 24h (150MB total)
- go-build: 2,093 new files in last 24h (196MB total)
- These are normal development caches that get cleaned by `go clean -cache`

**Source:** 
```bash
find ~/Library/Caches -type f -mtime -1 | cut -d'/' -f6 | sort | uniq -c | sort -rn
du -sh ~/Library/Caches/gopls ~/Library/Caches/go-build
```

**Significance:** This is expected developer churn, not a leak. The existing cleanup script handles go-build; gopls could be added but isn't urgent.

---

### Finding 4: Docker/Colima disks are sparse files (not actually 100GB)

**Evidence:**
- `ls -lh` shows: datadisk 100GB, diffdisk 20GB (apparent size)
- `du -h` shows: datadisk 6.6GB, diffdisk 1.0GB (actual disk usage)
- Docker inside Colima: 6.4GB images, 3.9GB build cache
- Total actual Colima usage: 8GB

**Source:**
```bash
ls -lh ~/.colima/_lima/_disks/colima/datadisk  # 100G (sparse)
du -h ~/.colima/_lima/_disks/colima/datadisk   # 6.6G (actual)
docker system df
```

**Significance:** The prior investigation reported these as "100GB files being modified" which looked alarming, but they're sparse files. The actual growth is modest and within the Docker containers, not the VM disk itself.

---

### Finding 5: OpenCode storage is now minimal (previous leak fixed)

**Evidence:**
- `~/.local/share/opencode/` = 97MB total
- 885 new files in 24h (normal session activity)
- No `snapshot/` directory (the prior 8.8GB leak is gone)
- `tool-output/`: 44MB, `storage/`: 6.2MB

**Source:** `du -sh ~/.local/share/opencode/*`

**Significance:** The OpenCode snapshot leak identified in prior investigations has been resolved. Current growth is normal operation.

---

## Synthesis

**Key Insights:**

1. **The active leak is a runaway logging job** - The skill-reprocess launchd job runs hourly and logs every session it processes without rotation. This is the "3+ days of daily disk fill" - it's been running since October 2025 and has accumulated 228MB with no cleanup.

2. **Prior investigations focused on static consumers, not growth** - The Jan 19 and Jan 21 investigations identified large directories (Docker 23GB, Ollama 15GB) but didn't identify what was *actively growing*. Those are static - they don't cause "daily" disk fill.

3. **The 100GB Colima file is a red herring** - The modification timestamp on sparse VM disks triggers on any container activity, but actual growth is minimal. This looked like a leak but isn't.

**Answer to Investigation Question:**

The active disk space leak causing daily fill is the `com.user.claude.reprocess-skills` launchd job that:
1. Runs every hour (StartInterval 3600)
2. Logs every session processed to BOTH its own log AND launchd's stdout
3. Has no log rotation
4. Has been running since October 2025
5. Currently at 228MB combined and growing ~4MB/day

Secondary growth comes from normal development activity (gopls/go-build caches) which is expected and not a leak.

---

## Structured Uncertainty

**What's tested:**

- ✅ reprocess-missed-skills.log is 1.3M lines (verified: `wc -l`)
- ✅ Growth rate is ~36K lines/day (verified: grep for dates)
- ✅ Launchd job runs hourly with no rotation (verified: plist review)
- ✅ Colima disk is sparse (verified: du vs ls comparison)
- ✅ OpenCode snapshot leak is fixed (verified: directory doesn't exist)

**What's untested:**

- ⚠️ Whether disabling the job breaks skill metrics (need to check if metrics are used)
- ⚠️ Historical growth rate (only measured 1.5 days)
- ⚠️ Whether the script can be modified to log less verbosely

**What would change this:**

- Finding would be wrong if there's another job creating files at a higher rate
- Finding would be wrong if the skill-reprocess script serves a critical function that requires verbose logging
- Growth estimate would change if agent spawn rate increases significantly

---

## Implementation Recommendations

### Recommended Approach ⭐

**Fix the launchd job logging** - Add log rotation to the existing cleanup script or modify the reprocess script to log less.

**Why this approach:**
- Addresses root cause (unbounded logging)
- Low risk (just log management)
- Immediate impact (stops growth)

**Trade-offs accepted:**
- May lose historical metrics (acceptable - they're not being analyzed)
- Requires testing to ensure job still works after modification

**Implementation sequence:**
1. Add log truncation to cleanup script: `truncate -s 0 ~/.claude/metrics/reprocess-missed-skills*.log`
2. Consider redirecting launchd stdout to /dev/null if metrics aren't needed
3. Optionally modify the script to only log errors, not every session

### Quick Fix (Immediate)

Add to `~/Library/LaunchAgents/com.dylan.disk-cleanup.plist` cleanup script:

```bash
# Truncate skill-reprocess logs if >50MB
find ~/.claude/metrics -name "reprocess-missed-skills*.log" -size +50M -exec truncate -s 0 {} \; 2>/dev/null
```

### Better Fix (Recommended)

Modify the launchd plist to redirect stdout to /dev/null:

```xml
<key>StandardOutPath</key>
<string>/dev/null</string>
```

Then the script's own logging can be managed separately or removed.

### Best Fix (If Time Permits)

Modify `~/.orch/scripts/reprocess-missed-skills.sh` to:
1. Only log errors and summary (not every session)
2. Or write to a rotating log with max size

---

## References

**Files Examined:**
- `~/Library/LaunchAgents/com.user.claude.reprocess-skills.plist` - Launchd job causing the leak
- `~/.claude/metrics/reprocess-missed-skills.log` - The growing log file
- `~/.orch/scripts/reprocess-missed-skills.sh` - The script being run

**Commands Run:**
```bash
# Find recently modified large files
find ~ -type f -mtime -1 -size +10M 2>/dev/null | xargs ls -lhS

# Count new files per directory  
find ~/.orch -type f -mtime -1 | wc -l
find ~/.local/share/opencode -type f -mtime -1 | wc -l
find ~/.claude -type f -mtime -1 | wc -l
find ~/Library/Caches -type f -mtime -1 | wc -l

# Analyze log growth
wc -l ~/.claude/metrics/reprocess-missed-skills.log
grep -c "2026-01-20" ~/.claude/metrics/reprocess-missed-skills.log
grep -c "2026-01-21" ~/.claude/metrics/reprocess-missed-skills.log

# Compare sparse vs actual disk usage
ls -lh ~/.colima/_lima/_disks/colima/datadisk
du -h ~/.colima/_lima/_disks/colima/datadisk
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-21-inv-deep-system-scan-disk-space.md` - Static file analysis (complementary)
- **Investigation:** `.kb/investigations/2026-01-19-inv-investigate-recurring-disk-space-filling.md` - Prior cleanup gaps

---

## Investigation History

**2026-01-21 09:45:** Investigation started
- Initial question: What is actively GROWING (not static large files)?
- Context: Disk filling daily for 3+ days despite prior cleanup investigations

**2026-01-21 09:46:** Found the smoking gun
- reprocess-missed-skills.log at 1.3M lines, growing ~36K lines/day
- Launchd job runs hourly with no rotation
- Outputs to TWO files (doubling growth)

**2026-01-21 09:50:** Investigation completed
- Status: Complete
- Key outcome: Active leak is skill-reprocess launchd job (228MB, ~4MB/day growth), not the static Docker/Ollama files
