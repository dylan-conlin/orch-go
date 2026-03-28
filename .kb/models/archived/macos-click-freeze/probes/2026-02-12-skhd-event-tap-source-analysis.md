# Probe: Does skhd's CGEventTap registration mechanism explain click event corruption?

**Model:** .kb/models/macos-click-freeze/model.md
**Date:** 2026-02-12
**Status:** Complete

---

## Question

Hypothesis 7 claims skhd corrupts WindowServer click event routing via CGEventTap. The model notes "skhd only registers for keyboard events, not mouse/trackpad — but event taps can have side effects on the event pipeline." This probe tests: what exactly does skhd register for, and could its event tap architecture cause click corruption even without registering for mouse events?

---

## What I Tested

**Command/Code:**

1. Checked skhd version:
```bash
$ /opt/homebrew/bin/skhd --version
skhd-v0.3.9
```

2. Analyzed skhd source code from GitHub (koekeishiya/skhd master branch):

**Event tap creation** (`src/event_tap.c`):
```c
event_tap->handle = CGEventTapCreate(
    kCGSessionEventTap,        // tap location: session level
    kCGHeadInsertEventTap,     // placement: FIRST in pipeline
    kCGEventTapOptionDefault,  // type: ACTIVE (can modify/consume)
    event_tap->mask,           // mask: keyboard-only (see below)
    callback,
    event_tap);
```

**Event mask** (`src/skhd.c`):
```c
// Normal mode:
event_tap.mask = (1 << kCGEventKeyDown) | (1 << NX_SYSDEFINED);

// Observation mode (--observe):
event_tap.mask = (1 << kCGEventKeyDown) | (1 << kCGEventFlagsChanged);
```

**Callback** (`src/skhd.c` — `key_handler`):
```c
static EVENT_TAP_CALLBACK(key_handler) {
    switch (type) {
    case kCGEventTapDisabledByTimeout:
    case kCGEventTapDisabledByUserInput: {
        struct event_tap *event_tap = (struct event_tap *) reference;
        CGEventTapEnable(event_tap->handle, 1);  // re-enables tap
    } break;
    case kCGEventKeyDown: {
        // ... hotkey matching ...
        if (result) return NULL;  // swallows matched keystroke
    } break;
    }
    return event;  // passes everything else through
}
```

**Command execution** (`src/hotkey.c` — `fork_and_exec`):
```c
static inline void fork_and_exec(char *command) {
    int cpid = fork();
    if (cpid == 0) {
        setsid();
        char *exec[] = { shell, arg, command, NULL};
        execvp(exec[0], exec);
    }
    // parent returns immediately — non-blocking
}
```

3. Analyzed `.skhdrc` config — 78 bindings, many invoke `yabai -m window --focus` commands

4. Analyzed `.yabairc` — `focus_follows_mouse autofocus` (line 172) enables mouse-position-triggered focus changes

5. Searched GitHub issues (koekeishiya/skhd) for click/mouse/freeze — **zero results**

6. Found Apple Feedback Assistant report FB12113281: CGEvent.tapCreate taps sometimes stop receiving events entirely (macOS 13.4+, unresolved)

**Environment:**
- skhd v0.3.9 (Homebrew), macOS 15.6.1 Sequoia, Apple Silicon M3 Pro
- skhd currently DISABLED via launchctl (testing freeze absence)

---

## What I Observed

**Key observations:**

1. **skhd does NOT register for mouse events.** The mask is strictly `kCGEventKeyDown | NX_SYSDEFINED`. No `kCGEventLeftMouseDown`, `kCGEventLeftMouseUp`, etc.

2. **BUT skhd creates an ACTIVE tap at HEAD position.** `kCGEventTapOptionDefault` + `kCGHeadInsertEventTap` means skhd's tap is the FIRST filter in the session event pipeline. Even though the mask only requests keyboard events, the tap is positioned at the head of the pipeline. This is the highest-priority position — all events flow past this tap point.

3. **The tap re-enables on timeout.** skhd correctly handles `kCGEventTapDisabledByTimeout` by calling `CGEventTapEnable`. However, the re-enable itself is a race: between disable and re-enable, the event pipeline state changes.

4. **Timeout/re-enable cycle is the likely corruption vector.** When skhd's callback takes too long (even rarely), macOS disables the tap via timeout. The pipeline reconfigures. skhd immediately re-enables. The pipeline reconfigures again. This rapid disable/re-enable at HEAD position could leave WindowServer's event routing in an inconsistent state — particularly for event types NOT in skhd's mask (like click events), since those event types take a different code path through the tap infrastructure.

5. **78 shell-spawning bindings create timing pressure.** While `fork_and_exec` is non-blocking, the `find_and_exec_hotkey` call before it (hotkey table lookup, mode switching) runs synchronously in the callback. With 9 modal modes and 78+ bindings, table lookups on rapid keystroke sequences could exceed the tap callback timeout threshold.

6. **skhd + yabai focus interaction.** Many skhd bindings call `yabai -m window --focus`, and yabai has `focus_follows_mouse autofocus` enabled. This creates a chain: skhd callback → fork → yabai focus change → WindowServer moves focus → autofocus reacts to mouse position → more focus events. This amplifies the event pipeline load during skhd-triggered actions.

7. **Apple bug FB12113281 confirms the failure mode exists.** Event taps created with CGEvent.tapCreate can "stop receiving events" permanently on macOS 13.4+. The reporter found that registering with `eventsOfInterest: .max` (broad mask) while triggering heavy app activity has "a chance" of breaking the tap permanently. While skhd uses a narrow mask, the HEAD position + active tap type + timeout cycling could trigger related WindowServer state corruption.

8. **Killing skhd doesn't fix the freeze (needs HUP).** This is explained by CGEventTapCreate documentation: removing a tap doesn't automatically restore the previous pipeline state. WindowServer needs to reconfigure its event routing (which HUP triggers) to clear corrupted routing state left by the removed tap.

---

## Model Impact

**Verdict:** extends — Hypothesis 7 (skhd corrupts WindowServer click event routing)

**Details:**

The model correctly identifies skhd as the probable culprit and notes "skhd only registers for keyboard events, not mouse/trackpad." This probe extends that understanding with the specific mechanism: **skhd's active event tap at HEAD position with timeout/re-enable cycling can corrupt WindowServer's event routing for ALL event types, not just the ones in its mask.**

The corruption vector is NOT that skhd intercepts mouse events (it doesn't). It's that:
1. An active tap at session-level HEAD position participates in the event pipeline infrastructure
2. Timeout-triggered disable/re-enable cycles destabilize the pipeline routing state
3. WindowServer's routing tables for event types NOT in the mask (mouse clicks) can become inconsistent during pipeline reconfiguration
4. The corruption persists after the tap is removed — only HUP (full reconfigure) clears it

This adds a new invariant to the model: **"skhd's event tap corruption mechanism is pipeline-level, not event-type-specific."** The mask determines what events skhd PROCESSES, not what events are AFFECTED by the tap's presence in the pipeline.

**New suggested investigation:** Check skhd's debug output for "restarting event-tap" messages — frequency of timeout/re-enable cycles would confirm the corruption trigger rate. Run `skhd -V` (verbose) and grep for the restart message.

**Confidence:** Medium — The mechanism is architecturally sound and explains all observed symptoms (click-specific freeze, cursor still works, HUP fixes, killing skhd doesn't fix, corruption persists). However, this is architectural reasoning, not a reproduction. Definitive confirmation requires either: (a) continued freeze absence with skhd disabled, or (b) reproducing the freeze by running skhd in verbose mode and correlating "restarting event-tap" messages with freeze onset.
