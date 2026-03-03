# Codebase Audit: Architecture

**TLDR:** Architecture-focused audit identifying coupling issues, god objects, missing abstractions, and modularity problems.

**Status:** STUB - To be fleshed out when needed

**When to use:** Hard to add features, tight coupling between modules, unclear boundaries, refactoring needed

**Output:** Investigation file with architectural issues, dependency analysis, and refactoring effort estimates

---

## Focus Areas (To be expanded)

1. **God Objects** - Classes/modules doing too much
2. **Tight Coupling** - Modules depending on too many others
3. **Missing Abstractions** - Repeated patterns not extracted
4. **Circular Dependencies** - Modules importing each other
5. **Poor Modularity** - Unclear boundaries, leaky abstractions
6. **Violation of SOLID Principles** - SRP, OCP, LSP, ISP, DIP violations

---

## Pattern Search Commands (To be expanded)

```bash
# God classes (many methods)
rg "^\s+def \w+\(self" --type py | uniq -c | sort -rn | head -10

# Tight coupling (many imports from one module)
rg "^from (\w+) import" --type py | cut -d' ' -f2 | sort | uniq -c | sort -rn

# Large files (potential god objects)
find . -name "*.py" -o -name "*.js" | xargs wc -l | sort -rn | head -20

# Missing abstractions (switch/if-elif chains on type)
rg "if.*isinstance|if.*type\(.*\) ==" --type py -C 3

# Circular dependencies (imports at bottom of file)
rg "^from .* import" --type py | tail -20

# Deep nesting (complexity indicator)
rg "^\s{16,}(if|for|while|def)" --type py
```

---

*This skill stub establishes architecture audit structure. Expand with dependency analysis, refactoring patterns, and SOLID principles when architecture audit is needed.*
