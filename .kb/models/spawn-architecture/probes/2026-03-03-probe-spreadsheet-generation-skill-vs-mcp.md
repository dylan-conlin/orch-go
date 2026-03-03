# Probe: Spreadsheet Model Generation — Skill vs MCP Plugin Architecture

**Model:** spawn-architecture
**Date:** 2026-03-03
**Status:** Complete

---

## Question

The spawn-architecture model establishes that skills are instruction sets embedded at spawn time via SPAWN_CONTEXT.md. Skills tell agents *how* to do work, while MCP plugins provide *tool-level capabilities* at runtime. For spreadsheet model generation (Excel with live formulas, Google Sheets API), which architecture is correct: a skill that guides agents to use existing libraries, or an MCP plugin that provides spreadsheet-specific tools?

The model claims skills are "embedded at spawn time" and "agents don't load skills dynamically." This implies skills are procedural guidance, not runtime tooling. Does this distinction hold for spreadsheet generation, or does the boundary blur?

---

## What I Tested

### Test 1: Library Landscape — Can agents generate spreadsheets with existing tools?

**Tested openpyxl (Python, already installed):**
```bash
python3 << 'EOF'
import openpyxl
from openpyxl.chart import BarChart, Reference
wb = openpyxl.Workbook()
ws = wb.active; ws.title = "Inputs"
# ... [full test: 3 tabs, cross-tab formulas, charts, styling]
wb.save("/tmp/test_spreadsheet_model.xlsx")
EOF
```
**Result:** SUCCESS — Created 8,621 byte .xlsx with 3 tabs (Inputs, Calculations, Dashboard), 24 live formulas including cross-tab references (=Inputs!B2/(1-Inputs!E2)), bar chart, INDEX/MATCH, COUNTA, SUM/currency formatting.

**Tested xlsxwriter (Python, pip-installed):**
```bash
pip3 install xlsxwriter
python3 << 'EOF'
import xlsxwriter
wb = xlsxwriter.Workbook('/tmp/test_xlsxwriter_model.xlsx')
# ... [full test: 3 tabs, cross-tab formulas, column chart, styling]
wb.close()
EOF
```
**Result:** SUCCESS — Created 9,518 byte .xlsx with identical feature coverage. Zero dependencies, cleaner API for write-only workloads.

### Test 2: Google Sheets API capabilities

**Researched (not executed — requires GCP project + service account setup):**
- Google Sheets API v4 supports all needed operations via `batchUpdate` (67+ request types)
- Formulas work with `valueInputOption: "USER_ENTERED"` — any formula starting with `=` is interpreted as live
- Charts: `AddChartRequest` supports column, bar, line, pie, scatter
- Formatting: `RepeatCellRequest`, `UpdateBordersRequest`, currency/percent formats
- Auth: Service account (headless, no browser) is the right choice for agents
- Rate limits: 300 writes/min, each `batchUpdate` counts as 1 request regardless of sub-request count

### Test 3: Existing MCP servers and Claude Code skills

**Excel MCP servers found:**
| Server | Features | Chart Support |
|---|---|---|
| haris-musa/excel-mcp-server (Python/openpyxl) | Create, read, update, formulas, charts, pivot tables, formatting | Yes |
| negokaz/excel-mcp-server (Go/Excelize) | Read, write, formulas, create sheets | No |
| sbroenne/mcp-server-excel (COM API) | 23 tools, 214 ops, PivotTables, VBA | Yes (Windows only) |

**Google Sheets MCP servers found:**
| Server | Features | Chart/Format Support |
|---|---|---|
| xing5/mcp-google-sheets (19 tools) | CRUD, batch, share, search | No batchUpdate for charts/formatting |
| ringo380/claude-google-sheets-mcp | Read, write, append, clear | No |
| gmickel/sheets-cli (Claude Code Skill) | Read, write, append | No — values only |

**Key gap:** No existing MCP server for Google Sheets exposes `batchUpdate` with formatting, charts, or named ranges. All are limited to cell value operations.

### Test 4: Architecture decision — Skill vs MCP

**The decisive test:** Can a Claude Code agent produce a complete spreadsheet model using ONLY its existing tools (Bash + Write)?

Answer: **Yes, trivially.** The test scripts above were written inline and executed via Bash. An agent with skill guidance on spreadsheet model structure can:
1. Write a Python script using openpyxl/xlsxwriter (Write tool)
2. Execute it (Bash tool)
3. Produce a complete .xlsx with live formulas, charts, styling, multi-tab

No new runtime tools are needed. What's missing is **procedural knowledge**:
- How to structure a spreadsheet model (inputs → calculations → dashboard)
- When to use formulas vs static values
- Cross-tab reference patterns
- Chart selection and placement
- Stakeholder delivery patterns (self-contained models, not data dumps)

---

## What I Observed

### 1. The existing tool surface is sufficient

Claude Code agents already have everything needed:
- **Write tool** → create Python scripts
- **Bash tool** → run pip install + execute scripts
- **openpyxl** (already installed) → full Excel generation with formulas, charts, styling
- **xlsxwriter** (pip install) → zero-dependency alternative with slightly cleaner write-only API

