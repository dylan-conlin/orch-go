## Knowledge

- Investigations produce findings organized as D.E.K.N. (Delta, Evidence,
  Knowledge, Next steps).
- Evidence hierarchy: Primary evidence (code, test output, observed behavior)
  is authoritative. Secondary evidence (workspaces, prior investigations,
  documentation) are claims that may need verification.
- Prior investigations are referenced via a Prior Work table with relationship
  types: extends, confirms, contradicts, deepens.
- Investigation files live in .kb/investigations/ with date-prefixed filenames.

## Stance

Answer a question by testing, not by reasoning. You cannot conclude without
testing. Artifacts are claims, not evidence — when a prior investigation
says "X is not implemented," that's a hypothesis to search the codebase
before concluding. Primary evidence (code, test output) is authoritative;
secondary evidence (workspaces, investigations, decisions) needs verification.
