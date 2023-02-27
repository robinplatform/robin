import { useRouter } from 'next/router';
import React from 'react';
import { Alert } from './Alert';
import { Button } from './Button';
import { useIsMutating } from '@tanstack/react-query';
import styles from './AppWindow.module.scss';
import { GearIcon, SyncIcon, ToolsIcon } from '@primer/octicons-react';
import toast from 'react-hot-toast';
import { useRpcMutation, useRpcQuery } from '../hooks/useRpcQuery';
import { z } from 'zod';
import cx from 'classnames';
import Link from 'next/link';

type Props = {
	id: string | undefined;
	setTitle: React.Dispatch<React.SetStateAction<string>>;
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

function AppWindowContent({ id, setTitle }: Props) {
	const router = useRouter();

	const iframeRef = React.useRef<HTMLIFrameElement | null>(null);
	const [error, setError] = React.useState<string | null>(null);

	React.useEffect(() => {
		const onMessage = (message: MessageEvent) => {
			try {
				switch (message.data.type) {
					case 'locationUpdate': {
						const location = {
							pathname: window.location.pathname,
							search: new URL(message.data.location).search,
						};
						router.push(location, undefined, { shallow: true });
						break;
					}

					case 'titleUpdate':
						setTitle((title) => message.data.title || title);
						break;

					case 'appError':
						setError(message.data.error);
						break;
				}
			} catch {}
		};

		window.addEventListener('message', onMessage);
		return () => window.removeEventListener('message', onMessage);
	}, [router, setTitle]);

	React.useEffect(() => {
		if (!id) return;
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

	const { data: appConfig } = useRpcQuery({
		method: 'GetAppById',
		data: { appId: id },
		result: z.object({
			id: z.string(),
			name: z.string(),
			pageIcon: z.string(),
		}),
		onError: (err) => {
			toast.error(`Failed to load robin app config: ${(err as Error).message}`);
		},
	});

	return (
		<div className={'full col'}>
			{error && (
				<div style={{ width: '100%', padding: '1rem' }}>
					<Alert variant="error" title={`The '${id}' app has crashed.`}>
						<pre style={{ marginBottom: '1rem' }}>
							<code>{error}</code>
						</pre>

						<Button variant="primary" onClick={() => setError(null)}>
							Reload app
						</Button>
					</Alert>
				</div>
			)}
			{!!id && !error && (
				<>
					<div className={styles.toolbar}>
						<p>{appConfig?.name ?? 'Loading ...'}</p>

						<div className={styles.toolbarIcons}>
							<RestartAppButton />

							{/* TODO: This should be the app's settings, not the global settings */}
							<Link
								href="/settings"
								className={cx(
									styles.toolbarButton,
									'robin-rounded robin-bg-dark-purple',
								)}
								style={{ marginLeft: '.5rem' }}
							>
								<GearIcon />
							</Link>
						</div>
					</div>

					<iframe
						ref={iframeRef}
						src={`http://localhost:9010/api/app-resources/${id}/base.html`}
						style={{ border: '0', flexGrow: 1 }}
					/>
				</>
			)}
		</div>
	);
}

export function AppWindow(props: Props) {
	const numRestarts = useIsMutating({ mutationKey: ['RestartApp'] });

	return <AppWindowContent key={String(props.id) + numRestarts} {...props} />;
}
