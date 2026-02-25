#!/bin/bash
# Systematic investigation analysis script

# Extract metadata from each investigation
analyze_investigation() {
	local file="$1"
	local basename=$(basename "$file")
	local date=$(echo "$basename" | grep -oE "^[0-9]{4}-[0-9]{2}-[0-9]{2}")

	# Extract title/question
	local title=$(head -20 "$file" | grep -E "^(# |##|\*\*Question:)" | head -1 | sed 's/^[#* ]*//; s/\*\*//g; s/Question: //')

	# Extract status
	local status=$(grep -E "^\*\*Status:\*\*|^\*\*Phase:\*\*" "$file" | head -1 | sed 's/\*\*Status:\*\*//' | sed 's/\*\*Phase:\*\*//' | xargs)

	# Extract topic indicators from filename
	local topic=""
	if echo "$basename" | grep -q "inv-cli"; then topic="cli"; fi
	if echo "$basename" | grep -q "inv-opencode\|client-opencode"; then topic="opencode"; fi
	if echo "$basename" | grep -q "inv-daemon\|daemon-"; then topic="daemon"; fi
	if echo "$basename" | grep -q "spawn\|headless"; then topic="spawn"; fi
	if echo "$basename" | grep -q "design-\|architect"; then topic="design"; fi
	if echo "$basename" | grep -q "audit-"; then topic="audit"; fi
	if echo "$basename" | grep -q "research-"; then topic="research"; fi
	if echo "$basename" | grep -q "synthesis\|synthesize"; then topic="synthesis"; fi
	if echo "$basename" | grep -q "entropy"; then topic="entropy"; fi
	if echo "$basename" | grep -q "skill"; then topic="skill"; fi
	if echo "$basename" | grep -q "beads\|bd-"; then topic="beads"; fi
	if echo "$basename" | grep -q "dashboard\|web-ui"; then topic="dashboard"; fi
	if echo "$basename" | grep -q "verify\|verification\|gate"; then topic="verification"; fi
	if echo "$basename" | grep -q "coaching"; then topic="coaching"; fi
	if echo "$basename" | grep -q "model-\|probe-"; then topic="knowledge"; fi

	echo "$date|$topic|$status|$basename|$title"
}

export -f analyze_investigation

# Process all uncategorized investigations
cat /tmp/uncategorized_investigations.txt | while read file; do
	analyze_investigation "$file"
done | sort
