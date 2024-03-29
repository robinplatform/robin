import { z } from 'zod';
import { request } from '.';
import { Robin } from './internal/types';

export async function onAppStart(handler: () => Promise<void>) {
	if (Robin.isDaemonProcess) {
		Robin.startupHandlers.push(handler);
	}
}

export class Topic<T> {
	private constructor(
		private readonly category: string[],
		private readonly key: string,
	) {}

	// Creates a topic under the specified category and key, as a subcategory of
	// `/app-topics/{app}/`.
	public static async createTopic<T>(
		category: string[],
		key: string,
	): Promise<Topic<T>> {
		await request({
			pathname: '/api/apps/rpc/CreateTopic',
			resultType: z.object({}),
			body: {
				appId: process.env.ROBIN_APP_ID,
				category,
				key,
			},
		});

		return new Topic<T>(category, key);
	}

	async publish(t: T) {
		await request({
			pathname: '/api/apps/rpc/PublishToTopic',
			resultType: z.object({}),
			body: {
				appId: process.env.ROBIN_APP_ID,
				category: this.category,
				key: this.key,
				data: t,
			},
		});
	}
}
