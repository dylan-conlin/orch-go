/**
 * Shallow equality check for objects and arrays
 * Used to prevent unnecessary store updates when API returns identical data
 * 
 * @param a - First value
 * @param b - Second value
 * @returns true if values are shallowly equal, false otherwise
 */
export function shallowEqual<T>(a: T, b: T): boolean {
	// Same reference
	if (a === b) {
		return true;
	}

	// Different types or one is null/undefined
	if (typeof a !== typeof b || a == null || b == null) {
		return false;
	}

	// Arrays: compare length and elements
	if (Array.isArray(a) && Array.isArray(b)) {
		if (a.length !== b.length) {
			return false;
		}
		for (let i = 0; i < a.length; i++) {
			if (a[i] !== b[i]) {
				return false;
			}
		}
		return true;
	}

	// Objects: compare keys and values
	if (typeof a === 'object' && typeof b === 'object') {
		const keysA = Object.keys(a as object);
		const keysB = Object.keys(b as object);

		if (keysA.length !== keysB.length) {
			return false;
		}

		for (const key of keysA) {
			if (!Object.prototype.hasOwnProperty.call(b, key)) {
				return false;
			}
			if ((a as any)[key] !== (b as any)[key]) {
				return false;
			}
		}

		return true;
	}

	// Primitives (already handled by === check above, but for completeness)
	return false;
}
