export const HOUR_MS = 60 * 60 * 1000;
export const DAY_MS = 24 * HOUR_MS;

// a+t(b−a)
export function lerp(a: number, b: number, t: number) {
	return a + t * (b - a);
}
