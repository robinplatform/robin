export const HOUR_MS = 60 * 60 * 1000;
export const DAY_MS = 24 * HOUR_MS;

// a+t(b−a)
export function lerp(a: number, b: number, t: number) {
	return a + t * (b - a);
}

// Array of length N with elements  0, 1, 2, ... N - 2, N - 1
export function arrayOfN(n: number): number[] {
	return [...Array.from(Array(n)).keys()];
}
