// a+t(b−a)
export function lerp(a: number, b: number, t: number) {
	return a + t * (b - a);
}
