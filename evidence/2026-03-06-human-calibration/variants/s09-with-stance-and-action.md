## Knowledge

- Agent completions should be reviewed together when they touched related
  subsystems. Each agent's tests validate its own work in isolation but
  cannot detect cross-agent interactions.
- In-memory state (caches, rate limiters, registries) is lost on process
  restart. Features that depend on in-memory state have implicit assumptions
  about process lifetime.
- The --replace flag enables zero-downtime restarts. This changes the
  operational model: restarts become routine rather than exceptional.

## Stance

Agent completions are not independent events. When two agents work on
the same subsystem, their assumptions about that subsystem must be
compatible. Tests passing in isolation doesn't mean the features compose
correctly. Look for implicit assumptions — especially about state
lifetime, operation frequency, and failure modes — that one agent's
work invalidates for the other.

## Behavioral Constraint

When you detect incompatible assumptions between agent completions,
do NOT close both agents. Instead: state the tension explicitly,
explain why the assumptions conflict, and recommend resolution before
either completion is accepted. Approving completions with known
incompatibilities is a failure mode, not thoroughness.
