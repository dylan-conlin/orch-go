# Brief: orch-go-wgkj4

## Frame

You accepted the decision that thread/comprehension is the product and execution is substrate. But what does that look like in the actual codebase? I went through every package and counted, expecting rough parity — maybe 40/60 core-to-substrate. The number I found was 16/72. For every line of comprehension code, there are nearly five lines of plumbing. The thread package — the conceptual spine of the product you just named — is seventeen times smaller than the daemon.

## Resolution

The inventory classifies 55 packages, 100+ command files, 5 web routes, and the skills directory into core, substrate, and adjacent. Three packages alone — daemon (39K), spawn (30K), verify (23K) — contain 48% of all package code, all substrate. The core layer is compact: thread, claims, kbmetrics, kbgate, tree, and a handful of completion/review pieces totaling about 47K lines with their command files.

The ratio isn't wrong in itself. Plumbing is necessary, and some of it was the scaffolding that revealed the stronger product thesis. But there are two findings that are more interesting than the ratio. First, the web UI's default page is a 32KB execution dashboard — a user's first impression contradicts the product decision before they encounter any comprehension surface. Second, some packages straddle the boundary in important ways: verify contains both mechanical completion checks (substrate) and probe-model-merge / consequence sensors / confidence gates (core). Orient leads with thread state. Attention tracks knowledge decay. These bridge packages are the connective tissue between execution and comprehension, and they're where the layers talk to each other.

## Tension

The 16/72 ratio tells you where the code mass is, but it doesn't tell you where the value mass is. The thread package at 2,300 lines may be doing more work per line than the daemon at 39,000. Do you want to change the ratio (move investment toward core, shrink substrate), or do you want to keep the ratio and just change the front door (make the UI and positioning match the decision)? Those are different projects. The front door change is a weekend. The ratio change is a quarter.
