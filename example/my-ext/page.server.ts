import { onAppStart } from '@robinplatform/toolkit/daemon';
import * as fs from 'fs';
import _ from 'lodash';
import * as path from 'path';

onAppStart(async () => {
	console.log('Server started for my-ext');
});

export async function getSelfSource({
	filename,
}: {
	filename: string;
}): Promise<{ data: string }> {
	// Random lodash code to test modules
	_.each([1, 2, 3], () => {});

	return {
		data: await fs.promises.readFile(
			path.resolve(process.env.ROBIN_PROJECT_PATH ?? '', filename),
			'utf8',
		),
	};
}
