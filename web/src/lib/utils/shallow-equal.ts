/**
 * Performs a shallow equality check between two objects or values.
 * Returns true if all top-level properties are strictly equal.
 */
export function shallowEqual(a: any, b: any): boolean {
	// Same reference
	if (a === b) return true;

	// Handle null/undefined
	if (a == null || b == null) return false;

	// Handle non-objects (primitives)
	if (typeof a !== 'object' || typeof b !== 'object') {
		return a === b;
	}

	// Handle arrays
	if (Array.isArray(a) && Array.isArray(b)) {
		if (a.length !== b.length) return false;
		return a.every((item, index) => item === b[index]);
	}

	// Handle objects
	const keysA = Object.keys(a);
	const keysB = Object.keys(b);

	if (keysA.length !== keysB.length) return false;

	return keysA.every(key => a[key] === b[key]);
}
