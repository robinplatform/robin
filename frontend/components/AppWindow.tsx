import { useRouter } from 'next/router';
import React from 'react';
import { Alert } from './Alert';
import { Button } from './Button';
import { useIsMutating } from '@tanstack/react-query';
import { GearIcon, SyncIcon } from '@primer/octicons-react';
import toast from 'react-hot-toast';
import { useRpcMutation, useRpcQuery } from '../hooks/useRpcQuery';
import { z } from 'zod';
import cx from 'classnames';
import Link from 'next/link';
import { AppToolbar } from './AppToolbar';
import styles from './AppToolbar.module.scss';

type AppWindowProps = {
	id: string;
	setTitle: React.Dispatch<React.SetStateAction<string>>;
	route: string;
	setRoute: React.Dispatch<React.SetStateAction<string>>;
};

const RestartAppButton: React.FC = () => {
	const router = useRouter();
	const { mutate: restartApp, isLoading } = useRpcMutation({
		method: 'RestartApp',
		result: z.unknown(),

		onSuccess: () => {
			toast.success('Restarted app', { id: 'restart-app' });
		},
		onError: (err) => {
			toast.error(`Failed to restart app: ${(err as Error).message}`, {
				id: 'restart-app',
			});
		},
	});
	React.useEffect(() => {
		if (isLoading) {
			toast.loading('Restarting app', { id: 'restart-app' });
		}
	}, [isLoading]);

	return (
		<button
			type="button"
			disabled={isLoading}
			className={cx(styles.toolbarButton, 'robin-rounded robin-bg-dark-purple')}
			onClick={() =>
				restartApp({
					appId: router.query.id as string,
				})
			}
		>
			<SyncIcon />
		</button>
	);
};

function AppWindowContent({ id, setTitle, route, setRoute }: AppWindowProps) {
	const iframeRef = React.useRef<HTMLIFrameElement | null>(null);
	const [error, setError] = React.useState<string | null>(null);
	const mostCurrentRouteRef = React.useRef<string>(route);

	React.useEffect(() => {
		mostCurrentRouteRef.current = route;
	}, [route]);

	// NOTE: We don't want to re-create the iframe every time the route changes.
	React.useEffect(() => {
		if (!iframeRef.current) {
			return;
		}

		const target = `http://localhost:9010/api/app-resources/${id}/base${route}`;
		if (!iframeRef.current.src) {
			iframeRef.current.src = target;
		} else if (iframeRef.current.src !== target) {
			iframeRef.current.contentWindow?.postMessage({
				type: 'robinRouteChange',
				route: target,
			});
		}
	}, [route]);

	React.useEffect(() => {
		const onMessage = (message: MessageEvent) => {
			try {
				switch (message.data.type) {
					case 'locationUpdate': {
						const location = message.data.location;
						if (!location || typeof location !== 'string') {
							break;
						}

						console.log('locationUpdate', location);
						const url = new URL(location);
						const newRoute = url.pathname.substring(
							`/api/app-resources/${id}/base`.length,
						);

						const currentRoute = mostCurrentRouteRef.current;
						if (newRoute !== currentRoute) {
							console.log('calling setRoute', newRoute, currentRoute);
							setRoute(newRoute);
						}
						break;
					}

					case 'titleUpdate':
						setTitle((title) => String(message.data.title || title));
						break;

					case 'appError':
						setError(message.data.error);
						break;

					default:
						// toast.error(`Unknown app message type: ${message.data.type}`, {
						// 	id: 'unknown-message-type',
						// });
						console.warn(
							`Unknown app message type on message: ${JSON.stringify(
								message.data,
							)}`,
						);
				}
			} catch (e: any) {
				toast.error(
					`Error when receiving app message: ${String(e)}\ndata:\n${
						message.data
					}`,
					{
						id: 'unknown-message-type',
					},
				);
			}
		};

		window.addEventListener('message', onMessage);
		return () => window.removeEventListener('message', onMessage);
	}, [setTitle]);

	React.useEffect(() => {
		setTitle(id);

		if (!iframeRef.current) return;
		const iframe = iframeRef.current;

		const listener = () => {
			if (iframe.contentDocument) {
				setTitle(iframe.contentDocument.title);
			}
		};

		iframe.addEventListener('load', listener);
		return () => iframe.removeEventListener('load', listener);
	}, [id, setTitle]);

	return (
		<div className={'full col'}>
			{error && (
				<div style={{ width: '100%', padding: '1rem' }}>
					<Alert variant="error" title={`The '${id}' app has crashed.`}>
						<pre style={{ marginBottom: '1rem' }}>
							<code>{String(error)}</code>
						</pre>

						<Button variant="primary" onClick={() => setError(null)}>
							Reload app
						</Button>
					</Alert>
				</div>
			)}
			{!!id && !error && (
				<>
					<AppToolbar
						appId={id}
						actions={
							<>
								<RestartAppButton />

								<Link
									href={`/app-settings/${id}`}
									className={cx(
										styles.toolbarButton,
										'robin-rounded robin-bg-dark-purple',
									)}
									style={{ marginLeft: '.5rem' }}
								>
									<GearIcon />
								</Link>
							</>
						}
					/>

					<iframe
						ref={iframeRef}
						style={{ border: '0', flexGrow: 1, width: '100%', height: '100%' }}
					/>
				</>
			)}
		</div>
	);
}

export function AppWindow(props: AppWindowProps) {
	const numRestarts = useIsMutating({ mutationKey: ['RestartApp'] });

	return <AppWindowContent key={String(props.id) + numRestarts} {...props} />;
}
