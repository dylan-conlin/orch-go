#!/bin/bash
# Archive old completed investigations
# Usage: ./scripts/archive-old-investigations.sh [--dry-run]

set -e

DRY_RUN=false
if [ "$1" = "--dry-run" ]; then
	DRY_RUN=true
	echo "=== DRY RUN MODE - No files will be moved ==="
	echo ""
fi

# Ensure we're in project root
cd "$(dirname "$0")/.."

# Create archived directory if it doesn't exist
ARCHIVE_DIR=".kb/investigations/archived"
if [ "$DRY_RUN" = false ]; then
	mkdir -p "$ARCHIVE_DIR"
fi

# Calculate cutoff date (30 days ago)
CUTOFF_DATE=$(date -v-30d +%Y-%m-%d 2>/dev/null || date -d "30 days ago" +%Y-%m-%d)
TODAY=$(date +%Y-%m-%d)
FEB_2026="2026-02"

echo "Today: $TODAY"
echo "Cutoff date (30 days ago): $CUTOFF_DATE"
echo "Archive directory: $ARCHIVE_DIR"
echo ""

# Count files before archival
TOTAL_BEFORE=$(ls -1 .kb/investigations/*.md 2>/dev/null | wc -l | tr -d ' ')
ARCHIVED_BEFORE=$(ls -1 "$ARCHIVE_DIR"/*.md 2>/dev/null | wc -l | tr -d ' ') || ARCHIVED_BEFORE=0

echo "Files before archival:"
echo "  Active: $TOTAL_BEFORE"
echo "  Archived: $ARCHIVED_BEFORE"
echo ""

# Specific files to archive (with status updates needed)
SPECIFIC_FILES=(
	"2025-12-25-inv-pattern-tool-relationship-shareability.md|exploratory, 44 days stale"
	"2026-01-06-inv-diagnose-investigation-skill-32-completion.md|superseded by agent-completion-lifecycle-separation decision"
	"2026-01-09-inv-anthropic-oauth-community-workarounds.md|already decided: Claude Max"
	"2026-01-22-audit-feature-impl-skill-constitutional-constraints.md|already promoted to decision"
	"2026-01-22-philosophical-claude-constitution-vs-multi-agent-orchestration.md|already promoted to decision"
)

echo "=== Step 1: Archive 5 specific stale investigations ==="
echo ""

SPECIFIC_COUNT=0
FILES_TO_MOVE=()

for entry in "${SPECIFIC_FILES[@]}"; do
	IFS='|' read -r filename reason <<<"$entry"
	filepath=".kb/investigations/$filename"

	if [ -f "$filepath" ]; then
		echo "Archiving: $filename"
		echo "  Reason: $reason"

		if [ "$DRY_RUN" = false ]; then
			# Update status in file before moving
			# Use perl for cross-platform sed compatibility
			perl -i -pe 's/\*\*Status:\*\* In Progress/\*\*Status:\*\* Complete (archived)/;
                         s/\*\*Phase:\*\* In Progress/\*\*Phase:\*\* Complete (archived)/' "$filepath"

			FILES_TO_MOVE+=("$filepath")
			SPECIFIC_COUNT=$((SPECIFIC_COUNT + 1))
		else
			echo "  [DRY RUN] Would update status and move to $ARCHIVE_DIR/"
			SPECIFIC_COUNT=$((SPECIFIC_COUNT + 1))
		fi
	else
		echo "WARNING: File not found: $filepath"
	fi
	echo ""
done

echo "=== Step 2: Archive old completed investigations (Status:Complete + Next Step:None + >30 days) ==="
echo ""

AUTO_COUNT=0
for file in .kb/investigations/*.md; do
	if [ -f "$file" ]; then
		FILENAME=$(basename "$file")

		# Extract date from filename (YYYY-MM-DD prefix)
		FILE_DATE=$(echo "$FILENAME" | grep -oE '^[0-9]{4}-[0-9]{2}-[0-9]{2}')

		if [ -z "$FILE_DATE" ]; then
			continue
		fi

		# Skip February 2026 files unless they match ALL criteria
		FILE_MONTH=$(echo "$FILE_DATE" | cut -d'-' -f1,2)
		if [ "$FILE_MONTH" = "$FEB_2026" ]; then
			# Feb 2026 files require ALL criteria to be met
			# (handled below)
			:
		fi

		# Check if older than 30 days
		if [[ "$FILE_DATE" < "$CUTOFF_DATE" ]]; then
			# Check for Status: Complete AND Next Step: None
			HAS_COMPLETE=$(grep -q '^\*\*Status:\*\* Complete' "$file" && echo "yes" || echo "no")
			HAS_NEXT_NONE=$(grep -q '^\*\*Next Step:\*\* None' "$file" && echo "yes" || echo "no")

			if [ "$HAS_COMPLETE" = "yes" ] && [ "$HAS_NEXT_NONE" = "yes" ]; then
				# Extra check for Feb 2026: only archive if ALL criteria met
				if [ "$FILE_MONTH" = "$FEB_2026" ]; then
					echo "Feb 2026 file matches ALL criteria: $FILENAME (Date: $FILE_DATE)"
				else
					echo "Archiving: $FILENAME (Date: $FILE_DATE)"
				fi

				if [ "$DRY_RUN" = false ]; then
					FILES_TO_MOVE+=("$file")
					AUTO_COUNT=$((AUTO_COUNT + 1))
				else
					echo "  [DRY RUN] Would move to $ARCHIVE_DIR/"
					AUTO_COUNT=$((AUTO_COUNT + 1))
				fi
			fi
		fi
	fi
done

echo ""
echo "=== Summary ==="
echo "Specific files archived: $SPECIFIC_COUNT"
echo "Auto-archived (old + complete): $AUTO_COUNT"
echo "Total to archive: $((SPECIFIC_COUNT + AUTO_COUNT))"
echo ""

if [ "$DRY_RUN" = false ]; then
	# Batch move all files
	echo "Moving ${#FILES_TO_MOVE[@]} files to archive..."
	for file in "${FILES_TO_MOVE[@]}"; do
		mv "$file" "$ARCHIVE_DIR/"
	done

	# Single git add for all moved files
	git add .kb/investigations/ "$ARCHIVE_DIR/"
	echo "Git staging complete"
	echo ""

	# Count files after archival
	TOTAL_AFTER=$(ls -1 .kb/investigations/*.md 2>/dev/null | wc -l | tr -d ' ')
	ARCHIVED_AFTER=$(ls -1 "$ARCHIVE_DIR"/*.md 2>/dev/null | wc -l | tr -d ' ')

	echo "Files after archival:"
	echo "  Active: $TOTAL_AFTER (was $TOTAL_BEFORE)"
	echo "  Archived: $ARCHIVED_AFTER (was $ARCHIVED_BEFORE)"
	echo "  Moved: $((TOTAL_BEFORE - TOTAL_AFTER))"
	echo ""
	echo "✓ Archival complete"
else
	echo "[DRY RUN] No files were moved. Run without --dry-run to execute."
fi
