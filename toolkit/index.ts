import fetch from 'isomorphic-fetch';
import { z } from 'zod';

const isBrowser = (function () {
	try {
		return global === window;
	} catch {
		return false;
	}
})();

const ROBIN_SERVER_PORT = Number(process.env.ROBIN_SERVER_PORT ?? 9010);
const ROBIN_APP_ID = process.env.ROBIN_APP_ID;

if (!ROBIN_APP_ID) {
	throw new Error('ROBIN_APP_ID must be set - was this app compiled by Robin?');
}

const baseUrl = isBrowser
	? `${window.location.protocol}//${window.location.host}`
	: `http://localhost:${ROBIN_SERVER_PORT ?? 9010}`;

export async function request<T>({
	pathname,
	resultType,
	body,
	...overrides
}: Omit<RequestInit, 'body'> & {
	pathname: string;
	resultType: z.ZodSchema<T>;
	body?: object;
}): Promise<T> {
	const targetUrl = new URL(pathname, baseUrl).toString();
	const res = await fetch(targetUrl, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
			...(overrides?.headers ?? {}),
		},
		redirect: 'follow',
		body: JSON.stringify(body ?? {}),
		...overrides,
	});

	const resBody = await res.json();
	if (typeof resBody === 'object' && resBody && resBody.type === 'error') {
		throw Object.assign(new Error(resBody.error), {
			...resBody,
			status: res.status,
		});
	}

	if (!res.ok) {
		throw new Error(`Request failed with status ${res.status}`);
	}

	return resultType.parse(resBody);
}

export async function getAppSettings<T extends Record<string, unknown>>(
	settingsShape: z.Schema<T>,
) {
	return request({
		pathname: `/api/apps/rpc/GetAppSettingsById`,
		resultType: settingsShape,
		body: {
			appId: ROBIN_APP_ID,
		},
	});
}