For Google Sheets: agent writes a Python script using `google-api-python-client` with service account auth, calls `values.update` (USER_ENTERED for formulas) + `batchUpdate` (for charts/formatting). Same pattern: Write + Bash.

### 2. The bottleneck is knowledge, not tooling

Without guidance, agents produce:
- Static data dumps (hardcoded numbers instead of formulas)
- Single-tab flat exports (no model structure)
- Missing charts and formatting
- No stakeholder-ready presentation

With skill guidance, agents would know to:
- Separate inputs (editable by stakeholder) from calculations (formula-driven)
- Use cross-tab references (=Inputs!B2) so stakeholders can change assumptions
- Add summary dashboards with KPI formulas
- Include charts that reference formula-driven data
- Apply professional formatting (currency, percentages, headers)

### 3. MCP plugin would be over-engineering

An MCP plugin (wrapping openpyxl behind tool calls like `create_spreadsheet`, `add_formula`, `add_chart`) would:
- Add a service to maintain (MCP server process)
- Reduce agent flexibility (constrained to exposed tool surface)
- Not address the actual gap (knowledge of model structure)
- Duplicate what Bash+Write already provides

The one scenario where MCP adds value: **Google Sheets with pre-configured auth.** An MCP server with service account credentials baked in would save the agent from managing auth setup each time. But this is a deployment convenience, not an architectural necessity.

### 4. Library recommendations

| Use Case | Library | Why |
|---|---|---|
| Excel generation (default) | **openpyxl** | Already installed, read+write, full formula/chart/styling support |
| Excel generation (write-only, large datasets) | **xlsxwriter** | Zero deps, faster writes, excellent chart API |
| Google Sheets | **google-api-python-client** + service account | Official SDK, full batchUpdate access, headless auth |
| Node.js alternative (Excel) | **exceljs** | Best of weak options (charts experimental) |

### 5. Existing ecosystem gap

- **gmickel/sheets-cli** is the only Claude Code skill for Google Sheets — but it's limited to cell values (no formulas, formatting, or charts)
- **No Claude Code skill exists for Excel generation** with formulas/charts
- **Existing MCP servers for Google Sheets lack batchUpdate** for charts/formatting
- **haris-musa/excel-mcp-server** is the most complete Excel MCP but adds unnecessary indirection vs direct Python

---

## Model Impact

- [x] **Confirms** invariant: Skills are procedural guidance embedded at spawn time — spreadsheet model generation is a knowledge/guidance problem, not a tooling gap. The existing tool surface (Bash + Write) is sufficient; the agent needs procedural knowledge about model structure and formula patterns.

- [x] **Extends** model with: **The skill/MCP boundary test** — when an agent can already accomplish the task with existing tools but produces poor results, the answer is a skill (guidance), not an MCP plugin (new tools). MCP plugins are warranted when the existing tool surface literally cannot accomplish the task (e.g., no way to authenticate to a service, no way to interact with a running process). For spreadsheet generation, the tool surface is complete; only the procedural knowledge is missing.

---

## Notes

### Recommended skill structure

```
skills/src/worker/spreadsheet-model/.skillc/
├── skill.yaml
├── SKILL.md.template          # Core: model structure, formula patterns, delivery checklist
└── reference/
    ├── excel-patterns.md      # openpyxl/xlsxwriter code patterns for formulas, charts, styling
    ├── google-sheets.md       # Sheets API patterns (batchUpdate, USER_ENTERED, service account)
    └── model-templates.md     # Template structures: financial model, estimator, tracker, dashboard
```

### Key skill content (what the agent needs to know)

1. **Model structure pattern:** Inputs tab (editable assumptions) → Calculations tab (100% formulas) → Dashboard tab (KPI summary + charts)
2. **Formula-first rule:** Every calculated value must be a formula. If a stakeholder can't change an input and see downstream values update, it's not a model — it's a report.
3. **Cross-tab reference pattern:** Calculations tab references Inputs tab (=Inputs!B2). Dashboard references Calculations (=Calculations!E7). Never duplicate data.
4. **Delivery format:** .xlsx file in workspace for email/download, OR Google Sheet URL with sharing permissions set.
5. **Validation checklist:** No hardcoded numbers in Calculations tab, all charts reference formula-driven ranges, stakeholder can edit Inputs and see results change.

### When MCP makes sense (future)

If spreadsheet generation becomes a high-frequency operation across many projects, an MCP server for Google Sheets with pre-configured service account auth would reduce per-session setup friction. But this is a deployment optimization, not an architectural change — the skill remains the primary mechanism.

### Google Sheets auth prerequisite

Before an agent can create Google Sheets, a one-time setup is needed:
1. GCP project with Sheets + Drive APIs enabled
2. Service account created, JSON key stored at a known path (e.g., `~/.config/gcloud/sheets-service-account.json`)
3. The skill references this path; agents don't manage auth themselves

This setup is a Dylan task, not an agent task.
