import { onAppStart } from '@robinplatform/toolkit/daemon';
import * as fs from 'fs';
import _ from 'lodash';

onAppStart(async () => {
	console.log('Server started for my-ext');
});

export async function getSelfSource({ filename }: { filename: string }) {
	// Random lodash code to test modules
	_.each([1, 2, 3], () => {});

	return fs.promises.readFile(filename, 'utf8');
}
