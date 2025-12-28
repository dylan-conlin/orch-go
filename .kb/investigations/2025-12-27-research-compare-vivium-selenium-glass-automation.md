<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Cannot find public information about Vivium (by Selenium creator) or Glass browser automation tools - neither exists as publicly discoverable projects.

**Evidence:** Searched GitHub (0 results for "vivium browser automation"), checked Simon Stewart's repos (58 repos, none named Vivium), searched web for glass.dev (unreachable), Selenium blog (no mentions).

**Knowledge:** These may be private/unreleased tools, internal projects, or names may need clarification. Without public documentation or source code, cannot perform meaningful comparison.

**Next:** Escalate to orchestrator for clarification on tool names, URLs, or access to private documentation.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Compare Vivium (Selenium Creator) vs Glass Browser Automation

**Question:** What is Vivium's architecture and how does it differ from Glass's Chrome DevTools Protocol approach? What are the tradeoffs (reliability, speed, capabilities, AI-native features)? What lessons could Glass learn from Vivium's design?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Research agent
**Phase:** BLOCKED
**Next Step:** Awaiting clarification from orchestrator on tool names/URLs
**Status:** BLOCKED - Cannot locate public information

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Vivium Not Found on GitHub

**Evidence:** 
- GitHub search for "vivium browser automation" returned 0 results
- Simon Stewart's GitHub profile (shs96c) has 58 repositories, none named "Vivium"
- His repos include Selenium forks, webdriver-bidi-java, bazel-related projects, but no Vivium
- vivium.com and vivium.ai resolve to an unrelated German cookware company

**Source:** 
- https://github.com/search?q=vivium+browser+automation&type=repositories (0 results)
- https://github.com/shs96c?tab=repositories (58 repos, checked all)
- https://www.vivium.com (German cookware, not browser automation)

**Significance:** If Vivium exists as Simon Stewart's new browser automation project, it is not publicly available on GitHub or his personal repos.

---

### Finding 2: Glass Browser Automation Not Publicly Discoverable

**Evidence:**
- glass.dev domain unreachable (connection error)
- GitHub search for "glass browser automation" returned only 2 unrelated results:
  - fohara/red-glass: A Selenium observer tool from 2014 (12 stars, archived)
  - LalithSrinivasAS/Glassdoor_Jobs_Web_Scrapping_Tool: Job scraping tool
- Neither matches the described "Chrome DevTools Protocol approach" tool

**Source:**
- https://glass.dev (connection failed)
- https://github.com/search?q=glass+browser+automation&type=repositories (2 unrelated results)

**Significance:** Glass as a CDP-based browser automation tool is not publicly discoverable. May be a private project, unreleased, or name needs clarification.

---

### Finding 3: Simon Stewart's Current Focus

**Evidence:**
- Simon Stewart (shs96c) is the confirmed Selenium creator
- His personal website (rocketpoweredjetpants.com) lists him as leading the Selenium project
- Recent GitHub activity focuses on:
  - Selenium development (forked from SeleniumHQ/selenium)
  - Bazel build system contributions (rules_closure, rules_jvm, etc.)
  - webdriver-bidi-java: "Exploring ideas for WebDriver Bidi" (March 2025)
- No public announcements of "Vivium" found on his website or linked profiles

**Source:**
- https://www.rocketpoweredjetpants.com (Simon Stewart's personal site)
- https://github.com/shs96c/webdriver-bidi-java (WebDriver BiDi exploration)
- https://github.com/shs96c?tab=repositories (recent activity)

**Significance:** Simon Stewart's publicly visible work is on Selenium and WebDriver BiDi, not on a project called "Vivium". This suggests Vivium may be unreleased, internal, or the name may be incorrect.

---

## Synthesis

**Key Insights:**

1. **Neither tool is publicly discoverable** - Extensive searches across GitHub, official Selenium channels, and domain lookups failed to locate either Vivium or Glass as browser automation tools.

2. **Simon Stewart's public work focuses on Selenium/WebDriver BiDi** - His recent WebDriver BiDi Java exploration project may be related to what's being called "Vivium", but this is speculation without confirmation.

