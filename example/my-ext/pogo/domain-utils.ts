export const TypeColors: Record<string, string> = {
	steel: 'gray',
	ground: 'chocolate',
	dark: 'darkslateblue',
	fire: 'orange',
};

export const MegaRequirements = {
	1: 1,
	2: 7,
	3: 30,
} as const;

export const MegaWaitDays = {
	1: 7,
	2: 5,
	3: 3,
} as const;

export const MegaWaitTime = {
	1: 7 * 24 * 60 * 60 * 1000,
	2: 5 * 24 * 60 * 60 * 1000,
	3: 3 * 24 * 60 * 60 * 1000,
} as const;

export function megaLevelFromCount(count: number): 0 | 1 | 2 | 3 {
	switch (true) {
		case count >= 30:
			return 3;

		case count >= 7:
			return 2;

		case count >= 1:
			return 1;

		default:
			return 0;
	}
}

export function nextMegaDeadline(count: number, lastMega: Date): Date {
	const date = new Date(lastMega);
	const offset = MegaWaitTime[megaLevelFromCount(count)] ?? 0;
	date.setTime(date.getTime() + offset);

	return date;
}
