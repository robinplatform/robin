import { Robin } from './internal/types';

export async function onAppStart(handler: () => Promise<void>) {
	if (Robin.isDaemonProcess) {
		Robin.startupHandlers.push(handler);
	}
}
