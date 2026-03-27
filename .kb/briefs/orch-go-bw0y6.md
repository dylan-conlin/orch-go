# Brief: orch-go-bw0y6

## Frame

The architecture guide was stuck in January. It opened with a spawn diagram, listed packages alphabetically, and spent its whole length on execution mechanics — backends, daemon plumbing, tmux vs headless. A new reader would conclude this project is an orchestration CLI. After the product-boundary decision, that's not just incomplete; it actively reproduces the wrong identity.

## Resolution

The rewrite opens with what orch-go actually is: a coordination and comprehension layer that makes agent output compound into understanding. Then it shows three boundary tables — core, substrate, adjacent — each with the concerns, what they do, and which packages implement them. The system diagram became a three-layer stack (core → substrate → external). The directory listing got layer annotations so an engineer can see whether a package is product-defining or execution plumbing at a glance.

The pleasant surprise was how clean the mapping is. I expected borderline cases, but the package structure already reflects the boundary — `thread/`, `claims/`, `completion/` on one side; `spawn/`, `tmux/`, `opencode/` on the other. The codebase already knew the answer; the guide just hadn't asked the question. All execution detail was preserved verbatim under "Execution Substrate Detail" — the point isn't to hide the plumbing, it's to stop letting the plumbing introduce the house.

## Tension

Two packages — `pkg/attention/` and `pkg/orient/` — sit at the boundary. They're daemon mechanics (substrate), but their purpose is routing toward comprehension (core). I classified them as substrate because that's where they run, but if the daemon evolves toward being a comprehension agent rather than an execution manager, they may belong in core. Worth watching as the thread-first surfaces develop.