3. **Name clarification needed** - Both tools may exist under different names, be private/unreleased, or be internal projects at companies.

**Answer to Investigation Question:**

Cannot answer the comparison question as posed. Neither Vivium nor Glass browser automation tools are publicly accessible for research. The investigation is blocked pending clarification from the orchestrator on:
- Correct tool names
- URLs or access to private documentation
- Whether these are public or internal projects
- Alternative names or references to locate these tools

---

## Structured Uncertainty

**What's tested:**

- ✅ GitHub search for "vivium browser automation" (verified: 0 results on 2025-12-27)
- ✅ Simon Stewart's GitHub repos do not include "Vivium" (verified: checked 58 repos)
- ✅ glass.dev domain unreachable (verified: connection error)
- ✅ GitHub search for "glass browser automation" (verified: 2 unrelated results)

**What's untested:**

- ⚠️ Vivium may exist under a different name (not verified)
- ⚠️ Glass may be an internal/private project with different access (not verified)
- ⚠️ webdriver-bidi-java may be related to what's called "Vivium" (speculation only)
- ⚠️ Tools may be announced but not yet released (not verified via Twitter/Bluesky/conferences)

**What would change this:**

- Finding would be wrong if orchestrator provides correct URLs/names
- Finding would be wrong if tools are private and access needs to be granted
- Finding would be wrong if tools were recently announced at a conference not indexed yet

---

## Resolution Path

**Purpose:** Document what's needed to unblock this investigation.

### Required Clarifications

1. **Vivium details:**
   - Is "Vivium" the correct name?
   - Is it a public or private project?
   - URL, GitHub repo, or documentation access?
   - Is it related to Simon Stewart's webdriver-bidi-java exploration?

2. **Glass details:**
   - Is "Glass" the correct name for a CDP-based browser automation tool?
   - Is glass.dev the correct domain?
   - Alternative URLs or access mechanism?

3. **Context:**
   - Where did you hear about these tools? (conference, blog post, Twitter?)
   - Any specific source I can reference?

### Alternative Investigation Paths

If clarification unavailable, could investigate:

**Option A: WebDriver BiDi vs CDP comparison**
- Compare the emerging WebDriver BiDi standard vs Chrome DevTools Protocol
- This addresses the underlying technology question without specific tool names
- Simon Stewart is working on webdriver-bidi-java, which may be the "Vivium" reference

**Option B: Browser automation landscape review**
- Survey current browser automation approaches (Selenium, Playwright, Puppeteer, etc.)
- Compare CDP-based vs WebDriver-based approaches
- Document tradeoffs relevant to Glass-style architecture

**Option C: Wait for clarification**
- Keep investigation paused until correct tool names/URLs provided

---

## References

**URLs Searched:**
- https://github.com/search?q=vivium+browser+automation&type=repositories - 0 results
- https://github.com/shs96c - Simon Stewart's GitHub (58 repos, no Vivium)
- https://github.com/shs96c/webdriver-bidi-java - WebDriver BiDi exploration project
- https://github.com/SeleniumHQ - Selenium organization (no Vivium)
- https://www.selenium.dev/blog - Selenium blog (no Vivium mentions in 2025)
- https://www.rocketpoweredjetpants.com - Simon Stewart's personal site
- https://vivium.ai - Redirects to German cookware company (unrelated)
- https://vivium.com - German cookware company (unrelated)
- https://glass.dev - Connection failed

**Related Artifacts:**
- None - blocked investigation

---

## Investigation History

**2025-12-27:** Investigation started
- Initial question: Compare Vivium (by Selenium creator) vs Glass browser automation
- Context: Research request to understand architecture differences and tradeoffs

**2025-12-27:** Search phase completed
- Searched GitHub, Simon Stewart's repos, Selenium org, official blog
- Neither tool found publicly
- webdriver-bidi-java found as possible related project

**2025-12-27:** Investigation BLOCKED
- Status: BLOCKED - awaiting clarification
- Key outcome: Cannot proceed without correct tool names/URLs
