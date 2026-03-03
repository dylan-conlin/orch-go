# Codebase Audit: Performance

**TLDR:** Performance-focused audit identifying bottlenecks, algorithmic issues, inefficient queries, and optimization opportunities.

**Status:** STUB - To be fleshed out when needed

**When to use:** App is slow, high CPU/memory usage, scaling problems, response time issues

**Output:** Investigation file with performance findings, profiling data, and optimization recommendations with effort estimates

---

## Focus Areas (To be expanded)

1. **Algorithmic Complexity** - O(n²) loops, inefficient algorithms
2. **Database Queries** - N+1 queries, missing indexes, slow queries
3. **Resource Usage** - Memory leaks, excessive allocations
4. **I/O Operations** - Blocking I/O, unnecessary file reads
5. **Caching** - Missing caches, cache invalidation issues
6. **Concurrency** - Poor parallelization, lock contention

---

## Pattern Search Commands (To be expanded)

```bash
# Nested loops (potential O(n²))
rg "for.*:\s*\n.*for.*:" --type py -U

# N+1 query patterns
rg "\.all\(\)|\.filter\(" --type py -C 3 | rg "for.*in"

# Large files (potential complexity issues)
find . -name "*.py" -o -name "*.js" | xargs wc -l | sort -rn | head -20

# TODO/FIXME about performance
rg "TODO.*performance|FIXME.*slow|HACK.*optimize" -i

# Blocking I/O in loops
rg "for.*:\s*\n.*open\(|for.*:\s*\n.*requests\." --type py -U
```

---

*This skill stub establishes performance audit structure. Expand with profiling methodology, optimization patterns, and benchmarking when performance audit is needed.*
