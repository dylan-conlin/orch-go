<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** SendCutSend has 6 official paid holidays: New Year's Day, Memorial Day, Independence Day, Labor Day, Thanksgiving Day, and Christmas Day.

**Evidence:** Extracted text from "SendCutSend Employee Handbook 2024.docx (1).pdf" using pdftotext (poppler), found Holidays section on page 14.

**Knowledge:** Full-time employees get paid time-off for these holidays; 1.5x pay for regular shift hours worked on holidays.

**Next:** Close - information extracted successfully.

---

# Investigation: Extract SCS Holidays PDF Task

**Question:** What are the official SendCutSend holidays/days off for 2024?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Agent (og-inv-extract-scs-holidays-30dec)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: PDF extraction successful using pdftotext

**Evidence:** Used `/opt/homebrew/bin/pdftotext` from poppler package to extract text from the PDF. Text extraction was clean and readable.

**Source:** `/Users/dylanconlin/Downloads/SendCutSend Employee Handbook 2024.docx (1).pdf`

**Significance:** Confirms pdftotext is an effective tool for extracting text from PDF documents for holiday/policy searches.

---

### Finding 2: SendCutSend Official Holidays (6 total)

**Evidence:** From the Holidays section on page 14 of the Employee Handbook:

```
Holidays
We are happy to show our appreciation to you by providing paid time off for holidays recognized by the
Company. If you are a full-time employee, you may be eligible for paid time-off for the following holidays:
New Year's Day
Memorial Day
Independence Day
Labor Day
Thanksgiving Day
Christmas Day
```

**Source:** SendCutSend Employee Handbook 2024, Page 14, Holidays section

**Significance:** These are the 6 official company-recognized holidays that full-time employees receive paid time off for.

---

### Finding 3: Holiday pay policies

**Evidence:** Additional holiday policy details from the handbook:

- Holiday pay is computed at base rate of pay for normally scheduled hours
- Holiday pay is NOT considered as time worked for overtime calculations
- Employees receive 1.5x pay for all regular shift hours worked on holidays
- Once you volunteer to work a holiday, you're scheduled and subject to attendance policy
- Employees on leaves of absence or suspension do NOT receive holiday pay

**Source:** SendCutSend Employee Handbook 2024, Page 14, Holidays section

**Significance:** Important compensation details for understanding the holiday policy.

---

## Synthesis

**Key Insights:**

1. **Standard US holidays** - SCS recognizes 6 major US federal holidays, which is a fairly standard set for most companies.

2. **Premium pay for holiday work** - The 1.5x pay rate for working on holidays provides clear compensation incentive structure.

3. **Eligibility restrictions** - Full-time employee status is required; employees on leave don't receive holiday pay.

**Answer to Investigation Question:**

SendCutSend officially recognizes the following 6 paid holidays for 2024:
1. New Year's Day
2. Memorial Day
3. Independence Day
4. Labor Day
5. Thanksgiving Day
6. Christmas Day

Full-time employees receive paid time off for these holidays at their base rate. If they work on a holiday, they receive 1.5x pay.

---

## Structured Uncertainty

**What's tested:**

- ✅ PDF text extraction works (verified: pdftotext command produced readable text)
- ✅ Holidays section exists on page 14 (verified: grep found the section)
- ✅ 6 holidays listed in handbook (verified: extracted and listed above)

**What's untested:**

- ⚠️ 2024-specific dates (handbook doesn't list specific calendar dates)
- ⚠️ Floating holidays or additional office closures (not mentioned in this section)
- ⚠️ Part-time employee holiday eligibility (only mentions full-time)

**What would change this:**

- If there's a separate 2024 holiday calendar document with specific dates
- If there are company-wide email announcements about additional closures
- If state-specific addendums add additional holidays (handbook has state addendums but I focused on the main holidays section)

---

## Implementation Recommendations

**Purpose:** N/A - This is an information extraction task, not an implementation task.

### Recommended Approach ⭐

**Use extracted holiday list** - The 6 holidays listed above are the official SCS holidays.

**For 2024 specific dates:**
1. New Year's Day - January 1, 2024 (Monday)
2. Memorial Day - May 27, 2024 (Monday)
3. Independence Day - July 4, 2024 (Thursday)
4. Labor Day - September 2, 2024 (Monday)
5. Thanksgiving Day - November 28, 2024 (Thursday)
6. Christmas Day - December 25, 2024 (Wednesday)

---

## References

**Files Examined:**
- `/Users/dylanconlin/Downloads/SendCutSend Employee Handbook 2024.docx (1).pdf` - Full employee handbook

**Commands Run:**
```bash
# Install poppler for PDF text extraction
/opt/homebrew/bin/brew install poppler

# Extract text from PDF
/opt/homebrew/bin/pdftotext "/Users/dylanconlin/Downloads/SendCutSend Employee Handbook 2024.docx (1).pdf" -

# Search for holidays section
pdftotext ... | grep -A 50 -i "^Holidays$"
```

**External Documentation:**
- N/A

**Related Artifacts:**
- N/A

---

## Investigation History

**2025-12-30 10:XX:** Investigation started
- Initial question: What are the official SCS holidays for 2024?
- Context: Need to extract holiday information from the employee handbook PDF

**2025-12-30 10:XX:** PDF extraction successful
- Used pdftotext (poppler) to extract text
- Found Holidays section on page 14

**2025-12-30 10:XX:** Investigation completed
- Status: Complete
- Key outcome: Identified 6 official SCS holidays with pay policies

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED
