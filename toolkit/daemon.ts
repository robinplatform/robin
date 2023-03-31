import { Robin } from './internal/types';
import fetch from 'isomorphic-fetch';

export async function onAppStart(handler: () => Promise<void>) {
	if (Robin.isDaemonProcess) {
		Robin.startupHandlers.push(handler);
	}
}

export class Topic {
	private constructor() {}

	public static async createTopic() {}

	async publish() {}
}
