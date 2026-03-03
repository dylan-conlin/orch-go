# Codebase Audit: Security

**TLDR:** Security-focused audit identifying vulnerabilities, unsafe patterns, secrets exposure, and OWASP compliance gaps.

**Status:** STUB - To be fleshed out when needed

**When to use:** Security review needed, penetration test prep, compliance audit, incident investigation

**Output:** Investigation file with security findings rated by severity (Critical/High/Medium/Low) with remediation steps

---

## Focus Areas (To be expanded)

1. **Secrets Exposure** - API keys, passwords, tokens in code/git history
2. **Injection Vulnerabilities** - SQL injection, command injection, XSS
3. **Authentication/Authorization** - Weak auth, missing access controls
4. **Cryptography** - Weak encryption, insecure random, poor key management
5. **Dependencies** - Known vulnerabilities in packages
6. **Input Validation** - Unsafe user input handling
7. **OWASP Top 10** - Compliance with OWASP security standards

---

## Pattern Search Commands (To be expanded)

```bash
# Secrets exposure
rg "password|secret|api_key|token|private_key" --type py --type js -i

# SQL injection
rg "execute\(.*%|\.format\(|f\".*FROM|f\".*WHERE" --type py

# Command injection
rg "subprocess\.call|os\.system|eval\(|exec\(" --type py

# XSS vulnerabilities
rg "innerHTML|dangerouslySetInnerHTML|\.html\(" --type js --type jsx

# Hardcoded credentials
rg "password\s*=\s*['\"]|api_key\s*=\s*['\"]" --type py --type js
```

---

*This skill stub establishes security audit structure. Expand with detailed workflow, severity ratings, and remediation patterns when security audit is needed.*
