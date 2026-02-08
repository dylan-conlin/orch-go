// Package daemon provides autonomous overnight processing capabilities.
package daemon

import "time"

// RecordFirstSeen records the first time an issue is seen in the triage queue.
// Returns true if this is the first time the issue was seen.
func (d *Daemon) RecordFirstSeen(id string) bool {
	if d.firstSeen == nil {
		d.firstSeen = make(map[string]time.Time)
	}
	if _, exists := d.firstSeen[id]; exists {
		return false
	}
	d.firstSeen[id] = time.Now()
	return true
}

// InGracePeriod returns true if an issue is still within the grace period.
// Records the first-seen time if this is the first time seeing this issue.
func (d *Daemon) InGracePeriod(id string) bool {
	if d.Config.GracePeriod <= 0 {
		return false
	}
	d.RecordFirstSeen(id)
	return time.Since(d.firstSeen[id]) < d.Config.GracePeriod
}

// InGracePeriodWithoutRecording checks grace-period status without mutating
// firstSeen state. Unseen issues are treated as in grace period.
func (d *Daemon) InGracePeriodWithoutRecording(id string) bool {
	if d.Config.GracePeriod <= 0 {
		return false
	}
	if d.firstSeen == nil {
		return true
	}
	firstSeen, exists := d.firstSeen[id]
	if !exists {
		return true
	}
	return time.Since(firstSeen) < d.Config.GracePeriod
}

// CleanFirstSeen removes entries from firstSeen that are no longer in the issue queue.
func (d *Daemon) CleanFirstSeen(activeIDs map[string]bool) {
	if d.firstSeen == nil {
		return
	}
	for id := range d.firstSeen {
		if !activeIDs[id] {
			delete(d.firstSeen, id)
		}
	}
}
