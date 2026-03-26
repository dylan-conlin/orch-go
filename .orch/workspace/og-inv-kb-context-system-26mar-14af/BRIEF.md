# Brief: orch-go-nid7l

## Frame

The question looked simple: where does the kb context in a worker prompt actually come from? But spawn has enough moving pieces that it is easy to blame the wrong stage - the kb query, the formatter, the workspace writer, or the prompt handoff - unless the whole path is traced end to end.

## Resolution

The important turn is that the worker never "discovers" kb context after startup. `orch spawn` does the work up front: it derives keywords from the task and sometimes the orientation frame, runs the kb query in the target project, formats the results into a markdown block, then threads that string through the spawn structs until `GenerateContext()` drops it into the `{{.KBContext}}` slot of the worker template. After that, `AtomicSpawnPhase1()` just writes the rendered file to disk and the minimal prompt points the worker at it.

That means the system is more linear than it first feels. If a worker gets bad prior knowledge, the likely failure is not some mysterious runtime injection step. It is almost always one of three places: keyword derivation, kb-result formatting, or the template render that writes `SPAWN_CONTEXT.md`. Once that clicked, the trace stopped feeling like a maze and started feeling like a simple pipeline.

## Tension

The open question is whether the current system gives enough observability when that pipeline goes wrong. The code is traceable, but the runtime experience still pushes you toward reading source instead of asking the system to show each stage directly.
