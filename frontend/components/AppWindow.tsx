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

function AppWindowContent({ id, setTitle }: AppWindowProps) {
	const router = useRouter();

	const iframeRef = React.useRef<HTMLIFrameElement | null>(null);
	const [error, setError] = React.useState<string | null>(null);
	const subRoute = React.useMemo(
		() =>
			router.isReady
				? router.asPath.substring('/app/'.length + id.length)
				: null,
		[router.isReady, router.asPath, id],
	);

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
									href={`/app/${id}/settings`}
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
						src={`http://localhost:9010/api/app-resources/${id}/base${subRoute}`}
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
